package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ahtwr/cw/internal/claude"
	"github.com/ahtwr/cw/internal/config"
	cwembed "github.com/ahtwr/cw/internal/embed"
	"github.com/ahtwr/cw/internal/env"
	"github.com/ahtwr/cw/internal/paths"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/session"
	"github.com/ahtwr/cw/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func uninstall() {
	binary := filepath.Join(paths.BinDir(), "cw")
	dataDir := paths.DataDir()
	projectsDir := paths.ProjectsDir()

	fmt.Println("This will remove:")
	fmt.Printf("  • Binary:    %s\n", binary)
	fmt.Printf("  • Data:      %s\n", dataDir)
	if projectsDir != dataDir && strings.HasPrefix(projectsDir, dataDir) {
		fmt.Printf("  • Projects:  %s (included in data dir)\n", projectsDir)
	} else if projectsDir != filepath.Join(dataDir, "projects") {
		fmt.Printf("  • Projects:  %s\n", projectsDir)
	}

	fmt.Print("\nAre you sure? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}

	removed := 0

	if err := os.Remove(binary); err == nil {
		fmt.Printf("Removed %s\n", binary)
		removed++
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", binary, err)
	}

	if err := os.RemoveAll(dataDir); err == nil {
		fmt.Printf("Removed %s\n", dataDir)
		removed++
	} else {
		fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", dataDir, err)
	}

	if !strings.HasPrefix(projectsDir, dataDir) {
		fmt.Print(fmt.Sprintf("\nAlso remove projects at %s? [y/N] ", projectsDir))
		answer, _ = reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "yes" {
			if err := os.RemoveAll(projectsDir); err == nil {
				fmt.Printf("Removed %s\n", projectsDir)
				removed++
			} else {
				fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", projectsDir, err)
			}
		}
	}

	if removed > 0 {
		fmt.Println("\ncw has been uninstalled.")
	}
}

func internalReload() {
	pidStr := os.Getenv("CW_PID")
	if pidStr == "" {
		fmt.Fprintln(os.Stderr, "CW_PID not set — not running inside cw")
		os.Exit(1)
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid CW_PID: %s\n", pidStr)
		os.Exit(1)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find cw process %d: %v\n", pid, err)
		os.Exit(1)
	}
	if err := proc.Signal(syscall.SIGUSR1); err != nil {
		fmt.Fprintf(os.Stderr, "cannot signal cw process: %v\n", err)
		os.Exit(1)
	}
}

func internalNewSession() {
	if err := os.WriteFile(paths.NewSessionFile(), []byte("1"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write new-session file: %v\n", err)
		os.Exit(1)
	}
	internalReload()
}

func internalAutoCompact() {
	if err := os.WriteFile(paths.AutoCompactFile(), []byte("1"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write auto-compact file: %v\n", err)
		os.Exit(1)
	}
	internalReload()
}

func internalModeSwitch(modeName string) {
	config.InitModes(cwembed.ModesDir())
	if _, ok := config.GetMode(modeName); !ok {
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", modeName)
		os.Exit(1)
	}
	if err := os.WriteFile(paths.ModeOverrideFile(), []byte(modeName), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write mode override: %v\n", err)
		os.Exit(1)
	}
	internalReload()
}

func internalPermissionsSwitch(value string) {
	if value != "bypass" && value != "normal" {
		fmt.Fprintf(os.Stderr, "invalid permissions value: %s (use 'bypass' or 'normal')\n", value)
		os.Exit(1)
	}
	if err := os.WriteFile(paths.PermissionsOverrideFile(), []byte(value), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write permissions override: %v\n", err)
		os.Exit(1)
	}
	internalReload()
}

func internalSaveMetadata(jsonStr string) {
	projectDir := os.Getenv("CW_PROJECT_DIR")
	if projectDir == "" {
		fmt.Fprintln(os.Stderr, "CW_PROJECT_DIR not set — not running inside cw")
		os.Exit(1)
	}
	if err := project.SaveMetadata(projectDir, jsonStr); err != nil {
		fmt.Fprintf(os.Stderr, "save-metadata: %v\n", err)
		os.Exit(1)
	}
}

// openFolder opens a directory in the best available editor/file manager.
// Priority: $VISUAL, VS Code, then platform file explorer.
func openFolder(dir string) {
	if editor := os.Getenv("VISUAL"); editor != "" {
		cmd := exec.Command(editor, dir)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "editor exited: %v\n", err)
		}
		return
	}

	if path, err := exec.LookPath("code"); err == nil {
		exec.Command(path, dir).Start()
		return
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", dir)
	} else {
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "could not open folder: %v\n", err)
	}
}

func internalYoloStart() {
	projectDir := os.Getenv("CW_PROJECT_DIR")
	if projectDir == "" {
		fmt.Fprintln(os.Stderr, "CW_PROJECT_DIR not set — not running inside cw")
		os.Exit(1)
	}
	matches, _ := filepath.Glob(filepath.Join(projectDir, ".cw", "yolo", "plan-*.md"))
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "No yolo plan found in "+filepath.Join(projectDir, ".cw", "yolo"))
		os.Exit(1)
	}
	planPath, _ := filepath.Abs(matches[0])
	if err := os.WriteFile(paths.YoloActiveFile(), []byte(planPath), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write yolo sideband: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Yolo mode activated")
}

func internalYoloStop() {
	// Read plan path from sideband
	data, err := os.ReadFile(paths.YoloActiveFile())
	if err == nil {
		planPath := strings.TrimSpace(string(data))
		os.Remove(planPath)
	}
	os.Remove(paths.YoloActiveFile())
	fmt.Println("Yolo mode deactivated")
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "uninstall" {
		uninstall()
		return
	}

	// Internal CLI commands (called by Claude via Bash tool)
	if len(os.Args) > 2 && os.Args[1] == "internal" {
		switch os.Args[2] {
		case "reload":
			internalReload()
			return
		case "new-session":
			internalNewSession()
			return
		case "auto-compact", "compact-and-continue":
			internalAutoCompact()
			return
		case "mode-switch":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: cw internal mode-switch <mode>")
				os.Exit(1)
			}
			internalModeSwitch(os.Args[3])
			return
		case "permissions-switch":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: cw internal permissions-switch <bypass|normal>")
				os.Exit(1)
			}
			internalPermissionsSwitch(os.Args[3])
			return
		case "open-project":
			dir := os.Getenv("CW_PROJECT_DIR")
			if dir == "" {
				fmt.Fprintln(os.Stderr, "CW_PROJECT_DIR not set — not running inside cw")
				os.Exit(1)
			}
			go openFolder(dir)
			time.Sleep(500 * time.Millisecond)
			return
		case "save-metadata":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: cw internal save-metadata '<json>'")
				os.Exit(1)
			}
			internalSaveMetadata(os.Args[3])
			return
		case "yolo-start":
			internalYoloStart()
			return
		case "yolo-stop":
			internalYoloStop()
			return
		}
	}

	// Extract embedded files (plugins, modes, hooks)
	if err := cwembed.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing embedded files: %v\n", err)
		os.Exit(1)
	}

	config.InitModes(cwembed.ModesDir())

	if err := paths.EnsureProjectsDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating projects dir: %v\n", err)
		os.Exit(1)
	}

	var returnProject *project.Project
	var returnBypass bool

	for {
		var m tui.Model
		if returnProject != nil {
			m = tui.NewModelWithProject(cwembed.PluginDir(), returnProject, returnBypass)
			returnProject = nil
		} else {
			m = tui.NewModel(cwembed.PluginDir())
		}
		p := tea.NewProgram(m, tea.WithAltScreen())

		result, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		model, ok := result.(tui.Model)
		if !ok || !model.ShouldLaunch() {
			break
		}

		cfg := model.LaunchConfig()

		// Editor mode: open project folder and loop back to TUI
		if cfg.EditorMode {
			openFolder(cfg.WorkDir)
			continue
		}

		// Create or load the CW session for this launch
		var cwSession *session.Session
		if cfg.SessionID != "" {
			// Resuming a specific CW session — load it and restore state
			if s, err := session.Load(cfg.WorkDir, cfg.SessionID); err == nil {
				cwSession = s
				// Restore mode from session
				if m, ok := config.GetMode(s.Mode); ok {
					cfg.Mode = m
				}
				cfg.SkipPermissions = s.SkipPermissions
				cfg.AutoCompactLimit = s.AutoCompactLimit
				// Use the Claude session ID for --resume
				cfg.SessionID = s.ClaudeSessionID
			}
		}
		if cwSession == nil {
			// New session — generate an ID and create
			cwSession = session.New(
				generateSessionID(),
				cfg.Mode.Name,
				cfg.SkipPermissions,
				cfg.AutoCompactLimit,
			)
			cwSession.Save(cfg.WorkDir)
		}

		// Reload loop: re-sync and re-launch on reload signal
		for {
			var envWatcher *env.Watcher
			if cfg.ProjectName != "" {
				sysPrompt, err := project.BuildSystemPrompt(cfg.ProjectName)
				if err == nil {
					cfg.SystemPrompts = append(cfg.SystemPrompts, sysPrompt)
				}

				// Inject instructions from project metadata
				if meta, err := project.LoadMetadata(cfg.ProjectName); err == nil && meta.Instructions != "" {
					cfg.SystemPrompts = append(cfg.SystemPrompts, meta.Instructions)
				}

				// Add each repo as --add-dir so Claude loads their rules and CLAUDE.md
				if p, err := project.Get(cfg.ProjectName); err == nil {
					cfg.AddDirs = nil
					for _, r := range p.Repos {
						cfg.AddDirs = append(cfg.AddDirs, r.Path)
					}
				}

				project.EnsureHooks(cfg.ProjectName, cwembed.Dir())
				project.EnsureAgents(cfg.ProjectName, cwembed.Dir())
				project.SyncCommands(cfg.ProjectName)

				// Sync env files: merge .env.<repo>.global + .env.<repo>.override → repo/.env
				// Then watch for changes and auto-regenerate while Claude runs.
				if p, err := project.Get(cfg.ProjectName); err == nil {
					var repoNames []string
					for _, r := range p.Repos {
						repoNames = append(repoNames, r.Name)
					}
					env.Sync(p.Path, repoNames)
					if w, err := env.Watch(p.Path, repoNames); err == nil {
						envWatcher = w
					}
				}
			}

			// Show spinner during quiet compact (Phase 1 of auto-compact)
			var stopSpinner func()
			if cfg.Quiet {
				stopSpinner = showCompactSpinner()
			}

			// Set CW session ID on launch config
			if cwSession != nil {
				cfg.CWSessionID = cwSession.ID
			}

			if err := claude.Launch(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "claude exited: %v\n", err)
			}

			if stopSpinner != nil {
				stopSpinner()
			}

			if envWatcher != nil {
				envWatcher.Stop()
				envWatcher = nil
			}

			// Capture Claude session ID and update CW session
			if cwSession != nil {
				if claudeID := session.FindClaudeSessionID(cfg.WorkDir); claudeID != "" {
					cwSession.ClaudeSessionID = claudeID
				}
				cwSession.Touch(cfg.WorkDir)
			}

			// After print-mode compact (Phase 1), Claude exits normally (not via reload).
			// Check for pending compact context to continue to Phase 2.
			hasPendingContext := false
			if _, err := os.Stat(paths.CompactContextFile()); err == nil {
				hasPendingContext = true
			}

			if !claude.WasReload() && !hasPendingContext {
				// Session ended normally — mark as completed
				if cwSession != nil {
					cwSession.Status = "completed"
					cwSession.Save(cfg.WorkDir)
				}
				// Clean up yolo plan files and sideband
				if planFiles, err := filepath.Glob(filepath.Join(cfg.WorkDir, ".cw", "yolo", "plan-*.md")); err == nil {
					for _, f := range planFiles {
						os.Remove(f)
					}
				}
				os.Remove(paths.YoloActiveFile())

				// Return to launcher screen for the same project
				if cfg.ProjectName != "" {
					if proj, err := project.Get(cfg.ProjectName); err == nil {
						returnProject = proj
						returnBypass = cfg.SkipPermissions
					}
				}
				break
			}

			// Check for mode switch via sideband file
			if modeData, err := os.ReadFile(paths.ModeOverrideFile()); err == nil {
				modeName := strings.TrimSpace(string(modeData))
				os.Remove(paths.ModeOverrideFile())
				if m, ok := config.GetMode(modeName); ok {
					cfg.Mode = m
					if cwSession != nil {
						cwSession.Mode = modeName
					}
				}
			}

			// Check for permissions switch via sideband file
			if permData, err := os.ReadFile(paths.PermissionsOverrideFile()); err == nil {
				permValue := strings.TrimSpace(string(permData))
				os.Remove(paths.PermissionsOverrideFile())
				cfg.SkipPermissions = permValue == "bypass"
				if cwSession != nil {
					cwSession.SkipPermissions = cfg.SkipPermissions
				}
			}

			// Check for yolo sideband (persists across reloads — do NOT delete)
			if yoloData, err := os.ReadFile(paths.YoloActiveFile()); err == nil {
				planPath := strings.TrimSpace(string(yoloData))
				cfg.SkipPermissions = true
				cfg.YoloActive = true
				cfg.YoloPlanPath = planPath
				// Append yolo execution system prompt
				if yoloPrompt, err := os.ReadFile(filepath.Join(cwembed.ModesDir(), "yolo.md")); err == nil {
					cfg.SystemPrompts = append(cfg.SystemPrompts, string(yoloPrompt))
				}
			} else {
				cfg.YoloActive = false
				cfg.YoloPlanPath = ""
			}

			// Check for auto-compact request via sideband file
			autoCompact := false
			if _, err := os.Stat(paths.AutoCompactFile()); err == nil {
				os.Remove(paths.AutoCompactFile())
				autoCompact = true
			}

			// Check for new-session request via sideband file
			newSession := false
			if _, err := os.Stat(paths.NewSessionFile()); err == nil {
				os.Remove(paths.NewSessionFile())
				newSession = true
				// Mark old session as completed and create a fresh one
				if cwSession != nil {
					cwSession.Status = "completed"
					cwSession.Save(cfg.WorkDir)
				}
				cwSession = session.New(
					generateSessionID(),
					cfg.Mode.Name,
					cfg.SkipPermissions,
					cfg.AutoCompactLimit,
				)
				cwSession.Save(cfg.WorkDir)
			}

			// Reload: clear one-shot fields
			cfg.Prompt = ""
			cfg.Print = false
			cfg.Quiet = false

			// Check for pending compact context from a previous Phase 1
			if ctxData, err := os.ReadFile(paths.CompactContextFile()); err == nil {
				os.Remove(paths.CompactContextFile())
				cfg.Prompt = string(ctxData)
				autoCompact = false // already compacted in Phase 1
			}

			if autoCompact {
				// Auto-compact: run /compact in print mode (exits immediately),
				// then on next reload iteration the compact context file is read as continuation prompt.
				cfg.Prompt = "/compact"
				cfg.Print = true
				cfg.Quiet = true
			}

			if !newSession && cwSession != nil && cwSession.ClaudeSessionID != "" {
				cfg.SessionID = cwSession.ClaudeSessionID
			} else {
				cfg.SessionID = ""
			}
			cfg.AutoSetup = false
			cfg.SystemPrompts = nil
		}
	}
}

// generateSessionID creates a random hex session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// showCompactSpinner displays a terminal spinner while auto-compact runs.
// Returns a stop function that clears the spinner line.
func showCompactSpinner() func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	var once sync.Once
	done := make(chan struct{})

	go func() {
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stdout, "\r\033[K  \033[36m%s\033[0m Compacting context...", frames[i%len(frames)])
				i++
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			fmt.Fprint(os.Stdout, "\r\033[K")
		})
	}
}
