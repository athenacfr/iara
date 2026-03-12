package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cwembed "github.com/ahtwr/cw/internal/embed"
	"github.com/ahtwr/cw/internal/claude"
	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func uninstall() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	binary := filepath.Join(home, ".local", "bin", "cw")
	dataDir := filepath.Join(home, ".local", "share", "cw")
	projectsDir := config.ProjectsDir()

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

	// If projects dir is outside the data dir (custom CW_PROJECTS_DIR), remove it too
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

func main() {
	if len(os.Args) > 1 && os.Args[1] == "uninstall" {
		uninstall()
		return
	}

	// Extract embedded files (plugins, modes, hooks)
	if err := cwembed.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing embedded files: %v\n", err)
		os.Exit(1)
	}

	config.InitModes(cwembed.ModesDir())

	if err := config.EnsureProjectsDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating projects dir: %v\n", err)
		os.Exit(1)
	}

	for {
		m := tui.NewModel(cwembed.PluginDir())
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

		if cfg.ProjectName != "" {
			sysPrompt, err := project.EnsureSystemPrompt(cfg.ProjectName)
			if err == nil {
				cfg.SystemFiles = append(cfg.SystemFiles, sysPrompt)
			}

			if cfg.Mode.Flag == "--append-system-prompt-file" && cfg.Mode.Value != "" {
				cfg.SystemFiles = append(cfg.SystemFiles, cfg.Mode.Value)
			}

			// Use embedded hooks path
			project.EnsureHooks(cfg.ProjectName, cwembed.Dir())
		}

		if err := claude.Launch(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "claude exited: %v\n", err)
		}
	}
}
