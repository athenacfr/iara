package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tk := New("add-auth", "Add authentication", "feat/add-auth")

	if tk.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	// UUID v4 format: 8-4-4-4-12
	parts := strings.Split(tk.ID, "-")
	if len(parts) != 5 {
		t.Fatalf("expected UUID format, got %q", tk.ID)
	}

	if tk.Name != "add-auth" {
		t.Fatalf("expected name 'add-auth', got %q", tk.Name)
	}
	if tk.Description != "Add authentication" {
		t.Fatalf("expected description 'Add authentication', got %q", tk.Description)
	}
	if tk.Branch != "feat/add-auth" {
		t.Fatalf("expected branch 'feat/add-auth', got %q", tk.Branch)
	}
	if tk.Status != "active" {
		t.Fatalf("expected status 'active', got %q", tk.Status)
	}
	if tk.CreatedAt == "" {
		t.Fatal("expected non-empty CreatedAt")
	}
	if tk.LastActive == "" {
		t.Fatal("expected non-empty LastActive")
	}
	if tk.CreatedAt != tk.LastActive {
		t.Fatal("expected CreatedAt == LastActive on new task")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	tk := New("test-task", "A test task", "feat/test")
	if err := Save(dir, tk); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir, tk.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != tk.ID {
		t.Fatalf("ID mismatch: %q != %q", loaded.ID, tk.ID)
	}
	if loaded.Name != tk.Name {
		t.Fatalf("Name mismatch: %q != %q", loaded.Name, tk.Name)
	}
	if loaded.Description != tk.Description {
		t.Fatalf("Description mismatch")
	}
	if loaded.Branch != tk.Branch {
		t.Fatalf("Branch mismatch")
	}
	if loaded.Status != tk.Status {
		t.Fatalf("Status mismatch")
	}
	if loaded.CreatedAt != tk.CreatedAt {
		t.Fatalf("CreatedAt mismatch")
	}
	if loaded.LastActive != tk.LastActive {
		t.Fatalf("LastActive mismatch")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	// Create 3 tasks with different LastActive times
	t1 := New("task-a", "First", "branch-a")
	t1.LastActive = "2024-01-01T00:00:00Z"
	if err := Save(dir, t1); err != nil {
		t.Fatalf("Save t1: %v", err)
	}

	t2 := New("task-b", "Second", "branch-b")
	t2.LastActive = "2024-01-03T00:00:00Z"
	if err := Save(dir, t2); err != nil {
		t.Fatalf("Save t2: %v", err)
	}

	t3 := New("task-c", "Third", "branch-c")
	t3.LastActive = "2024-01-02T00:00:00Z"
	if err := Save(dir, t3); err != nil {
		t.Fatalf("Save t3: %v", err)
	}

	// Create a "default" directory that should be skipped
	defaultDir := filepath.Join(dir, ".cw", "tasks", "default", "sessions")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatalf("create default dir: %v", err)
	}

	tasks, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Should be sorted by LastActive desc: t2, t3, t1
	if tasks[0].ID != t2.ID {
		t.Fatalf("expected first task to be t2, got %q", tasks[0].Name)
	}
	if tasks[1].ID != t3.ID {
		t.Fatalf("expected second task to be t3, got %q", tasks[1].Name)
	}
	if tasks[2].ID != t1.ID {
		t.Fatalf("expected third task to be t1, got %q", tasks[2].Name)
	}
}

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	tasks, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if tasks != nil {
		t.Fatalf("expected nil, got %v", tasks)
	}
}

func TestTouch(t *testing.T) {
	dir := t.TempDir()

	tk := New("touch-test", "desc", "branch")
	tk.LastActive = "2020-01-01T00:00:00Z"
	if err := Save(dir, tk); err != nil {
		t.Fatalf("Save: %v", err)
	}

	before := tk.LastActive

	// Small delay to ensure timestamp differs
	time.Sleep(10 * time.Millisecond)

	if err := Touch(dir, tk.ID); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	loaded, err := Load(dir, tk.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.LastActive == before {
		t.Fatal("expected LastActive to be updated after Touch")
	}
	if loaded.LastActive <= before {
		t.Fatal("expected LastActive to be newer")
	}
}

func TestSetStatus(t *testing.T) {
	dir := t.TempDir()

	tk := New("status-test", "desc", "branch")
	if err := Save(dir, tk); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := SetStatus(dir, tk.ID, "completed"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	loaded, err := Load(dir, tk.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Status != "completed" {
		t.Fatalf("expected status 'completed', got %q", loaded.Status)
	}
}

func TestSessionsDir(t *testing.T) {
	got := SessionsDir("/projects/myproj", "abc-123")
	want := filepath.Join("/projects/myproj", ".cw", "tasks", "abc-123", "sessions")
	if got != want {
		t.Fatalf("SessionsDir: got %q, want %q", got, want)
	}
}

func TestDefaultSessionsDir(t *testing.T) {
	got := DefaultSessionsDir("/projects/myproj")
	want := filepath.Join("/projects/myproj", ".cw", "tasks", "default", "sessions")
	if got != want {
		t.Fatalf("DefaultSessionsDir: got %q, want %q", got, want)
	}
}

func TestTaskDir(t *testing.T) {
	got := TaskDir("/projects/myproj", "abc-123")
	want := filepath.Join("/projects/myproj", ".cw", "tasks", "abc-123")
	if got != want {
		t.Fatalf("TaskDir: got %q, want %q", got, want)
	}
}

func TestWorktreeBase(t *testing.T) {
	got := WorktreeBase("/projects/myproj", "add-auth")
	want := filepath.Join("/projects/myproj", ".worktrees", "add-auth")
	if got != want {
		t.Fatalf("WorktreeBase: got %q, want %q", got, want)
	}
}

// initGitRepo creates a git repo with an initial commit.
func initGitRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	cmds := [][]string{
		{"git", "init", path},
		{"git", "-C", path, "config", "user.email", "test@test.com"},
		{"git", "-C", path, "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("cmd %v: %s: %s", args, err, out)
		}
	}
	// Create a file and commit so branch exists
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmds = [][]string{
		{"git", "-C", path, "add", "."},
		{"git", "-C", path, "commit", "-m", "init"},
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("cmd %v: %s: %s", args, err, out)
		}
	}
}

func TestSetupWorktree(t *testing.T) {
	dir := t.TempDir()

	// Create a real git repo
	repoName := "myrepo"
	repoPath := filepath.Join(dir, repoName)
	initGitRepo(t, repoPath)

	// Write a project CLAUDE.md
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project rules\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tk := New("add-auth", "Add authentication support", "feat/add-auth")

	if err := SetupWorktree(dir, tk, []string{repoName}); err != nil {
		t.Fatalf("SetupWorktree: %v", err)
	}

	wtBase := WorktreeBase(dir, tk.Name)

	// Verify worktree directory exists
	wtRepo := filepath.Join(wtBase, repoName)
	if _, err := os.Stat(wtRepo); err != nil {
		t.Fatalf("worktree repo dir should exist: %v", err)
	}

	// Verify CLAUDE.md exists with task content
	claudeMD := filepath.Join(wtBase, "CLAUDE.md")
	data, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Task: add-auth") {
		t.Fatalf("CLAUDE.md should contain task name, got: %s", content)
	}
	if !strings.Contains(content, "Add authentication support") {
		t.Fatalf("CLAUDE.md should contain description, got: %s", content)
	}

	// Verify .claude/rules/PROJECT.md symlink exists
	symlinkPath := filepath.Join(wtBase, ".claude", "rules", "PROJECT.md")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("readlink PROJECT.md: %v", err)
	}
	expectedTarget := filepath.Join("..", "..", "..", "..", "CLAUDE.md")
	if target != expectedTarget {
		t.Fatalf("symlink target: got %q, want %q", target, expectedTarget)
	}

	// Verify the symlink resolves correctly
	resolved, err := os.ReadFile(symlinkPath)
	if err != nil {
		t.Fatalf("read through symlink: %v", err)
	}
	if string(resolved) != "# Project rules\n" {
		t.Fatalf("symlink content mismatch: %q", string(resolved))
	}
}

func TestRemoveWorktree(t *testing.T) {
	dir := t.TempDir()

	// Create a real git repo
	repoName := "myrepo"
	repoPath := filepath.Join(dir, repoName)
	initGitRepo(t, repoPath)

	// Write a project CLAUDE.md
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project rules\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tk := New("remove-test", "Test removal", "feat/remove-test")

	if err := SetupWorktree(dir, tk, []string{repoName}); err != nil {
		t.Fatalf("SetupWorktree: %v", err)
	}

	wtBase := WorktreeBase(dir, tk.Name)

	// Verify it exists first
	if _, err := os.Stat(wtBase); err != nil {
		t.Fatalf("worktree should exist before remove: %v", err)
	}

	if err := RemoveWorktree(dir, tk, []string{repoName}); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	// Verify worktree directory is gone
	if _, err := os.Stat(wtBase); !os.IsNotExist(err) {
		t.Fatal("worktree directory should not exist after remove")
	}

	// Verify .worktrees/ is removed when empty
	worktreesDir := filepath.Join(dir, ".worktrees")
	if _, err := os.Stat(worktreesDir); !os.IsNotExist(err) {
		t.Fatal(".worktrees/ should be removed when empty")
	}
}
