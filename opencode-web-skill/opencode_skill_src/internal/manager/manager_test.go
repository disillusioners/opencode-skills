package manager

import (
	"testing"
	"time"

	"opencode_skill/internal/api"
)

func TestSessionManager_NewSessionManager_Defaults(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager("test-session", "/tmp", nil)

	if sm.SessionID != "test-session" {
		t.Errorf("Expected SessionID test-session, got %s", sm.SessionID)
	}
	if sm.State != StateIdle {
		t.Errorf("Expected State IDLE, got %s", sm.State)
	}
	if sm.params.LastAgent != "sisyphus" {
		t.Errorf("Expected LastAgent sisyphus, got %s", sm.params.LastAgent)
	}
	if sm.isAgentLocked != false {
		t.Errorf("Expected isAgentLocked false, got %v", sm.isAgentLocked)
	}
	if len(sm.Questions) != 0 {
		t.Errorf("Expected empty Questions, got %d", len(sm.Questions))
	}
}

func TestSessionManager_RestoreFromPersistedState(t *testing.T) {
	t.Parallel()

	persisted := &PersistedState{
		LastAgent:      "atlas",
		IsAgentLocked:  true,
		State:          "BUSY",
		LatestResponse: `{"result": "test"}`,
		Questions:      `[{"id": "q1", "sessionID": "s1", "questions": []}]`,
		LastActivity:   "2026-02-16T14:00:00Z",
	}

	sm := NewSessionManager("test-session", "/tmp", persisted)

	if sm.params.LastAgent != "atlas" {
		t.Errorf("Expected LastAgent atlas, got %s", sm.params.LastAgent)
	}
	if !sm.isAgentLocked {
		t.Errorf("Expected isAgentLocked true, got false")
	}
	if sm.State != StateBusy {
		t.Errorf("Expected State BUSY, got %s", sm.State)
	}
	if sm.LatestResponse == nil {
		t.Errorf("Expected LatestResponse to be set, got nil")
	}
	if len(sm.Questions) != 1 {
		t.Errorf("Expected 1 Question, got %d", len(sm.Questions))
	}
	if sm.Questions[0].ID != "q1" {
		t.Errorf("Expected Question ID q1, got %s", sm.Questions[0].ID)
	}
}

func TestSessionManager_RestoreFromPersistedState_EmptyQuestions(t *testing.T) {
	t.Parallel()

	persisted := &PersistedState{
		LastAgent: "sisyphus",
		State:     "IDLE",
		Questions: "[]",
	}

	sm := NewSessionManager("test-session", "/tmp", persisted)

	if len(sm.Questions) != 0 {
		t.Errorf("Expected empty Questions, got %d", len(sm.Questions))
	}
}

func TestSessionManager_RestoreFromPersistedState_InvalidJSON(t *testing.T) {
	t.Parallel()

	persisted := &PersistedState{
		LastAgent:      "sisyphus",
		State:          "IDLE",
		Questions:      `invalid json`,
		LatestResponse: `also invalid`,
	}

	sm := NewSessionManager("test-session", "/tmp", persisted)

	if len(sm.Questions) != 0 {
		t.Errorf("Expected empty Questions on invalid JSON, got %d", len(sm.Questions))
	}
	if sm.LatestResponse != nil {
		t.Errorf("Expected nil LatestResponse on invalid JSON, got %v", sm.LatestResponse)
	}
}

func TestSessionManager_SaveState(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager("test-session", "/tmp", nil)
	sm.params.LastAgent = "atlas"
	sm.isAgentLocked = true
	sm.State = StateBusy
	sm.Questions = []api.Question{{ID: "q1", SessionID: "test-session"}}
	sm.LatestResponse = map[string]interface{}{"result": "success"}
	sm.lastActivity = time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC)

	state := sm.SaveState()

	if state.LastAgent != "atlas" {
		t.Errorf("Expected LastAgent atlas, got %s", state.LastAgent)
	}
	if !state.IsAgentLocked {
		t.Errorf("Expected IsAgentLocked true, got false")
	}
	if state.State != "BUSY" {
		t.Errorf("Expected State BUSY, got %s", state.State)
	}
	if state.Questions == "" || state.Questions == "[]" {
		t.Errorf("Expected Questions JSON, got %s", state.Questions)
	}
	if state.LatestResponse == "" {
		t.Errorf("Expected LatestResponse JSON, got empty")
	}
	if state.LastActivity != "2026-02-16T14:00:00Z" {
		t.Errorf("Expected LastActivity 2026-02-16T14:00:00Z, got %s", state.LastActivity)
	}
}

func TestSessionManager_SaveState_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &PersistedState{
		LastAgent:      "prometheus",
		IsAgentLocked:  true,
		State:          "WAITING_FOR_INPUT",
		Questions:      `[{"id":"q1","sessionID":"s1","questions":[]}]`,
		LatestResponse: `{"result":"ok"}`,
		LastActivity:   "2026-02-16T10:30:00Z",
	}

	sm1 := NewSessionManager("test-session", "/tmp", original)
	saved := sm1.SaveState()

	sm2 := NewSessionManager("test-session", "/tmp", &saved)

	if sm2.params.LastAgent != sm1.params.LastAgent {
		t.Errorf("LastAgent mismatch: %s vs %s", sm2.params.LastAgent, sm1.params.LastAgent)
	}
	if sm2.isAgentLocked != sm1.isAgentLocked {
		t.Errorf("isAgentLocked mismatch: %v vs %v", sm2.isAgentLocked, sm1.isAgentLocked)
	}
	if sm2.State != sm1.State {
		t.Errorf("State mismatch: %s vs %s", sm2.State, sm1.State)
	}
}

func TestSessionManager_SetLastAgent(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager("test-session", "/tmp", nil)
	sm.SetLastAgent("atlas")

	if sm.params.LastAgent != "atlas" {
		t.Errorf("Expected LastAgent atlas, got %s", sm.params.LastAgent)
	}
}

func TestSessionManager_SetAgentLocked(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager("test-session", "/tmp", nil)
	sm.SetAgentLocked(true)

	if !sm.isAgentLocked {
		t.Errorf("Expected isAgentLocked true, got false")
	}

	sm.SetAgentLocked(false)
	if sm.isAgentLocked {
		t.Errorf("Expected isAgentLocked false, got true")
	}
}
