package daemon

import (
	"log"
	"opencode_skill/internal/manager"
)

func (s *Server) setupStatePersistence(sm *manager.SessionManager) {
	sm.OnStateChange = func(state manager.PersistedState) {
		sessionData, err := s.registry.FindByID(sm.SessionID)
		if err != nil {
			log.Printf("Failed to find session %s for persistence: %v", sm.SessionID, err)
			return
		}

		sessionData.LastAgent = state.LastAgent
		sessionData.IsAgentLocked = state.IsAgentLocked
		sessionData.State = state.State
		sessionData.LatestResponse = state.LatestResponse
		sessionData.Questions = state.Questions
		sessionData.LastActivity = state.LastActivity

		if err := s.registry.UpdateSessionData(sessionData.Project, sessionData.SessionName, *sessionData); err != nil {
			log.Printf("Failed to persist state for session %s: %v", sm.SessionID, err)
		}
	}
}
