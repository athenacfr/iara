package config

import (
	"path/filepath"
)

type Mode struct {
	Name        string
	Description string
	Flag        string // "--append-system-prompt-file" for system prompt injection
	Value       string // file path or command string
}

var Modes []Mode

// GetMode looks up a mode by name. Returns the mode and true if found.
func GetMode(name string) (Mode, bool) {
	for _, m := range Modes {
		if m.Name == name {
			return m, true
		}
	}
	return Mode{}, false
}

func InitModes(modesDir string) {
	Modes = []Mode{
		{Name: "code", Description: "Write features, fix bugs (default)"},
		{Name: "research", Description: "Explore codebase, read-only", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "research.md")},
		{Name: "review", Description: "Review code changes", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "review.md")},
		{Name: "none", Description: "No preset behavior"},
	}
}

