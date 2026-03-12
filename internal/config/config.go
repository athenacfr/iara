package config

import (
	"os"
	"path/filepath"
)

type Mode struct {
	Name        string
	Description string
	Flag        string // "--append-system-prompt-file" for system prompt injection
	Value       string // file path or command string
}

var Modes []Mode

func InitModes(modesDir string) {
	Modes = []Mode{
		{Name: "code", Description: "Write features, fix bugs (default)"},
		{Name: "research", Description: "Explore codebase, read-only", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "research.md")},
		{Name: "review", Description: "Review code changes", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "review.md")},
		{Name: "debug", Description: "Investigate issues", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "debug.md")},
		{Name: "plan", Description: "Plan before building", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "plan.md")},
		{Name: "tdd", Description: "Test-driven development", Flag: "--append-system-prompt-file", Value: filepath.Join(modesDir, "tdd.md")},
		{Name: "free", Description: "No preset behavior"},
	}
}

func ProjectsDir() string {
	dir := os.Getenv("CW_PROJECTS_DIR")
	if dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cw", "projects")
}

func EnsureProjectsDir() error {
	return os.MkdirAll(ProjectsDir(), 0755)
}
