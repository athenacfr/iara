package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureHooksCreatesSettings(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var cfg hooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid settings JSON: %v", err)
	}

	if _, ok := cfg.Hooks["PreToolUse"]; !ok {
		t.Error("expected PreToolUse hooks")
	}
	if _, ok := cfg.Hooks["PostToolUse"]; !ok {
		t.Error("expected PostToolUse hooks")
	}
}

func TestEnsureHooksCreatesGitignore(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), ".*") {
		t.Error("expected .gitignore to contain '.*'")
	}
}

func TestEnsureHooksDoesNotOverwriteGitignore(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	// Create existing .gitignore
	existing := "node_modules/\n*.log\n"
	os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(existing), 0644)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Errorf("existing .gitignore was modified: got %q, want %q", string(data), existing)
	}
}

func TestEnsureHooksIdempotent(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	if err := EnsureHooks("test-project", cwRoot); err != nil {
		t.Fatal(err)
	}
	if err := EnsureHooks("test-project", cwRoot); err != nil {
		t.Fatal(err)
	}

	// Should still have valid JSON
	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg hooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid settings after double EnsureHooks: %v", err)
	}
}

func TestEnsureHooksMatcherContent(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	EnsureHooks("test-project", cwRoot)

	data, _ := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	var cfg hooksConfig
	json.Unmarshal(data, &cfg)

	// PreToolUse should have Edit|Write matcher
	preGroups := cfg.Hooks["PreToolUse"]
	if len(preGroups) == 0 {
		t.Fatal("expected PreToolUse groups")
	}
	if preGroups[0].Matcher != "Edit|Write" {
		t.Errorf("PreToolUse matcher = %q, want %q", preGroups[0].Matcher, "Edit|Write")
	}

	// PostToolUse should have * matcher
	postGroups := cfg.Hooks["PostToolUse"]
	if len(postGroups) == 0 {
		t.Fatal("expected PostToolUse groups")
	}
	if postGroups[0].Matcher != "*" {
		t.Errorf("PostToolUse matcher = %q, want %q", postGroups[0].Matcher, "*")
	}
}

func TestEnsureHooksContainsStopHook(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cfg hooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid settings JSON: %v", err)
	}

	stopGroups, ok := cfg.Hooks["Stop"]
	if !ok {
		t.Fatal("expected Stop hooks")
	}
	if len(stopGroups) == 0 {
		t.Fatal("expected Stop hook groups")
	}
	if len(stopGroups[0].Hooks) == 0 {
		t.Fatal("expected hooks in Stop group")
	}
	if !strings.Contains(stopGroups[0].Hooks[0].Command, "yolo-stop.sh") {
		t.Errorf("Stop hook command = %q, expected to contain yolo-stop.sh", stopGroups[0].Hooks[0].Command)
	}
}

func TestEnsureHooksQuotesPathsWithSpaces(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	// Simulate macOS-style path with spaces
	cwRoot := filepath.Join(t.TempDir(), "Library", "Application Support", "iara")
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Commands must be shell-quoted to handle spaces
	if !strings.Contains(content, "'") {
		t.Error("hook commands should be shell-quoted for paths with spaces")
	}
	// The path should appear quoted, not bare
	if strings.Contains(content, "Application Support/iara/hooks/auto-compact.sh\"") &&
		!strings.Contains(content, "'") {
		t.Error("bare path with spaces would break shell execution")
	}

	// Verify the JSON is still valid
	var cfg hooksConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid settings JSON: %v", err)
	}

	// Verify hook commands contain quoted paths
	for _, groups := range cfg.Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				if !strings.HasPrefix(h.Command, "'") || !strings.HasSuffix(h.Command, "'") {
					t.Errorf("hook command not shell-quoted: %s", h.Command)
				}
			}
		}
	}
}

func TestEnsureHooksContainsPreWriteGuard(t *testing.T) {
	dir := setTestProjectsDir(t)
	projectDir := filepath.Join(dir, "test-project")
	os.MkdirAll(projectDir, 0755)

	cwRoot := t.TempDir()
	os.MkdirAll(filepath.Join(cwRoot, "hooks"), 0755)

	err := EnsureHooks("test-project", cwRoot)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "pre-write-guard.sh") {
		t.Error("settings.json should reference pre-write-guard.sh")
	}
	if !strings.Contains(content, "auto-compact.sh") {
		t.Error("settings.json should reference auto-compact.sh")
	}
}
