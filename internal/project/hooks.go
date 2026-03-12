package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/config"
)

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type matcherGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

type hooksConfig struct {
	Hooks map[string][]matcherGroup `json:"hooks"`
}

func EnsureHooks(name, cwRoot string) error {
	projectDir := filepath.Join(config.ProjectsDir(), name)
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	hookScript := filepath.Join(cwRoot, "hooks", "pre-write-guard.sh")

	cfg := hooksConfig{
		Hooks: map[string][]matcherGroup{
			"PreToolUse": {
				{
					Matcher: "Edit|Write",
					Hooks: []hookEntry{
						{
							Type:    "command",
							Command: hookScript,
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	return os.WriteFile(settingsPath, data, 0644)
}
