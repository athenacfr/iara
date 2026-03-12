package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/config"
)

// SystemPromptTemplate is the static system prompt bundled with cw.
// The only dynamic part is the repo list, which is injected at launch time.
const systemPromptTemplate = `# CW Project Context

You are working in a cw-managed project directory. The root contains multiple git repositories as subfolders.

## Repos

%s

## Rules

- Do NOT create code files, configs, or dependencies in the project root directory.
- Only CLAUDE.md and .cw-* files belong in the root.
- Each subfolder is an independent git repo with its own git history, dependencies, and configuration.
- When working on code, always operate within the appropriate repo subfolder.
- Do not mix concerns between repos — treat each as isolated.
- The CLAUDE.md in the root contains project-wide instructions and conventions.
`

// BuildSystemPrompt creates a temp file with the system prompt, injecting the
// actual repo list for this project. Returns the temp file path.
func BuildSystemPrompt(name string) (string, error) {
	p, err := Get(name)
	if err != nil {
		return "", err
	}

	var repoList []string
	for _, r := range p.Repos {
		repoList = append(repoList, fmt.Sprintf("- `%s/`", r.Name))
	}

	content := fmt.Sprintf(systemPromptTemplate, strings.Join(repoList, "\n"))

	tmpDir := filepath.Join(os.TempDir(), "cw")
	os.MkdirAll(tmpDir, 0755)
	tmp, err := os.CreateTemp(tmpDir, "cw-system-prompt-*.md")
	if err != nil {
		return "", err
	}
	tmp.WriteString(content)
	tmp.Close()
	return tmp.Name(), nil
}

func HasClaudeMDAt(projectDir string) bool {
	_, err := os.Stat(filepath.Join(projectDir, "CLAUDE.md"))
	return err == nil
}

// Cleanup removes old temp prompt files.
func CleanupTempPrompts() {
	tmpDir := filepath.Join(os.TempDir(), "cw")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "cw-system-prompt-") {
			os.Remove(filepath.Join(tmpDir, e.Name()))
		}
	}
}

// EnsureSystemPrompt is kept for backward compat but now just calls BuildSystemPrompt.
func EnsureSystemPrompt(name string) (string, error) {
	// Clean up old temp files first
	CleanupTempPrompts()
	return BuildSystemPrompt(name)
}

// Remove per-project .cw-system-prompt.md if it exists (cleanup from old versions)
func CleanupLegacySystemPrompt(name string) {
	dir := filepath.Join(config.ProjectsDir(), name)
	os.Remove(filepath.Join(dir, ".cw-system-prompt.md"))
}
