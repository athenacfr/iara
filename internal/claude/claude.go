package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/config"
)

type LaunchConfig struct {
	WorkDir     string
	ProjectName string
	PluginDir   string
	Mode        config.Mode
	Prompt      string   // initial message (positional arg to claude)
	SystemFiles []string // concatenated into one --append-system-prompt-file
}

func Launch(cfg LaunchConfig) error {
	args := []string{"--dangerously-skip-permissions"}

	if cfg.PluginDir != "" {
		args = append(args, "--plugin-dir", cfg.PluginDir)
	}

	// Concatenate system files into a single temp file
	var tmpFile string
	if len(cfg.SystemFiles) > 0 {
		combined, err := combineSystemFiles(cfg.SystemFiles)
		if err == nil && combined != "" {
			tmpFile = combined
			args = append(args, "--append-system-prompt-file", combined)
		}
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
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// Clear screen after Claude exits
	os.Stdout.WriteString("\033[H\033[2J")

	if tmpFile != "" {
		os.Remove(tmpFile)
	}

	return err
}

func combineSystemFiles(files []string) (string, error) {
	var parts []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		parts = append(parts, strings.TrimSpace(string(data)))
	}
	if len(parts) == 0 {
		return "", nil
	}

	tmpDir := filepath.Join(os.TempDir(), "cw")
	os.MkdirAll(tmpDir, 0755)
	tmp, err := os.CreateTemp(tmpDir, "system-prompt-*.md")
	if err != nil {
		return "", err
	}
	content := strings.Join(parts, "\n\n---\n\n")
	tmp.WriteString(content)
	tmp.Close()
	return tmp.Name(), nil
}
