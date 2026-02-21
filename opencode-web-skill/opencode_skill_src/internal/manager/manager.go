package manager

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"opencode_skill/internal/api"
	"opencode_skill/internal/config"
	"opencode_skill/internal/types"
)

type State string

const (
	StateIdle            State = "IDLE"
	StateBusy            State = "BUSY"
	StateWaitingForInput State = "WAITING_FOR_INPUT"
)

type PersistedState struct {
	LastAgent      string
	IsAgentLocked  bool
	State          string
	LatestResponse string
	Questions      string
	LastActivity   string
}

type SessionManager struct {
	SessionID      string
	State          State
	LatestResponse interface{}
	Questions      []api.Question

	mu            sync.RWMutex // Protects State, LatestResponse, Questions, isWorkerBusy
	inputChan     chan Request
	stopChan      chan struct{}
	client        *api.Client
	isAgentLocked bool

	// Worker tracking
	workerDoneChan chan workerResult
	isWorkerBusy   bool
	taskStartTime  time.Time
	lastActivity   time.Time
	params         SessionParams
	aborted        bool
	OnStateChange  func(PersistedState)
}

type SessionParams struct {
	LastAgent string
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

func NewSessionManager(sessionID string, workingDir string, persistedState *PersistedState) *SessionManager {
	sm := &SessionManager{
		SessionID:      sessionID,
		State:          StateIdle,
		inputChan:      make(chan Request, 10),
		stopChan:       make(chan struct{}),
		workerDoneChan: make(chan workerResult, 1),
		client:         api.NewClient(workingDir),
		lastActivity:   time.Now(),
		params:         SessionParams{LastAgent: "sisyphus"},
	}

	if persistedState != nil {
		sm.restoreFromPersistedState(persistedState)
	}

	return sm
}

func (sm *SessionManager) restoreFromPersistedState(data *PersistedState) {
	if data.LastAgent != "" {
		sm.params.LastAgent = data.LastAgent
	}
	if data.State != "" {
		sm.State = State(data.State)
	}
	sm.isAgentLocked = data.IsAgentLocked
	if data.Questions != "" && data.Questions != "[]" {
		var questions []api.Question
		if err := json.Unmarshal([]byte(data.Questions), &questions); err == nil {
			sm.Questions = questions
		}
	}
	if data.LatestResponse != "" {
		var response interface{}
		if err := json.Unmarshal([]byte(data.LatestResponse), &response); err == nil {
			sm.LatestResponse = response
		}
	}
	if data.LastActivity != "" {
		if t, err := time.Parse(time.RFC3339, data.LastActivity); err == nil {
			sm.lastActivity = t
		}
	}
}

func (sm *SessionManager) saveStateLocked() PersistedState {
	questionsJSON, _ := json.Marshal(sm.Questions)
	responseJSON, _ := json.Marshal(sm.LatestResponse)

	return PersistedState{
		LastAgent:      sm.params.LastAgent,
		IsAgentLocked:  sm.isAgentLocked,
		State:          string(sm.State),
		LatestResponse: string(responseJSON),
		Questions:      string(questionsJSON),
		LastActivity:   sm.lastActivity.Format(time.RFC3339),
	}
}

func (sm *SessionManager) SaveState() PersistedState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.saveStateLocked()
}

func (sm *SessionManager) SetLastAgent(agent string) {
	sm.mu.Lock()
	sm.params.LastAgent = agent
	if sm.OnStateChange != nil {
		stateToSave := sm.saveStateLocked()
		sm.mu.Unlock()
		sm.OnStateChange(stateToSave)
	} else {
		sm.mu.Unlock()
	}
}

func (sm *SessionManager) SetAgentLocked(locked bool) {
	sm.mu.Lock()
	sm.isAgentLocked = locked
	if sm.OnStateChange != nil {
		stateToSave := sm.saveStateLocked()
		sm.mu.Unlock()
		sm.OnStateChange(stateToSave)
	} else {
		sm.mu.Unlock()
	}
}

func (sm *SessionManager) Start() {
	go sm.loop()
}

func (sm *SessionManager) UpdateWorkingDir(workingDir string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.client = api.NewClient(workingDir)
}

func (sm *SessionManager) Stop() {
	close(sm.stopChan)
}

func (sm *SessionManager) SubmitRequest(req Request) {
	log.Printf("SubmitRequest: acquiring lock for %s", req.Type)
	// Pre-set state to avoid race condition where GetSnapshot sees IDLE before loop picks up request
	sm.mu.Lock()
	log.Printf("SubmitRequest: lock acquired for %s", req.Type)
	if req.Type == "PROMPT" || req.Type == "COMMAND" {
		sm.State = StateBusy
		sm.LatestResponse = nil
		sm.isWorkerBusy = true // Optimistic lock
		log.Printf("SubmitRequest: OnStateChange is nil: %v", sm.OnStateChange == nil)
		if sm.OnStateChange != nil {
			log.Printf("SubmitRequest: calling OnStateChange")
			stateToSave := sm.saveStateLocked()
			sm.mu.Unlock() // avoid deadlock if OnStateChange blocks
			sm.OnStateChange(stateToSave)
			sm.mu.Lock()
			log.Printf("SubmitRequest: OnStateChange done")
		}
	}
	sm.mu.Unlock()
	log.Printf("SubmitRequest: lock released, sending to channel")
	sm.inputChan <- req
	log.Printf("SubmitRequest: sent to channel successfully")
}

func (sm *SessionManager) GetSnapshot() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

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
		sm.mu.Lock()

		if req.Type == "PROMPT" {
			if p, ok := req.Payload.(types.PromptRequest); ok {
				sm.params.LastAgent = p.Agent
			}
		} else if req.Type == "COMMAND" {
			if p, ok := req.Payload.(types.CommandRequest); ok {
				sm.params.LastAgent = p.Agent
			}
		}

		sm.State = StateBusy
		sm.LatestResponse = nil
		sm.taskStartTime = time.Now()
		sm.isWorkerBusy = true
		sm.mu.Unlock()

		log.Printf("Starting worker for PROMPT/COMMAND...")
		go sm.runWorker(req)

	case "ANSWER":
		payload, ok := req.Payload.(types.AnswerRequest)
		if ok {
			if err := sm.client.AnswerQuestion(payload); err != nil {
				log.Printf("Answer failed: %v", err)
			} else {
				sm.mu.Lock()
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
				sm.mu.Unlock()
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

	// Read client with lock if needed, but client itself is thread-safe (just struct with static fields)
	// sm.client pointer exchange needs lock.
	sm.mu.RLock()
	client := sm.client
	sm.mu.RUnlock()

	if req.Type == "COMMAND" {
		cmdReq, _ := req.Payload.(types.CommandRequest)
		res, err = client.SendCommand(sm.SessionID, cmdReq)
	} else {
		promptReq, _ := req.Payload.(types.PromptRequest)
		res, err = client.SendPrompt(sm.SessionID, promptReq)
	}

	sm.workerDoneChan <- workerResult{Result: res, Error: err}
}

func (sm *SessionManager) handleWorkerDone(res workerResult) {
	sm.mu.Lock()

	sm.isWorkerBusy = false

	if sm.aborted {
		sm.aborted = false
		sm.mu.Unlock()
		return
	}

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

	if sm.OnStateChange != nil {
		stateToSave := sm.saveStateLocked()
		sm.mu.Unlock()
		sm.OnStateChange(stateToSave)
	} else {
		sm.mu.Unlock()
	}
}

func (sm *SessionManager) pollQuestions() {
	sm.mu.RLock()
	client := sm.client
	sm.mu.RUnlock()

	questions, err := client.GetQuestions()
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

	sm.mu.Lock()
	defer sm.mu.Unlock()

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
	sm.mu.RLock()
	if len(sm.Questions) > 0 {
		sm.mu.RUnlock()
		return
	}

	if sm.State == StateBusy && sm.isWorkerBusy {
		if time.Since(sm.taskStartTime) > config.AutoFixTimeout {
			sm.mu.RUnlock()
			log.Printf("Session %s exceeded timeout. Triggering Auto-Fix.", sm.SessionID)
			go func() {
				sm.inputChan <- Request{Type: "FIX"}
			}()
			return
		}
	}
	sm.mu.RUnlock()
}

func (sm *SessionManager) performFix() {
	log.Printf("Performing FIX for session %s...", sm.SessionID)

	sm.mu.RLock()
	client := sm.client
	sm.mu.RUnlock()

	// 1. Abort
	_ = client.AbortSession(sm.SessionID)

	// Wait
	time.Sleep(3 * time.Second)

	sm.mu.Lock()
	// 3. Send Continue
	sm.isWorkerBusy = true
	sm.State = StateBusy
	sm.taskStartTime = time.Now()
	sm.LatestResponse = nil
	sm.mu.Unlock()

	req := types.PromptRequest{
		Agent: sm.params.LastAgent,
		Model: types.ModelDetails{ProviderID: "zai-coding-plan", ModelID: "glm-5"},
		Parts: []types.Part{{Type: "text", Text: "continue"}},
	}

	go func() {
		res, err := client.SendPrompt(sm.SessionID, req)
		sm.workerDoneChan <- workerResult{Result: res, Error: err}
	}()
}
