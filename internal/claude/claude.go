package claude

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/ahtwr/cw/internal/config"
)

type LaunchConfig struct {
	WorkDir         string
	ProjectName     string
	PluginDir       string
	Mode            config.Mode
	Prompt          string   // initial message (positional arg to claude)
	SystemPrompts   []string // prompt strings passed via --append-system-prompt
	AddDirs         []string // repo paths passed via --add-dir (loads rules via env var)
	SkipPermissions bool     // pass --dangerously-skip-permissions to claude
	Continue        bool     // pass --continue to resume most recent conversation
	EditorMode      bool     // open WorkDir with $EDITOR instead of launching claude
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
	if cfg.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	if cfg.Continue {
		args = append(args, "--continue")
	}

	if cfg.PluginDir != "" {
		args = append(args, "--plugin-dir", cfg.PluginDir)
	}

	if len(cfg.SystemPrompts) > 0 {
		combined := strings.Join(cfg.SystemPrompts, "\n\n---\n\n")
		args = append(args, "--append-system-prompt", combined)
	}

	for _, dir := range cfg.AddDirs {
		args = append(args, "--add-dir", dir)
	}

	// Initial prompt as positional arg (e.g., /cw:new-project for onboarding)
	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	// Clear screen and position cursor at top-left before launching Claude
	os.Stdout.WriteString("\033[H\033[2J\033[H")
	os.Stdout.Sync()

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
	if len(cfg.AddDirs) > 0 {
		env = append(env, "CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1")
	}
	cmd.Env = env

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
	os.Stdout.WriteString("\033[H\033[2J")

	// Suppress error if this was a reload
	if reloadRequested.Load() {
		return nil
	}

	return err
}
