package project

import (
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/paths"
)

// EnsureAgents copies agent definition files from the cw data directory
// to the project's .claude/agents/ directory.
func EnsureAgents(name, cwRoot string) error {
	projectDir := filepath.Join(paths.ProjectsDir(), name)
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	srcDir := filepath.Join(cwRoot, "agents")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		// No agents directory — not an error, just nothing to install
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(agentsDir, entry.Name()), data, 0644); err != nil {
			return err
		}
	}

	return nil
}
