package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"owner/repo", "repo"},
		{"org/my-project", "my-project"},
		{"repo-only", "repo-only"},
		{"deep/nested/repo", "repo"},
		{"a/b", "b"},
	}

	for _, tt := range tests {
		got := repoName(tt.input)
		if got != tt.want {
			t.Errorf("repoName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsRepo(t *testing.T) {
	// Not a repo
	tmp := t.TempDir()
	if IsRepo(tmp) {
		t.Error("expected false for non-repo dir")
	}

	// Create a .git dir to make it look like a repo
	gitDir := filepath.Join(tmp, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if !IsRepo(tmp) {
		t.Error("expected true for dir with .git")
	}
}

func TestIsRepoWorktree(t *testing.T) {
	// Worktrees have .git as a file, not a directory
	tmp := t.TempDir()
	gitFile := filepath.Join(tmp, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsRepo(tmp) {
		t.Error("expected true for worktree (.git file)")
	}
}

func TestBranch(t *testing.T) {
	repo := initTestRepo(t)

	branch := Branch(repo)
	// Git defaults to "main" or "master" depending on config
	if branch != "main" && branch != "master" {
		t.Errorf("Branch() = %q, want main or master", branch)
	}
}

func TestBranchDetached(t *testing.T) {
	got := Branch("/nonexistent/path")
	if got != "detached" {
		t.Errorf("Branch(bad path) = %q, want 'detached'", got)
	}
}

func TestDirtyFiles(t *testing.T) {
	repo := initTestRepo(t)

	// Clean repo
	dirty := DirtyFiles(repo)
	if len(dirty) != 0 {
		t.Errorf("expected no dirty files, got %v", dirty)
	}

	// Create an untracked file
	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	dirty = DirtyFiles(repo)
	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty file, got %d", len(dirty))
	}
}

func TestRecentCommits(t *testing.T) {
	repo := initTestRepo(t)

	// Should have 1 commit from initTestRepo
	commits := RecentCommits(repo, 5)
	if len(commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(commits))
	}
}

func TestRecentCommitsEmpty(t *testing.T) {
	commits := RecentCommits("/nonexistent", 5)
	if commits != nil {
		t.Errorf("expected nil for bad path, got %v", commits)
	}
}

func TestInit(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "new-repo")

	if err := Init(repo); err != nil {
		t.Fatal(err)
	}
	if !IsRepo(repo) {
		t.Error("Init did not create a git repo")
	}
}

func TestDetectDefaultBranch(t *testing.T) {
	repo := initTestRepo(t)

	branch := detectDefaultBranch(repo)
	// Should match whatever git init created
	if branch != "main" && branch != "master" {
		t.Errorf("detectDefaultBranch() = %q, want main or master", branch)
	}
}

func TestGetRepoInfo(t *testing.T) {
	tmp := t.TempDir()
	repoName := "myrepo"
	repoPath := filepath.Join(tmp, repoName)
	initTestRepoAt(t, repoPath)

	info := GetRepoInfo(tmp, repoName)
	if info.Name != repoName {
		t.Errorf("Name = %q, want %q", info.Name, repoName)
	}
	if info.Branch == "" {
		t.Error("Branch should not be empty")
	}
	if !info.Clean {
		t.Error("expected clean repo")
	}
	if len(info.RecentCommits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(info.RecentCommits))
	}
}

func TestGetAllRepoInfo(t *testing.T) {
	tmp := t.TempDir()
	repos := []string{"repo-a", "repo-b"}
	for _, name := range repos {
		initTestRepoAt(t, filepath.Join(tmp, name))
	}

	infos := GetAllRepoInfo(tmp, repos)
	if len(infos) != 2 {
		t.Fatalf("expected 2 infos, got %d", len(infos))
	}
	for i, info := range infos {
		if info.Name != repos[i] {
			t.Errorf("infos[%d].Name = %q, want %q", i, info.Name, repos[i])
		}
	}
}

// initTestRepo creates a git repo with one commit in a temp dir.
func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	initTestRepoAt(t, repo)
	return repo
}

func initTestRepoAt(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	cmds := [][]string{
		{"git", "init", path},
		{"git", "-C", path, "config", "user.email", "test@test.com"},
		{"git", "-C", path, "config", "user.name", "Test"},
		{"git", "-C", path, "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %s\n%s", args, err, out)
		}
	}
}
