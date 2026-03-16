package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/paths"
)

type Settings struct {
	BypassPermissions   bool   `json:"bypass_permissions"`
	AutoCompactLimit    int    `json:"auto_compact_limit,omitempty"`    // 0=off, 40/50/60/70/80
	LoadSubprojectRules bool   `json:"load_subproject_rules"`           // load CLAUDE.md from --add-dir repos
	DefaultMode         string `json:"default_mode,omitempty"`          // code, research, review, none
	EnableHooks         bool   `json:"enable_hooks"`                    // sync hooks to .claude/settings.json
}

// DefaultSettings returns settings with sensible defaults for first use.
func DefaultSettings() Settings {
	return Settings{
		BypassPermissions:   true,
		LoadSubprojectRules: true,
		EnableHooks:         true,
	}
}

func globalSettingsPath() string {
	return filepath.Join(paths.DataDir(), "settings.json")
}

func LoadGlobalSettings() Settings {
	data, err := os.ReadFile(globalSettingsPath())
	if err != nil {
		return DefaultSettings()
	}
	// Start from defaults so new bool fields are true when missing from old JSON
	s := DefaultSettings()
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings()
	}
	return s
}

func SaveGlobalSettings(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(globalSettingsPath(), append(data, '\n'), 0644)
}
