package manager

import (
	"opencode_skill/internal/api"
)

// AbortTask resets the session state to IDLE and stops waiting for any ongoing worker result.
func (sm *SessionManager) AbortTask() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.aborted = true
	sm.State = StateIdle
	sm.LatestResponse = map[string]interface{}{"status": "aborted", "message": "Task aborted by user"}
	sm.isWorkerBusy = false
	sm.Questions = []api.Question{}
}
