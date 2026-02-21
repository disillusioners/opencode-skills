package config

import (
	"os"
	"path/filepath"
	"time"
)

// OpenCode Configuration
const (
	OpenCodeURL  = "http://127.0.0.1:4096"
	DefaultAgent = "orchestrator"
	DefaultModel = "zai-coding-plan/glm-5"
)

// Daemon Configuration
const (
	DaemonHost = "127.0.0.1"
	DaemonPort = 44111
)

// Timing
const (
	PollInterval   = 2 * time.Second
	ClientTimeout  = 10 * time.Minute
	AutoFixTimeout = 15 * time.Minute
)

// Paths
var (
	ProjectRoot    string
	WrapperDir     string
	PidFile        string
	SessionMapFile string
	LogFile        string
)

func init() {
	var err error
	ProjectRoot, err = getProjectRoot()
	if err != nil {
		ProjectRoot, _ = os.Getwd()
	}

	homeDir, _ := os.UserHomeDir()
	WrapperDir = filepath.Join(homeDir, ".opencode_skill")

	if _, err := os.Stat(WrapperDir); os.IsNotExist(err) {
		_ = os.MkdirAll(WrapperDir, 0755)
	}

	PidFile = filepath.Join(WrapperDir, "daemon.pid")
	SessionMapFile = filepath.Join(WrapperDir, "sessions.db")
	LogFile = filepath.Join(WrapperDir, "daemon.log")
}

func getProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := cwd
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return cwd, nil
}
