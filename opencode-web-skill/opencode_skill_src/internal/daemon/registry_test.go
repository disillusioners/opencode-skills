package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_CreateAndGet(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("myproject", "main", "session-123", "/home/user/project")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session, err := registry.Get("myproject", "main")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.Project != "myproject" {
		t.Errorf("Expected project myproject, got %s", session.Project)
	}
	if session.SessionName != "main" {
		t.Errorf("Expected session name main, got %s", session.SessionName)
	}
	if session.ID != "session-123" {
		t.Errorf("Expected ID session-123, got %s", session.ID)
	}
	if session.WorkingDir != "/home/user/project" {
		t.Errorf("Expected working dir /home/user/project, got %s", session.WorkingDir)
	}
	if session.LastAgent != "" {
		t.Errorf("Expected empty last_agent, got %s", session.LastAgent)
	}
	if session.IsAgentLocked != false {
		t.Errorf("Expected IsAgentLocked false, got %v", session.IsAgentLocked)
	}
	if session.State != "IDLE" {
		t.Errorf("Expected state IDLE, got %s", session.State)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	_, err = registry.Get("nonexistent", "project")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_Create_Duplicate(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create first session
	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Try to create duplicate
	err = registry.Create("project", "session", "id-2", "/dir2")
	if err != ErrDuplicate {
		t.Errorf("Expected ErrDuplicate, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create multiple sessions
	sessions := []struct {
		project     string
		sessionName string
		id          string
		workingDir  string
	}{
		{"proj1", "main", "id1", "/dir1"},
		{"proj1", "dev", "id2", "/dir2"},
		{"proj2", "test", "id3", "/dir3"},
	}

	for _, s := range sessions {
		err := registry.Create(s.project, s.sessionName, s.id, s.workingDir)
		if err != nil {
			t.Fatalf("Create failed for %s/%s: %v", s.project, s.sessionName, err)
		}
	}

	// List all sessions
	all, err := registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(all))
	}
}

func TestRegistry_Delete(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create a session
	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete it
	err = registry.Delete("project", "session")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = registry.Get("project", "session")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

func TestRegistry_Delete_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Delete("nonexistent", "project")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create initial session
	err = registry.Create("project", "main", "id1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Simulate concurrent reads
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = registry.Get("project", "main")
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRegistry_EmptyList(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	all, err := registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected empty list, got %d sessions", len(all))
	}
}

func TestRegistry_SameProjectDifferentSessions(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create multiple sessions in same project
	err = registry.Create("project", "session1", "id1", "/dir1")
	if err != nil {
		t.Fatalf("Create session1 failed: %v", err)
	}
	err = registry.Create("project", "session2", "id2", "/dir2")
	if err != nil {
		t.Fatalf("Create session2 failed: %v", err)
	}

	// List should return both
	all, err := registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(all))
	}
}

func TestRegistry_InvalidDBPath(t *testing.T) {
	// This test intentionally does not run in parallel
	// because it tests file permission issues

	// Try to create registry with invalid path
	_, err := NewRegistry("/nonexistent/path/that/cannot/be/created/test.db")
	if err == nil {
		t.Error("Expected error for invalid DB path, got nil")
	}
}

func TestRegistry_MultipleProjects(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	// Create sessions in different projects
	err = registry.Create("proj1", "main", "id1", "/dir1")
	if err != nil {
		t.Fatalf("Create proj1 failed: %v", err)
	}
	err = registry.Create("proj2", "main", "id2", "/dir2")
	if err != nil {
		t.Fatalf("Create proj2 failed: %v", err)
	}

	// Get should work independently
	s1, err := registry.Get("proj1", "main")
	if err != nil {
		t.Fatalf("Get proj1 failed: %v", err)
	}
	s2, err := registry.Get("proj2", "main")
	if err != nil {
		t.Fatalf("Get proj2 failed: %v", err)
	}

	if s1.ID != "id1" || s2.ID != "id2" {
		t.Errorf("IDs mismatch: got %s/%s", s1.ID, s2.ID)
	}
}

func TestRegistry_CreateDirectory(t *testing.T) {
	t.Parallel()

	// Test that registry creates parent directory if it doesn't exist
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "id1", "/dir")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestRegistry_UpdateAgentState(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = registry.UpdateAgentState("project", "session", "atlas", true)
	if err != nil {
		t.Fatalf("UpdateAgentState failed: %v", err)
	}

	session, err := registry.Get("project", "session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.LastAgent != "atlas" {
		t.Errorf("Expected last_agent 'atlas', got '%s'", session.LastAgent)
	}
	if !session.IsAgentLocked {
		t.Errorf("Expected IsAgentLocked true, got false")
	}
}

func TestRegistry_UpdateAgentState_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.UpdateAgentState("nonexistent", "session", "atlas", true)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_FindByID(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "session-id-123", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session, err := registry.FindByID("session-id-123")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if session.ID != "session-id-123" {
		t.Errorf("Expected ID session-id-123, got %s", session.ID)
	}
	if session.Project != "project" {
		t.Errorf("Expected project 'project', got %s", session.Project)
	}
	if session.SessionName != "session" {
		t.Errorf("Expected session_name 'session', got %s", session.SessionName)
	}
}

func TestRegistry_FindByID_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	_, err = registry.FindByID("nonexistent-id")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_FindByID_WithAgentState(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "session-id-456", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = registry.UpdateAgentState("project", "session", "atlas", true)
	if err != nil {
		t.Fatalf("UpdateAgentState failed: %v", err)
	}

	session, err := registry.FindByID("session-id-456")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if session.LastAgent != "atlas" {
		t.Errorf("Expected last_agent 'atlas', got '%s'", session.LastAgent)
	}
	if !session.IsAgentLocked {
		t.Errorf("Expected IsAgentLocked true, got false")
	}
}

func TestRegistry_UpdateState(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = registry.UpdateState("project", "session", "BUSY")
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	session, err := registry.Get("project", "session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.State != "BUSY" {
		t.Errorf("Expected state BUSY, got %s", session.State)
	}
}

func TestRegistry_UpdateState_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.UpdateState("nonexistent", "session", "BUSY")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_UpdateLastActivity(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	timestamp := "2026-02-16T14:00:00Z"
	err = registry.UpdateLastActivity("project", "session", timestamp)
	if err != nil {
		t.Fatalf("UpdateLastActivity failed: %v", err)
	}

	session, err := registry.Get("project", "session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.LastActivity != timestamp {
		t.Errorf("Expected last_activity %s, got %s", timestamp, session.LastActivity)
	}
}

func TestRegistry_UpdateLastActivity_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.UpdateLastActivity("nonexistent", "session", "2026-02-16T14:00:00Z")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_UpdateSessionData(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	err = registry.Create("project", "session", "id-1", "/dir1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updatedData := SessionData{
		LastAgent:      "atlas",
		IsAgentLocked:  true,
		State:          "BUSY",
		LatestResponse: `{"result": "success"}`,
		Questions:      `[{"id": "q1", "text": "Question?"}]`,
		LastActivity:   "2026-02-16T14:00:00Z",
	}

	err = registry.UpdateSessionData("project", "session", updatedData)
	if err != nil {
		t.Fatalf("UpdateSessionData failed: %v", err)
	}

	session, err := registry.Get("project", "session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.LastAgent != "atlas" {
		t.Errorf("Expected last_agent atlas, got %s", session.LastAgent)
	}
	if !session.IsAgentLocked {
		t.Errorf("Expected IsAgentLocked true, got false")
	}
	if session.State != "BUSY" {
		t.Errorf("Expected state BUSY, got %s", session.State)
	}
	if session.LatestResponse != `{"result": "success"}` {
		t.Errorf("Expected latest_response JSON, got %s", session.LatestResponse)
	}
	if session.Questions != `[{"id": "q1", "text": "Question?"}]` {
		t.Errorf("Expected questions JSON, got %s", session.Questions)
	}
	if session.LastActivity != "2026-02-16T14:00:00Z" {
		t.Errorf("Expected last_activity timestamp, got %s", session.LastActivity)
	}
}

func TestRegistry_UpdateSessionData_NotFound(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	registry, err := NewRegistry(dbPath)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	updatedData := SessionData{
		LastAgent:     "atlas",
		IsAgentLocked: true,
		State:         "BUSY",
	}

	err = registry.UpdateSessionData("nonexistent", "session", updatedData)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}
