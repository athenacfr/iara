package main

import (
	"bufio"
	"crypto/rand"
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

	"encoding/json"

	"github.com/ahtwr/iara/internal/claude"
	"github.com/ahtwr/iara/internal/config"
	"github.com/ahtwr/iara/internal/devlog"
	iaraembed "github.com/ahtwr/iara/internal/embed"
	"github.com/ahtwr/iara/internal/env"
	"github.com/ahtwr/iara/internal/paths"
	"github.com/ahtwr/iara/internal/project"
	"github.com/ahtwr/iara/internal/session"
	"github.com/ahtwr/iara/internal/task"
	"github.com/ahtwr/iara/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func uninstall() {
	binary := filepath.Join(paths.BinDir(), "iara")
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
	pidStr := os.Getenv("IARA_PID")
	if pidStr == "" {
		fmt.Fprintln(os.Stderr, "IARA_PID not set — not running inside iara")
		os.Exit(1)
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid IARA_PID: %s\n", pidStr)
		os.Exit(1)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find iara process %d: %v\n", pid, err)
		os.Exit(1)
	}
	if err := proc.Signal(syscall.SIGUSR1); err != nil {
		fmt.Fprintf(os.Stderr, "cannot signal iara process: %v\n", err)
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
	config.InitModes()
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
	projectDir := os.Getenv("IARA_PROJECT_DIR")
	if projectDir == "" {
		fmt.Fprintln(os.Stderr, "IARA_PROJECT_DIR not set — not running inside iara")
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
	projectDir := os.Getenv("IARA_PROJECT_DIR")
	if projectDir == "" {
		fmt.Fprintln(os.Stderr, "IARA_PROJECT_DIR not set — not running inside iara")
		os.Exit(1)
	}
	matches, _ := filepath.Glob(filepath.Join(projectDir, ".iara", "yolo", "plan-*.md"))
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, "No yolo plan found in "+filepath.Join(projectDir, ".iara", "yolo"))
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

func internalSaveTask(jsonStr string) {
	projectDir := os.Getenv("IARA_PROJECT_DIR")
	if projectDir == "" {
		fmt.Fprintln(os.Stderr, "IARA_PROJECT_DIR not set — not running inside iara")
		os.Exit(1)
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Branch      string `json:"branch"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
		fmt.Fprintf(os.Stderr, "save-task: invalid JSON: %v\n", err)
		os.Exit(1)
	}
	if input.Name == "" || input.Branch == "" {
		fmt.Fprintln(os.Stderr, "save-task: name and branch are required")
		os.Exit(1)
	}

	t := task.New(input.Name, input.Description, input.Branch)
	if err := task.Save(projectDir, t); err != nil {
		fmt.Fprintf(os.Stderr, "save-task: %v\n", err)
		os.Exit(1)
	}

	// Discover repos in the project
	p, err := project.Get(filepath.Base(projectDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "save-task: cannot get project: %v\n", err)
		os.Exit(1)
	}
	var repoNames []string
	for _, r := range p.Repos {
		repoNames = append(repoNames, r.Name)
	}

	if err := task.SetupWorktree(projectDir, t, repoNames); err != nil {
		fmt.Fprintf(os.Stderr, "save-task: worktree setup failed: %v\n", err)
		os.Exit(1)
	}

	// Write task ID to sideband file so the reload loop can switch to the worktree
	if err := os.WriteFile(paths.TaskSwitchFile(), []byte(t.ID), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "save-task: warning: cannot write task-switch file: %v\n", err)
	}

	fmt.Printf("Task '%s' created (branch: %s)\n", t.Name, t.Branch)
}

func internalFinishTask() {
	projectDir := os.Getenv("IARA_PROJECT_DIR")
	taskID := os.Getenv("IARA_TASK_ID")
	if projectDir == "" || taskID == "" {
		fmt.Fprintln(os.Stderr, "IARA_PROJECT_DIR and IARA_TASK_ID must be set")
		os.Exit(1)
	}

	t, err := task.Load(projectDir, taskID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "finish-task: cannot load task: %v\n", err)
		os.Exit(1)
	}

	// Discover repos in the project
	p, err := project.Get(filepath.Base(projectDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "finish-task: cannot get project: %v\n", err)
		os.Exit(1)
	}
	var repoNames []string
	for _, r := range p.Repos {
		repoNames = append(repoNames, r.Name)
	}

	if err := task.RemoveWorktree(projectDir, t, repoNames); err != nil {
		fmt.Fprintf(os.Stderr, "finish-task: worktree removal failed: %v\n", err)
		os.Exit(1)
	}

	if err := task.SetStatus(projectDir, taskID, "completed"); err != nil {
		fmt.Fprintf(os.Stderr, "finish-task: cannot update status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task '%s' completed\n", t.Name)
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
				fmt.Fprintln(os.Stderr, "usage: iara internal mode-switch <mode>")
				os.Exit(1)
			}
			internalModeSwitch(os.Args[3])
			return
		case "permissions-switch":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: iara internal permissions-switch <bypass|normal>")
				os.Exit(1)
			}
			internalPermissionsSwitch(os.Args[3])
			return
		case "summarize":
			if len(os.Args) < 6 {
				fmt.Fprintln(os.Stderr, "usage: iara internal summarize <sessionID> <workDir> <projectDir>")
				os.Exit(1)
			}
			session.RunSummarize(os.Args[3], os.Args[4], os.Args[5])
			return
		case "save-metadata":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: iara internal save-metadata '<json>'")
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
		case "save-task":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: iara internal save-task '<json>'")
				os.Exit(1)
			}
			internalSaveTask(os.Args[3])
			return
		case "finish-task":
			internalFinishTask()
			return
		}
	}

	// Extract embedded files (plugins, modes, hooks)
	if err := iaraembed.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing embedded files: %v\n", err)
		os.Exit(1)
	}

	config.InitModes()

	if err := paths.EnsureProjectsDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating projects dir: %v\n", err)
		os.Exit(1)
	}

	var returnProject *project.Project
	var returnBypass bool

	for {
		var m tui.Model
		if returnProject != nil {
			m = tui.NewModelWithProject(iaraembed.PluginDir(), returnProject, returnBypass)
			returnProject = nil
		} else {
			m = tui.NewModel(iaraembed.PluginDir())
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

		// Resolve sessions directory — task-scoped if available, fallback to project-level
		sessionsDir := cfg.SessionsDir
		if sessionsDir == "" {
			sessionsDir = filepath.Join(cfg.WorkDir, ".iara", "sessions")
		}

		// Create or load the CW session for this launch
		var cwSession *session.Session
		var isNewSession bool
		if cfg.ResumeSessionID != "" {
			// Resuming a specific session — load it and restore state
			if s, err := session.Load(sessionsDir, cfg.ResumeSessionID); err == nil {
				cwSession = s
				// Restore mode from session
				if m, ok := config.GetMode(s.Mode); ok {
					cfg.Mode = m
				}
				cfg.SkipPermissions = s.SkipPermissions
				// Same ID is used for both iara and Claude --resume
				cfg.ResumeSessionID = s.ID
			}
		}
		if cwSession == nil {
			// New session — generate a UUID shared with Claude via --session-id
			isNewSession = true
			cwSession = session.New(
				generateSessionID(),
				cfg.Mode.Name,
				cfg.SkipPermissions,
			)
			cwSession.Save(sessionsDir)
		}

		// Channel closed when background summary generation completes.
		// Used to keep the compact spinner alive until both finish.
		var summaryDone <-chan struct{}

		// Reload loop: re-sync and re-launch on reload signal
		for {
			// Load global settings for this launch iteration
			globalSettings := project.LoadGlobalSettings()
			cfg.LoadSubprojectRules = globalSettings.LoadSubprojectRules

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
				// When in a task worktree, use worktree repo paths instead of main repos
				cfg.AddDirs = nil
				if cfg.TaskID != "" {
					// Worktree mode: discover repos in worktree base
					if entries, err := os.ReadDir(cfg.WorkDir); err == nil {
						for _, e := range entries {
							if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
								repoPath := filepath.Join(cfg.WorkDir, e.Name())
								if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
									cfg.AddDirs = append(cfg.AddDirs, repoPath)
								}
							}
						}
					}
				} else if p, err := project.Get(cfg.ProjectName); err == nil {
					for _, r := range p.Repos {
						cfg.AddDirs = append(cfg.AddDirs, r.Path)
					}
				}

				if globalSettings.EnableHooks {
					project.EnsureHooks(cfg.ProjectName, iaraembed.Dir())
				}
				project.EnsureAgents(cfg.ProjectName, iaraembed.Dir())
				project.SyncCommands(cfg.ProjectName)

				// Sync env files: merge .env.<repo>.global + .env.<repo>.override → repo/.env
				// When in a task worktree, sync to the worktree base (cfg.WorkDir).
				// Otherwise, sync to the project root.
				envSyncDir := cfg.WorkDir
				if p, err := project.Get(cfg.ProjectName); err == nil {
					var repoNames []string
					// Discover repos from the actual WorkDir (worktree or project root)
					if entries, err := os.ReadDir(envSyncDir); err == nil {
						for _, e := range entries {
							if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
								repoPath := filepath.Join(envSyncDir, e.Name())
								if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
									repoNames = append(repoNames, e.Name())
								}
							}
						}
					}
					if len(repoNames) == 0 {
						// Fallback to project repos
						for _, r := range p.Repos {
							repoNames = append(repoNames, r.Name)
						}
						envSyncDir = p.Path
					}
					env.Sync(envSyncDir, repoNames)
					if w, err := env.Watch(envSyncDir, repoNames); err == nil {
						envWatcher = w
					}
				}
			}

			// Task-scoped base dir for dev logs and config.
			// sessionsDir is .iara/tasks/<id>/sessions/ — parent is the task dir.
			// For non-task (legacy fallback), sessionsDir is .iara/sessions/ — parent is .iara/
			taskBaseDir := filepath.Dir(sessionsDir)

			// Manage dev logs: ensure dir exists, truncate oversized logs
			devlog.EnsureDir(taskBaseDir)
			devlog.TruncateOversized(taskBaseDir)

			// Show spinner during quiet compact (Phase 1 of auto-compact)
			var stopSpinner func()
			if cfg.Quiet {
				stopSpinner = showCompactSpinner()
			}

			// Set session IDs on launch config
			if cwSession != nil {
				cfg.IARASessionID = cwSession.ID
				if isNewSession {
					cfg.NewSessionID = cwSession.ID
					isNewSession = false
				}
			}

			if err := claude.Launch(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "claude exited: %v\n", err)
			}

			// Wait for background summary to finish before stopping spinner,
			// so the spinner covers both compact and summary generation.
			if summaryDone != nil {
				<-summaryDone
				summaryDone = nil
			}
			if stopSpinner != nil {
				stopSpinner()
			}

			if envWatcher != nil {
				envWatcher.Stop()
				envWatcher = nil
			}

			if cwSession != nil {
				cwSession.Touch(sessionsDir)
			}

			// After print-mode compact (Phase 1), Claude exits normally (not via reload).
			// Check for pending compact context to continue to Phase 2.
			hasPendingContext := false
			if _, err := os.Stat(paths.CompactContextFile()); err == nil {
				hasPendingContext = true
			}

			if !claude.WasReload() && !hasPendingContext {
				// Session ended normally — mark as completed and extract summary
				if cwSession != nil {
					cwSession.Status = "completed"
					cwSession.Save(sessionsDir)
					if cwSession.Summary == "" {
						session.GenerateSummaryAsync(cwSession.ID, sessionsDir, cfg.WorkDir)
					}
				}
				// Clean up dev logs and yolo plan files
				devlog.Cleanup(taskBaseDir)
				if planFiles, err := filepath.Glob(filepath.Join(cfg.WorkDir, ".iara", "yolo", "plan-*.md")); err == nil {
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
				isNewSession = true
				// Mark old session as completed and generate summary in background
				if cwSession != nil {
					cwSession.Status = "completed"
					cwSession.Save(sessionsDir)
					if cwSession.Summary == "" {
						summaryDone = session.GenerateSummaryBackground(cwSession.ID, sessionsDir, cfg.WorkDir)
					}
				}
				cwSession = session.New(
					generateSessionID(),
					cfg.Mode.Name,
					cfg.SkipPermissions,
				)
				cwSession.Save(sessionsDir)
			}

			// Check for task-switch sideband file (written by save-task)
			if taskIDData, err := os.ReadFile(paths.TaskSwitchFile()); err == nil {
				os.Remove(paths.TaskSwitchFile())
				taskID := strings.TrimSpace(string(taskIDData))
				projectDir := cfg.WorkDir
				if t, err := task.Load(projectDir, taskID); err == nil {
					cfg.TaskID = t.ID
					cfg.TaskName = t.Name
					cfg.WorkDir = task.WorktreeBase(projectDir, t.Name)
					sessionsDir = task.SessionsDir(projectDir, t.ID)
					cfg.SessionsDir = sessionsDir
					// Restore permissions from global settings (new-task forces bypass)
					s := project.LoadGlobalSettings()
					cfg.SkipPermissions = s.BypassPermissions
					// Re-save the new session in the task-scoped sessions dir
					if cwSession != nil {
						cwSession.SkipPermissions = cfg.SkipPermissions
						cwSession.Save(sessionsDir)
					}
				}
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

			if !newSession && cwSession != nil {
				cfg.ResumeSessionID = cwSession.ID
			} else {
				cfg.ResumeSessionID = ""
			}
			cfg.AutoSetup = false
			cfg.SystemPrompts = nil
		}
	}
}

// generateSessionID creates a random UUID v4 session ID.
// This ID is shared between iara and Claude via --session-id.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
