package dev

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigVersion is the current config schema version.
// Bump this when the config format changes to force re-discovery.
const ConfigVersion = 2

// Config is the dev-config.json structure.
type Config struct {
	Version     int          `json:"version"`
	PortBase    int          `json:"portBase,omitempty"`
	Subprojects []Subproject `json:"subprojects"`
}

// Subproject defines a subproject with its path, optional port/venv, and commands.
type Subproject struct {
	Path     string    `json:"path"`
	Port     int       `json:"port,omitempty"`
	Venv     string    `json:"venv,omitempty"`
	Commands []Command `json:"commands"`
}

// Command defines a single command to run within a subproject.
type Command struct {
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
	Type        string `json:"type"`     // "one-shot" or "long-running"
	Priority    int    `json:"priority"` // execution priority within type group (lower runs first)
}

// Status is written to dev-status.json by the supervisor.
type Status struct {
	PID       int             `json:"pid"`
	Started   string          `json:"started"` // RFC3339
	Processes []ProcessStatus `json:"processes"`
}

// ProcessStatus tracks the state of a managed process.
type ProcessStatus struct {
	Subproject string `json:"subproject"`
	Cmd        string `json:"cmd"`
	Type       string `json:"type"`
	PID        int    `json:"pid,omitempty"`
	Port       int    `json:"port,omitempty"`
	Status     string `json:"status"` // "running", "completed", "failed", "stopped"
	Error      string `json:"error,omitempty"`
}

// ConfigPath returns the path to dev-config.json in the given task directory.
func ConfigPath(taskDir string) string {
	return filepath.Join(taskDir, "dev-config.json")
}

// StatusPath returns the path to dev-status.json in the given task directory.
func StatusPath(taskDir string) string {
	return filepath.Join(taskDir, "dev-status.json")
}

// PIDPath returns the path to dev.pid in the given task directory.
func PIDPath(taskDir string) string {
	return filepath.Join(taskDir, "dev.pid")
}

// LoadConfig reads and parses the dev-config.json from the task directory.
func LoadConfig(taskDir string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(taskDir))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes the config to dev-config.json with indentation.
// It automatically sets the version to the current ConfigVersion.
func SaveConfig(taskDir string, cfg *Config) error {
	cfg.Version = ConfigVersion
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(ConfigPath(taskDir), data, 0644)
}

// IsCurrent checks if the config file exists and is at the current version.
// Returns true if config exists and version matches, false otherwise.
func IsCurrent(taskDir string) bool {
	cfg, err := LoadConfig(taskDir)
	if err != nil {
		return false
	}
	return cfg.Version == ConfigVersion
}

// DeleteConfig removes the config file so discovery re-runs.
func DeleteConfig(taskDir string) error {
	return os.Remove(ConfigPath(taskDir))
}
