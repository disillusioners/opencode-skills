package daemon

import (
	"log"
	"opencode_skill/internal/manager"
)

func (s *Server) setupStatePersistence(sm *manager.SessionManager) {
	log.Printf("Setting up state persistence for session %s", sm.SessionID)
	sm.OnStateChange = func(state manager.PersistedState) {
		log.Printf("OnStateChange ENTERED for session %s", sm.SessionID)
		log.Printf("OnStateChange: finding session %s", sm.SessionID)
		sessionData, err := s.registry.FindByID(sm.SessionID)
		if err != nil {
			log.Printf("Failed to find session %s for persistence: %v", sm.SessionID, err)
			return
		}
		log.Printf("OnStateChange: found session, updating")

		sessionData.LastAgent = state.LastAgent
		sessionData.IsAgentLocked = state.IsAgentLocked
		sessionData.State = state.State
		sessionData.LatestResponse = state.LatestResponse
		sessionData.Questions = state.Questions
		sessionData.LastActivity = state.LastActivity

		log.Printf("OnStateChange: calling UpdateSessionData")
		if err := s.registry.UpdateSessionData(sessionData.Project, sessionData.SessionName, *sessionData); err != nil {
			log.Printf("Failed to persist state for session %s: %v", sm.SessionID, err)
		}
		log.Printf("OnStateChange: done")
	}
}
