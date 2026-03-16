package claude

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/ahtwr/cw/internal/config"
)

type LaunchConfig struct {
	WorkDir             string
	ProjectName         string
	PluginDir           string
	Mode                config.Mode
	Prompt              string   // initial message (positional arg to claude)
	SystemPrompts       []string // prompt strings passed via --append-system-prompt
	AddDirs             []string // repo paths passed via --add-dir (loads rules via env var)
	ResumeSessionID     string   // pass --resume <id> to resume an existing session
	NewSessionID        string   // pass --session-id <uuid> to force a specific ID for a new session
	SkipPermissions     bool     // pass --dangerously-skip-permissions to claude
	EditorMode          bool     // open WorkDir with $EDITOR instead of launching claude
	AutoSetup           bool     // set CW_AUTO_SETUP=1 so skills know cw auto-invoked them
	AutoCompactLimit    int      // 0=off, 40/50/60/70/80 — context % threshold for auto-compact
	CWSessionID         string   // cw session ID (passed as env var for hooks)
	Print               bool     // run in non-interactive mode (-p): process prompt and exit
	Quiet               bool     // suppress stdout (used during auto-compact)
	YoloActive          bool     // set CW_YOLO_ACTIVE=1 for yolo autonomous execution
	YoloPlanPath        string   // absolute path to yolo plan file (CW_YOLO_PLAN env var)
	LoadSubprojectRules bool     // when true, set CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1
	TaskID              string   // task UUID (passed as CW_TASK_ID env var)
	TaskName            string   // task name/slug (passed as CW_TASK_NAME env var)
	SessionsDir         string   // full path to task-scoped sessions directory
}

// reloadRequested is set when SIGUSR1 is received from `cw internal reload`
var reloadRequested atomic.Bool

// WasReload returns true if the last session ended due to a reload signal.
func WasReload() bool {
	return reloadRequested.Load()
}

func Launch(cfg LaunchConfig) error {
	reloadRequested.Store(false)

	var args []string
	if cfg.Print {
		args = append(args, "-p")
	}
	if cfg.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	if cfg.ResumeSessionID != "" {
		args = append(args, "--resume", cfg.ResumeSessionID)
	} else if cfg.NewSessionID != "" {
		args = append(args, "--session-id", cfg.NewSessionID)
	}

	if cfg.PluginDir != "" {
		args = append(args, "--plugin-dir", cfg.PluginDir)
	}

	// Use --agent for modes that define an agent (e.g., researcher, reviewer)
	if cfg.Mode.Agent != "" {
		args = append(args, "--agent", cfg.Mode.Agent)
	}

	if len(cfg.SystemPrompts) > 0 {
		combined := strings.Join(cfg.SystemPrompts, "\n\n---\n\n")
		args = append(args, "--append-system-prompt", combined)
	}

	for _, dir := range cfg.AddDirs {
		args = append(args, "--add-dir", dir)
	}

	// Initial prompt as positional arg (e.g., /cw:new-intention for onboarding)
	if cfg.Prompt != "" {
		args = append(args, "--", cfg.Prompt)
	}

	// Clear screen and position cursor at top-left before launching Claude
	if !cfg.Quiet {
		os.Stdout.WriteString("\033[H\033[2J\033[H")
		os.Stdout.Sync()
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = cfg.WorkDir

	// Pass CW_PID so `cw internal reload` can signal us
	env := os.Environ()
	env = append(env, fmt.Sprintf("CW_PID=%d", os.Getpid()))
	if cfg.ProjectName != "" {
		env = append(env, fmt.Sprintf("CW_PROJECT_DIR=%s", cfg.WorkDir))
		env = append(env, fmt.Sprintf("CW_PROJECT=%s", cfg.ProjectName))
	}
	if cfg.Mode.Name != "" {
		env = append(env, fmt.Sprintf("CW_MODE=%s", cfg.Mode.Name))
	}
	if cfg.AutoSetup {
		env = append(env, "CW_AUTO_SETUP=1")
	}
	if cfg.AutoCompactLimit > 0 {
		env = append(env, fmt.Sprintf("CW_AUTO_COMPACT_LIMIT=%d", cfg.AutoCompactLimit))
	}
	if cfg.CWSessionID != "" {
		env = append(env, fmt.Sprintf("CW_SESSION_ID=%s", cfg.CWSessionID))
	}
	if cfg.YoloActive {
		env = append(env, "CW_YOLO_ACTIVE=1")
	}
	if cfg.YoloPlanPath != "" {
		env = append(env, fmt.Sprintf("CW_YOLO_PLAN=%s", cfg.YoloPlanPath))
	}
	if cfg.TaskID != "" {
		env = append(env, fmt.Sprintf("CW_TASK_ID=%s", cfg.TaskID))
	}
	if cfg.TaskName != "" {
		env = append(env, fmt.Sprintf("CW_TASK_NAME=%s", cfg.TaskName))
	}
	if cfg.SessionsDir != "" {
		// Task base dir is parent of sessions dir — used for dev-config and logs
		env = append(env, fmt.Sprintf("CW_TASK_DIR=%s", filepath.Dir(cfg.SessionsDir)))
	}
	if len(cfg.AddDirs) > 0 && cfg.LoadSubprojectRules {
		env = append(env, "CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1")
	}
	cmd.Env = env

	cmd.Stdin = os.Stdin
	if cfg.Quiet {
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Listen for SIGUSR1 (reload signal) and kill Claude when received
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	go func() {
		<-sigCh
		reloadRequested.Store(true)
		cmd.Process.Signal(syscall.SIGTERM)
	}()

	err := cmd.Wait()

	signal.Stop(sigCh)

	// Clear screen after Claude exits
	if !cfg.Quiet {
		os.Stdout.WriteString("\033[H\033[2J")
	}

	// Suppress error if this was a reload
	if reloadRequested.Load() {
		return nil
	}

	return err
}
