package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/paths"
)

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type matcherGroup struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []hookEntry `json:"hooks"`
}

type hooksConfig struct {
	Hooks map[string][]matcherGroup `json:"hooks"`
}

func EnsureHooks(name, cwRoot string) error {
	projectDir := filepath.Join(paths.ProjectsDir(), name)
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	preWriteGuard := filepath.Join(cwRoot, "hooks", "pre-write-guard.sh")
	autoCompact := filepath.Join(cwRoot, "hooks", "auto-compact.sh")
	yoloStop := filepath.Join(cwRoot, "hooks", "yolo-stop.sh")

	cfg := hooksConfig{
		Hooks: map[string][]matcherGroup{
			"PreToolUse": {
				{
					Matcher: "Edit|Write",
					Hooks: []hookEntry{
						{
							Type:    "command",
							Command: preWriteGuard,
						},
					},
				},
			},
			"PostToolUse": {
				{
					Matcher: "*",
					Hooks: []hookEntry{
						{
							Type:    "command",
							Command: autoCompact,
						},
					},
				},
			},
			"Stop": {
				{
					Hooks: []hookEntry{
						{
							Type:    "command",
							Command: yoloStop,
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
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return err
	}

	// Ensure .gitignore exists to hide .claude/ from Claude's file selector
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		os.WriteFile(gitignorePath, []byte(".*\n"), 0644)
	}

	return nil
}
