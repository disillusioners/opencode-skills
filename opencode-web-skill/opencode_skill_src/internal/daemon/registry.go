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
	Project     string `json:"project"`
	SessionName string `json:"session_name"`
	ID          string `json:"session_id"`
	WorkingDir  string `json:"working_dir"`
	LockedAgent string `json:"locked_agent"` // Empty (default) or agent name (e.g., "atlas")
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
		"locked_agent" TEXT DEFAULT '',
		PRIMARY KEY (project, session_name)
	);`

	// Migration: Add locked_agent column if it doesn't exist (for existing databases)
	migrateSQL := `ALTER TABLE sessions ADD COLUMN locked_agent TEXT DEFAULT ''`
	db.Exec(migrateSQL) // Ignore error if column already exists

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, err
	}

	return &Registry{db: db}, nil
}

func (r *Registry) Create(project, sessionName, id, workingDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stmt, err := r.db.Prepare("INSERT INTO sessions (project, session_name, id, working_dir, locked_agent) VALUES (?, ?, ?, ?, '')")
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

	row := r.db.QueryRow("SELECT project, session_name, id, working_dir, locked_agent FROM sessions WHERE project = ? AND session_name = ?", project, sessionName)

	var session SessionData
	err := row.Scan(&session.Project, &session.SessionName, &session.ID, &session.WorkingDir, &session.LockedAgent)
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

	rows, err := r.db.Query("SELECT project, session_name, id, working_dir, locked_agent FROM sessions ORDER BY project, session_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var s SessionData
		if err := rows.Scan(&s.Project, &s.SessionName, &s.ID, &s.WorkingDir, &s.LockedAgent); err != nil {
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

func (r *Registry) UpdateLockedAgent(project, sessionName, lockedAgent string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec("UPDATE sessions SET locked_agent = ? WHERE project = ? AND session_name = ?", lockedAgent, project, sessionName)
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

	row := r.db.QueryRow("SELECT project, session_name, id, working_dir, locked_agent FROM sessions WHERE id = ?", sessionID)

	var session SessionData
	err := row.Scan(&session.Project, &session.SessionName, &session.ID, &session.WorkingDir, &session.LockedAgent)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}
