package manager

import (
	"encoding/json"
	"log"
	"net"
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

	lastActivity  time.Time
	params        SessionParams
	aborted       bool
	OnStateChange func(PersistedState)
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
	sm.mu.Unlock()
}

// SyncStateWithOpenCode always verifies the local state against OpenCode's real status,
// even when local state is IDLE. This prevents stale IDLE state from being reported
// when OpenCode is actually busy with a task. Returns the (potentially updated) snapshot.
func (sm *SessionManager) SyncStateWithOpenCode() map[string]interface{} {
	// Always sync with OpenCode for accurate status reporting, even when IDLE.
	// This prevents stale IDLE state from being reported when OpenCode is busy.
	statusMap, err := sm.client.GetSessionStatus()
	if err != nil {
		log.Printf("SyncStateWithOpenCode: failed to get status from OpenCode: %v", err)
		return sm.GetSnapshot()
	}

	realStatus, exists := statusMap[sm.SessionID]
	if !exists {
		// Session not found in OpenCode - keep IDLE with error response if needed
		return sm.GetSnapshot()
	}

	// If OpenCode reports busy, update local state to match
	if realStatus.Type == "busy" {
		sm.mu.Lock()
		if sm.State != StateBusy {
			log.Printf("SyncStateWithOpenCode: local IDLE but OpenCode BUSY, updating state")
			sm.State = StateBusy
			sm.isWorkerBusy = true // Allow worker to handle the response
			if sm.OnStateChange != nil {
				stateToSave := sm.saveStateLocked()
				sm.mu.Unlock()
				sm.OnStateChange(stateToSave)
			} else {
				sm.mu.Unlock()
			}
		} else {
			sm.mu.Unlock()
		}
		return sm.GetSnapshot()
	}

	// OpenCode reports idle - validate we have a complete last message
	if realStatus.Type == "idle" {
		var lastMessage interface{}
		var messageComplete bool
		messages, err := sm.client.GetSessionMessages(sm.SessionID)
		if err != nil {
			log.Printf("SyncStateWithOpenCode: failed to get messages: %v", err)
		} else if len(messages) > 0 {
			lastMessage = messages[len(messages)-1]
			// Guard: Check for step-finish with reason=stop to verify clean completion
			// This protects against worker crashes, hangs, or abnormal termination
			if msgMap, ok := lastMessage.(map[string]interface{}); ok {
				if parts, ok := msgMap["parts"].([]interface{}); ok {
					for _, part := range parts {
						if partMap, ok := part.(map[string]interface{}); ok {
							if partType, ok := partMap["type"].(string); ok && partType == "step-finish" {
								if reason, ok := partMap["reason"].(string); ok && reason == "stop" {
									messageComplete = true
									break
								}
							}
						}
					}
				}
			}
			if !messageComplete {
				log.Printf("SyncStateWithOpenCode: OpenCode idle but message lacks step-finish guard (reason=stop) - possible abnormal termination")
			}
		}

		// If OpenCode reports idle, set State=IDLE since it's no longer processing.
		// The step-finish guard above is for logging/detection purposes only.
		// Always return the message as LatestResponse regardless of clean finish.
		shouldUpdateState := sm.State != StateIdle

		sm.mu.Lock()
		sm.LatestResponse = map[string]interface{}{"result": lastMessage}
		if shouldUpdateState {
			sm.State = StateIdle
			sm.isWorkerBusy = false
		}
		if sm.OnStateChange != nil {
			stateToSave := sm.saveStateLocked()
			sm.mu.Unlock()
			sm.OnStateChange(stateToSave)
		} else {
			sm.mu.Unlock()
		}
		_ = messageComplete // Suppress unused variable warning (kept for clarity)
	}

	return sm.GetSnapshot()
}

// getMessageFinish extracts the finish reason from a step-finish part.
func getMessageFinish(msg interface{}) string {
	if msgMap, ok := msg.(map[string]interface{}); ok {
		if parts, ok := msgMap["parts"].([]interface{}); ok {
			for _, part := range parts {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partType, ok := partMap["type"].(string); ok && partType == "step-finish" {
						if reason, ok := partMap["reason"].(string); ok {
							return reason
						}
					}
				}
			}
		}
	}
	return "<unknown>"
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
					} else {
						sm.State = StateIdle
					}
				}
				sm.mu.Unlock()
			}
		}

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

	var needAbort bool

	if res.Error != nil {
		if netErr, ok := res.Error.(net.Error); ok && netErr.Timeout() {
			// HTTP timeout after 1 hour: terminate OpenCode session to clean up resources.
			// Worker is done (terminated), so set State=IDLE with timeout error.
			needAbort = true
			sm.LatestResponse = map[string]interface{}{"error": "timeout after 1 hour"}
		} else {
			// Real error - set IDLE with error
			sm.LatestResponse = map[string]interface{}{"error": res.Error.Error()}
		}
	} else {
		// Success - set IDLE with result
		sm.LatestResponse = map[string]interface{}{"result": res.Result}
	}

	if len(sm.Questions) > 0 {
		sm.State = StateWaitingForInput
	} else {
		sm.State = StateIdle
	}

	// Copy client reference before releasing lock, then abort outside critical section
	client := sm.client
	sm.mu.Unlock()

	if needAbort {
		// Terminate OpenCode session to clean up resources
		if err := client.AbortSession(sm.SessionID); err != nil {
			log.Printf("Failed to abort session after timeout: %v", err)
		}
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
