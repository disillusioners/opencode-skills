package daemon

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type SessionData struct {
	Project        string `json:"project"`
	SessionName    string `json:"session_name"`
	ID             string `json:"session_id"`
	WorkingDir     string `json:"working_dir"`
	LastAgent      string `json:"last_agent"`
	IsAgentLocked  bool   `json:"is_agent_locked"`
	State          string `json:"state"`
	LatestResponse string `json:"latest_response"`
	Questions      string `json:"questions"`
	LastActivity   string `json:"last_activity"`
}

var (
	ErrNotFound  = errors.New("session not found")
	ErrDuplicate = errors.New("session already exists")
)

type Registry struct {
	db *sql.DB
	mu sync.Mutex
}

func NewRegistry(dbPath string) (*Registry, error) {
	if dbPath == "" {
		return nil, errors.New("database path cannot be empty")
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", absPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS sessions (
		"project" TEXT NOT NULL,
		"session_name" TEXT NOT NULL,
		"id" TEXT,
		"working_dir" TEXT,
		"last_agent" TEXT DEFAULT '',
		"is_agent_locked" INTEGER DEFAULT 0,
		"state" TEXT DEFAULT 'IDLE',
		"latest_response" TEXT DEFAULT '',
		"questions" TEXT DEFAULT '[]',
		"last_activity" TEXT DEFAULT '',
		PRIMARY KEY (project, session_name)
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, err
	}

	return &Registry{db: db}, nil
}

func (r *Registry) Create(project, sessionName, id, workingDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stmt, err := r.db.Prepare("INSERT INTO sessions (project, session_name, id, working_dir, last_agent) VALUES (?, ?, ?, ?, '')")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(project, sessionName, id, workingDir)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: sessions.project, sessions.session_name" {
			return ErrDuplicate
		}
		return err
	}

	return nil
}

func (r *Registry) Get(project, sessionName string) (*SessionData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	row := r.db.QueryRow("SELECT project, session_name, id, working_dir, last_agent, is_agent_locked, state, latest_response, questions, last_activity FROM sessions WHERE project = ? AND session_name = ?", project, sessionName)

	var session SessionData
	err := row.Scan(&session.Project, &session.SessionName, &session.ID, &session.WorkingDir, &session.LastAgent, &session.IsAgentLocked, &session.State, &session.LatestResponse, &session.Questions, &session.LastActivity)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *Registry) List() ([]SessionData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.Query("SELECT project, session_name, id, working_dir, last_agent, is_agent_locked, state, latest_response, questions, last_activity FROM sessions ORDER BY project, session_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var s SessionData
		if err := rows.Scan(&s.Project, &s.SessionName, &s.ID, &s.WorkingDir, &s.LastAgent, &s.IsAgentLocked, &s.State, &s.LatestResponse, &s.Questions, &s.LastActivity); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

func (r *Registry) Delete(project, sessionName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec("DELETE FROM sessions WHERE project = ? AND session_name = ?", project, sessionName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Registry) UpdateAgentState(project, sessionName, lastAgent string, isLocked bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lockedInt := 0
	if isLocked {
		lockedInt = 1
	}

	result, err := r.db.Exec("UPDATE sessions SET last_agent = ?, is_agent_locked = ? WHERE project = ? AND session_name = ?", lastAgent, lockedInt, project, sessionName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Registry) UpdateState(project, sessionName, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec("UPDATE sessions SET state = ? WHERE project = ? AND session_name = ?", state, project, sessionName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Registry) UpdateLastActivity(project, sessionName, timestamp string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec("UPDATE sessions SET last_activity = ? WHERE project = ? AND session_name = ?", timestamp, project, sessionName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Registry) UpdateSessionData(project, sessionName string, session SessionData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lockedInt := 0
	if session.IsAgentLocked {
		lockedInt = 1
	}

	result, err := r.db.Exec(
		"UPDATE sessions SET last_agent = ?, is_agent_locked = ?, state = ?, latest_response = ?, questions = ?, last_activity = ? WHERE project = ? AND session_name = ?",
		session.LastAgent, lockedInt, session.State, session.LatestResponse, session.Questions, session.LastActivity, project, sessionName,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.db.Close()
}

func (r *Registry) FindByID(sessionID string) (*SessionData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	row := r.db.QueryRow("SELECT project, session_name, id, working_dir, last_agent, is_agent_locked, state, latest_response, questions, last_activity FROM sessions WHERE id = ?", sessionID)

	var session SessionData
	err := row.Scan(&session.Project, &session.SessionName, &session.ID, &session.WorkingDir, &session.LastAgent, &session.IsAgentLocked, &session.State, &session.LatestResponse, &session.Questions, &session.LastActivity)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}
