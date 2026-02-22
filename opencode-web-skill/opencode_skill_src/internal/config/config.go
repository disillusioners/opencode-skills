package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OpenCode Configuration
const (
	OpenCodeURL  = "http://127.0.0.1:4096"
	DefaultAgent = "orchestrator"
	DefaultModel = "litellm/glm-5"
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
	ConfigFile     string
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
	ConfigFile = filepath.Join(WrapperDir, "config.json")
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

// UserConfig represents the user-configurable settings stored in config.json
type UserConfig struct {
	DefaultModel string `json:"defaultModel"`
}

var (
	configMutex  sync.RWMutex
	cachedConfig *UserConfig
)

// LoadConfig reads and parses the config.json file.
// If it doesn't exist or is invalid, a default configuration is returned.
// It caches the result after the first successful read.
func LoadConfig() *UserConfig {
	configMutex.RLock()
	if cachedConfig != nil {
		defer configMutex.RUnlock()
		return cachedConfig
	}
	configMutex.RUnlock()

	configMutex.Lock()
	defer configMutex.Unlock()

	// Double-check after acquiring write lock
	if cachedConfig != nil {
		return cachedConfig
	}

	defaultCfg := &UserConfig{
		DefaultModel: DefaultModel, // Fallback to hardcoded default
	}

	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		cachedConfig = defaultCfg
		return cachedConfig // File missing or unreadable
	}

	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		cachedConfig = defaultCfg
		return cachedConfig // Invalid JSON
	}

	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultModel // Ensure fallback if missing in JSON
	}

	cachedConfig = &cfg
	return cachedConfig
}

// SaveConfig writes the UserConfig to config.json and updates the cache.
func SaveConfig(cfg *UserConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(ConfigFile, data, 0644)
	if err == nil {
		// Update cache on successful save
		cachedConfig = cfg
	}
	return err
}

// GetDefaultModel returns the configured default model, or the hardcoded default.
func GetDefaultModel() string {
	return LoadConfig().DefaultModel
}
