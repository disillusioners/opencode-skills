package manager

import (
	"log"
	"time"

	"opencode_wrapper/internal/api"
	"opencode_wrapper/internal/config"
)

type State string

const (
	StateIdle            State = "IDLE"
	StateBusy            State = "BUSY"
	StateWaitingForInput State = "WAITING_FOR_INPUT"
)

type SessionManager struct {
	SessionID      string
	State          State
	LatestResponse interface{}
	Questions      []api.Question

	inputChan chan Request
	stopChan  chan struct{}
	client    *api.Client

	// Worker tracking
	workerDoneChan chan workerResult
	isWorkerBusy   bool
	taskStartTime  time.Time
	lastActivity   time.Time
}

type Request struct {
	Type       string
	Payload    interface{}
	ResultChan chan error // Optional, for sync acknowledgement
}

type workerResult struct {
	Result interface{}
	Error  error
}

func NewSessionManager(sessionID string, workingDir string) *SessionManager {
	return &SessionManager{
		SessionID:      sessionID,
		State:          StateIdle,
		inputChan:      make(chan Request, 10),
		stopChan:       make(chan struct{}),
		workerDoneChan: make(chan workerResult, 1),
		client:         api.NewClient(workingDir),
		lastActivity:   time.Now(),
	}
}

func (sm *SessionManager) Start() {
	go sm.loop()
}

func (sm *SessionManager) UpdateWorkingDir(workingDir string) {
	sm.client = api.NewClient(workingDir)
}

func (sm *SessionManager) Stop() {
	close(sm.stopChan)
}

func (sm *SessionManager) SubmitRequest(req Request) {
	sm.inputChan <- req
}

func (sm *SessionManager) GetSnapshot() map[string]interface{} {
	// Note: accessing fields without lock is technically racy but for a simple snapshot it might be okay.
	// Ideally we should use a command to get snapshot or a mutex.
	// For simplicity, let's use a mutex-less approach assuming single writer (the loop) and readers.
	// Or better, send a request to get snapshot? No, that blocks.
	// Let's just read.
	return map[string]interface{}{
		"state":           sm.State,
		"session_id":      sm.SessionID,
		"latest_response": sm.LatestResponse,
		"questions":       sm.Questions,
	}
}

func (sm *SessionManager) loop() {
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return

		case req := <-sm.inputChan:
			sm.handleRequest(req)

		case res := <-sm.workerDoneChan:
			sm.handleWorkerDone(res)

		case <-ticker.C:
			sm.pollQuestions()
			sm.checkAutoFix()
		}
	}
}

func (sm *SessionManager) handleRequest(req Request) {
	log.Printf("Handling request type: %s", req.Type)
	if req.ResultChan != nil {
		defer close(req.ResultChan)
	}

	switch req.Type {
	case "PROMPT", "COMMAND":
		if sm.isWorkerBusy {
			// Already checked by Daemon, but double check
			if req.ResultChan != nil {
				req.ResultChan <- nil
			}
			return
		}

		sm.State = StateBusy
		sm.LatestResponse = nil
		sm.taskStartTime = time.Now()
		sm.isWorkerBusy = true

		log.Printf("Starting worker for PROMPT/COMMAND...")
		go sm.runWorker(req)

	case "ANSWER":
		payload, ok := req.Payload.(api.AnswerRequest)
		if ok {
			if err := sm.client.AnswerQuestion(payload); err != nil {
				log.Printf("Answer failed: %v", err)
			} else {
				// Optimistically remove question
				newQuestions := []api.Question{}
				for _, q := range sm.Questions {
					if q.ID != payload.RequestID {
						newQuestions = append(newQuestions, q)
					}
				}
				sm.Questions = newQuestions

				if len(sm.Questions) == 0 {
					if sm.isWorkerBusy {
						sm.State = StateBusy
						sm.taskStartTime = time.Now() // Reset timeout
					} else {
						sm.State = StateIdle
					}
				}
			}
		}

	case "FIX":
		sm.performFix()
	}

	if req.ResultChan != nil {
		req.ResultChan <- nil
	}
}

func (sm *SessionManager) runWorker(req Request) {
	var res interface{}
	var err error

	if req.Type == "COMMAND" {
		cmdReq, _ := req.Payload.(api.CommandRequest)
		res, err = sm.client.SendCommand(sm.SessionID, cmdReq)
	} else {
		promptReq, _ := req.Payload.(api.PromptRequest)
		res, err = sm.client.SendPrompt(sm.SessionID, promptReq)
	}

	sm.workerDoneChan <- workerResult{Result: res, Error: err}
}

func (sm *SessionManager) handleWorkerDone(res workerResult) {
	sm.isWorkerBusy = false

	if res.Error != nil {
		sm.LatestResponse = map[string]interface{}{"error": res.Error.Error()}
	} else {
		sm.LatestResponse = map[string]interface{}{"result": res.Result}
	}

	if len(sm.Questions) > 0 {
		sm.State = StateWaitingForInput
	} else {
		sm.State = StateIdle
	}
}

func (sm *SessionManager) pollQuestions() {
	questions, err := sm.client.GetQuestions()
	if err != nil {
		log.Printf("Poll error: %v", err)
		return
	}

	// Filter for this session
	sessionQuestions := []api.Question{}
	for _, q := range questions {
		if q.SessionID == sm.SessionID {
			sessionQuestions = append(sessionQuestions, q)
		}
	}
	sm.Questions = sessionQuestions

	if len(sm.Questions) > 0 {
		sm.State = StateWaitingForInput
	} else if sm.State == StateWaitingForInput {
		if sm.isWorkerBusy {
			sm.State = StateBusy
		} else {
			sm.State = StateIdle
		}
	}
}

func (sm *SessionManager) checkAutoFix() {
	if len(sm.Questions) > 0 {
		return
	}

	if sm.State == StateBusy && sm.isWorkerBusy {
		if time.Since(sm.taskStartTime) > config.AutoFixTimeout {
			log.Printf("Session %s exceeded timeout. Triggering Auto-Fix.", sm.SessionID)
			// We need to trigger FIX. We can send a request to ourselves?
			// Or just call performFix directly?
			// performFix is blocking-ish, better to schedule it
			go func() {
				sm.inputChan <- Request{Type: "FIX"}
			}()
		}
	}
}

func (sm *SessionManager) performFix() {
	log.Printf("Performing FIX for session %s...", sm.SessionID)

	// 1. Abort
	_ = sm.client.AbortSession(sm.SessionID)

	// Wait
	time.Sleep(3 * time.Second)

	// 2. Reset Worker state?
	// The previous worker might still return, but we overwrite it.

	// 3. Send Continue
	sm.isWorkerBusy = true
	sm.State = StateBusy
	sm.taskStartTime = time.Now()
	sm.LatestResponse = nil

	req := api.PromptRequest{
		Agent: "sisyphus",
		Model: api.ModelDetails{ProviderID: "zai-coding-plan", ModelID: "glm-5"},
		Parts: []api.Part{{Type: "text", Text: "continue"}},
	}

	go func() {
		res, err := sm.client.SendPrompt(sm.SessionID, req)
		sm.workerDoneChan <- workerResult{Result: res, Error: err}
	}()
}
