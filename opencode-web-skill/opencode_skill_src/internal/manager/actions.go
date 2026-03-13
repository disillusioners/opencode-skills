package manager

import (
	"opencode_skill/internal/api"
	"time"
)

// AbortTask resets the session state to IDLE and stops waiting for any ongoing worker result.
func (sm *SessionManager) AbortTask() {
	sm.mu.Lock()

	sm.aborted = true
	sm.State = StateIdle
	sm.isWorkerBusy = false
	sm.taskStartTime = time.Time{}
	sm.Questions = []api.Question{}

	// Increment ResultID and include in response
	sm.ResultID++
	sm.LatestResponse = map[string]interface{}{
		"status":    "aborted",
		"message":   "Task aborted by user",
		"result_id": sm.ResultID,
	}

	// Persist state if callback is set
	if sm.OnStateChange != nil {
		stateToSave := sm.saveStateLocked()
		sm.mu.Unlock()
		sm.OnStateChange(stateToSave)
	} else {
		sm.mu.Unlock()
	}
}
