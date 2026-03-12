package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnvsDir returns the global envs directory (~/.local/share/cw/envs).
func EnvsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cw", "envs")
}

// GlobalPath returns the path to a repo's global env file.
func GlobalPath(repoName string) string {
	return filepath.Join(EnvsDir(), fmt.Sprintf(".env.%s.global", repoName))
}

// OverridePath returns the path to a repo's project-level override env file.
func OverridePath(projectDir, repoName string) string {
	return filepath.Join(projectDir, fmt.Sprintf(".env.%s.override", repoName))
}

// Sync generates .env files for all repos in a project by merging
// .env.<repo>.global with .env.<repo>.override (override wins).
// It also creates symlinks for global env files in the project directory.
func Sync(projectDir string, repoNames []string) error {
	if err := os.MkdirAll(EnvsDir(), 0755); err != nil {
		return err
	}

	for _, name := range repoNames {
		globalFile := GlobalPath(name)
		overrideFile := OverridePath(projectDir, name)

		// Ensure global file exists
		ensureFile(globalFile)
		// Ensure override file exists
		ensureFile(overrideFile)

		// Symlink global env file into the project directory
		symlink := filepath.Join(projectDir, filepath.Base(globalFile))
		// Only recreate if symlink is missing or points to the wrong target
		if target, err := os.Readlink(symlink); err != nil || target != globalFile {
			os.Remove(symlink)
			os.Symlink(globalFile, symlink)
		}

		merged, err := merge(globalFile, overrideFile)
		if err != nil {
			return fmt.Errorf("merge env for %s: %w", name, err)
		}

		target := filepath.Join(projectDir, name, ".env")
		// Only write if content has changed to avoid unnecessary writes
		// and feedback loops with the file watcher.
		existing, _ := os.ReadFile(target)
		if string(existing) == merged {
			continue
		}
		if err := os.WriteFile(target, []byte(merged), 0644); err != nil {
			return fmt.Errorf("write .env for %s: %w", name, err)
		}
	}

	return nil
}

// FilesForProject returns all env file paths related to a project's repos.
// Returns global files first, then override files.
func FilesForProject(projectDir string, repoNames []string) []string {
	var files []string
	for _, name := range repoNames {
		files = append(files, GlobalPath(name))
	}
	for _, name := range repoNames {
		files = append(files, OverridePath(projectDir, name))
	}
	return files
}

func ensureFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(""), 0644)
	}
}

// merge reads two env files and returns the merged content.
// Values from the override file take precedence.
func merge(globalPath, overridePath string) (string, error) {
	vars := make(map[string]string)
	var order []string

	if err := parseInto(globalPath, vars, &order); err != nil {
		return "", err
	}
	if err := parseInto(overridePath, vars, &order); err != nil {
		return "", err
	}

	// Sort for deterministic output
	sort.Strings(order)

	var b strings.Builder
	b.WriteString("# AUTO-GENERATED — do not edit. Modify .env.<repo>.global or .env.<repo>.override instead.\n")
	for _, key := range order {
		fmt.Fprintf(&b, "%s=%s\n", key, vars[key])
	}
	return b.String(), nil
}

// parseInto reads an env file and populates the map and order slice.
// Supports VAR=value and export VAR=value formats.
// Skips comments (#) and blank lines.
func parseInto(path string, vars map[string]string, order *[]string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export " prefix
		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		if _, exists := vars[key]; !exists {
			*order = append(*order, key)
		}
		vars[key] = val
	}

	return scanner.Err()
}
