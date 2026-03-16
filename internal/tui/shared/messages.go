package shared

import (
	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/task"
)

// Screen identifies a TUI screen.
type Screen int

const (
	ScreenProjectExplorer Screen = iota
	ScreenProjectWizard
	ScreenAddRepo
	ScreenLauncher
	ScreenSettings
	ScreenTaskSelect
)

// ProjectSelectedMsg is sent when a project is selected from the project list.
type ProjectSelectedMsg struct{ Project *project.Project }

// RepoSelectedMsg is sent when a repo is directly selected.
type RepoSelectedMsg struct {
	ProjectName string
	RepoName    string
}

// ModeSelectedMsg is sent when a mode and session are confirmed.
type ModeSelectedMsg struct {
	Mode            config.Mode
	SkipPermissions bool
	SessionKind     int
	ResumeSessionID string
}

// NavigateMsg is sent to switch between screens.
type NavigateMsg struct {
	Screen      Screen
	ProjectName string
}

// TaskSelectedMsg is sent when a task is selected from the task list.
type TaskSelectedMsg struct {
	Task        *task.Task // nil for default branch
	SessionsDir string     // full path to sessions directory
	WorkDir     string     // worktree base or project root
	IsDefault   bool       // true for default branch entry
	IsNew       bool       // true for "+ New Task" entry
}

// LaunchMsg signals that Claude should be launched.
type LaunchMsg struct{}
