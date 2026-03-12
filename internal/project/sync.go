package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/config"
)

const commandHeader = `> **Repo: %s** — Run all operations for this command within the ` + "`%s/`" + ` directory.

`

// managedMarker identifies files created by SyncCommands so we can clean them up
// without removing user-created commands.
const managedMarker = "<!-- cw:synced -->\n"

// SyncCommands copies each repo's .claude/commands/ files into the project's
// .claude/commands/ directory. Each copied command is prefixed with a header
// instructing Claude to run in the specific repo. For multi-repo projects,
// filenames are prefixed with the repo name (e.g., repo1--deploy.md).
func SyncCommands(name string) error {
	p, err := Get(name)
	if err != nil {
		return err
	}

	projectDir := filepath.Join(config.ProjectsDir(), name)
	destDir := filepath.Join(projectDir, ".claude", "commands")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Remove previously synced files (identified by marker)
	entries, _ := os.ReadDir(destDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(destDir, e.Name())
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), managedMarker) {
			os.Remove(path)
		}
	}

	singleRepo := len(p.Repos) == 1

	for _, r := range p.Repos {
		commandsDir := filepath.Join(r.Path, ".claude", "commands")
		files, err := os.ReadDir(commandsDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}

			src := filepath.Join(commandsDir, f.Name())
			content, err := os.ReadFile(src)
			if err != nil {
				continue
			}

			// Name as repo:command (e.g., app:deploy.md → /app:deploy)
			var destName string
			if singleRepo {
				destName = f.Name()
			} else {
				name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
				ext := filepath.Ext(f.Name())
				destName = r.Name + ":" + name + ext
			}

			header := fmt.Sprintf(commandHeader, r.Name, r.Name)
			out := managedMarker + header + string(content)

			os.WriteFile(filepath.Join(destDir, destName), []byte(out), 0644)
		}
	}

	return nil
}
