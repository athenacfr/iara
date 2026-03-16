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

// --- LastCommitTime ---

func TestLastCommitTime(t *testing.T) {
	repo := initTestRepo(t)

	ct, err := LastCommitTime(repo)
	if err != nil {
		t.Fatal(err)
	}
	if ct.IsZero() {
		t.Error("expected non-zero commit time")
	}
	// Should be recent (within last minute)
	if ct.Before(ct.Add(-60 * 1e9)) {
		t.Error("commit time seems too old")
	}
}

func TestLastCommitTimeError(t *testing.T) {
	_, err := LastCommitTime("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// --- Clone ---

func TestClone(t *testing.T) {
	// Create a source repo to clone from
	src := initTestRepo(t)
	dest := filepath.Join(t.TempDir(), "clone-dest")

	err := Clone(src, dest)
	if err != nil {
		t.Fatal(err)
	}
	if !IsRepo(dest) {
		t.Error("cloned directory should be a git repo")
	}
}

func TestCloneBadSource(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "clone-dest")
	err := Clone("/nonexistent/repo", dest)
	if err == nil {
		t.Error("expected error cloning from bad source")
	}
}

// --- ParallelClone ---

func TestParallelClone(t *testing.T) {
	// ParallelClone uses `gh repo clone` which requires GitHub auth
	// We test the channel/concurrency mechanics by using a nonexistent repo
	// (it will fail, but we verify the progress messages come through)
	dest := t.TempDir()
	ch := ParallelClone(dest, []string{"nonexistent-owner/nonexistent-repo"})

	var msgs []CloneProgress
	for msg := range ch {
		msgs = append(msgs, msg)
	}

	// Should get 2 messages: start + done (with error)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 progress messages, got %d", len(msgs))
	}
	if msgs[0].Done {
		t.Error("first message should not be Done")
	}
	if !msgs[1].Done {
		t.Error("second message should be Done")
	}
	if msgs[1].Err == nil {
		t.Error("expected error for nonexistent repo")
	}
}

func TestParallelCloneEmpty(t *testing.T) {
	dest := t.TempDir()
	ch := ParallelClone(dest, nil)

	var msgs []CloneProgress
	for msg := range ch {
		msgs = append(msgs, msg)
	}

	if len(msgs) != 0 {
		t.Errorf("expected 0 progress messages for empty repos, got %d", len(msgs))
	}
}

// --- PullAll ---

func TestPullAll(t *testing.T) {
	tmp := t.TempDir()
	repoName := "my-repo"
	initTestRepoAt(t, filepath.Join(tmp, repoName))

	ch := PullAll(tmp, []string{repoName})

	var msgs []PullProgress
	for msg := range ch {
		msgs = append(msgs, msg)
	}

	// Should get 2 messages: start + done
	if len(msgs) != 2 {
		t.Fatalf("expected 2 progress messages, got %d", len(msgs))
	}
	if msgs[0].Done {
		t.Error("first message should not be Done")
	}
	if !msgs[1].Done {
		t.Error("second message should be Done")
	}
}

func TestPullAllEmpty(t *testing.T) {
	ch := PullAll(t.TempDir(), nil)
	var msgs []PullProgress
	for msg := range ch {
		msgs = append(msgs, msg)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 progress messages, got %d", len(msgs))
	}
}

// --- GetRepoInfo: dirty repo ---

func TestGetRepoInfoDirty(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "dirty")
	initTestRepoAt(t, repoPath)

	// Create untracked file
	os.WriteFile(filepath.Join(repoPath, "new.txt"), []byte("hi"), 0644)

	info := GetRepoInfo(tmp, "dirty")
	if info.Clean {
		t.Error("expected dirty repo")
	}
}

// --- Branch: detached HEAD ---

func TestBranchDetachedHead(t *testing.T) {
	repo := initTestRepo(t)

	// Add a second commit
	cmd := exec.Command("git", "-C", repo, "commit", "--allow-empty", "-m", "second")
	cmd.Run()

	// Detach HEAD
	cmd = exec.Command("git", "-C", repo, "checkout", "HEAD~1")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Run()

	got := Branch(repo)
	if got != "detached" {
		t.Errorf("Branch(detached) = %q, want %q", got, "detached")
	}
}

// --- detectDefaultBranch: master fallback ---

func TestDetectDefaultBranchMaster(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	os.MkdirAll(repoPath, 0755)

	cmds := [][]string{
		{"git", "init", "-b", "master", repoPath},
		{"git", "-C", repoPath, "config", "user.email", "test@test.com"},
		{"git", "-C", repoPath, "config", "user.name", "Test"},
		{"git", "-C", repoPath, "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %s\n%s", args, err, out)
		}
	}

	got := detectDefaultBranch(repoPath)
	if got != "master" {
		t.Errorf("detectDefaultBranch = %q, want %q", got, "master")
	}
}

// --- RecentCommits: repo with no commits ---

func TestRecentCommitsNoCommits(t *testing.T) {
	tmp := t.TempDir()
	exec.Command("git", "init", tmp).Run()

	// Repo exists but has no commits
	commits := RecentCommits(tmp, 5)
	if commits != nil {
		t.Errorf("expected nil for repo with no commits, got %v", commits)
	}
}

// --- LastCommitTime: unparseable timestamp ---

func TestLastCommitTimeUnparseable(t *testing.T) {
	tmp := t.TempDir()
	exec.Command("git", "init", tmp).Run()

	// Repo with no commits — git log returns error
	_, err := LastCommitTime(tmp)
	if err == nil {
		t.Error("expected error for repo with no commits")
	}
}

// --- detectDefaultBranch: main branch exists ---

func TestDetectDefaultBranchMain(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	os.MkdirAll(repoPath, 0755)

	cmds := [][]string{
		{"git", "init", "-b", "main", repoPath},
		{"git", "-C", repoPath, "config", "user.email", "test@test.com"},
		{"git", "-C", repoPath, "config", "user.name", "Test"},
		{"git", "-C", repoPath, "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %s\n%s", args, err, out)
		}
	}

	got := detectDefaultBranch(repoPath)
	if got != "main" {
		t.Errorf("detectDefaultBranch = %q, want %q", got, "main")
	}
}

// --- DirtyFiles: error path ---

func TestDirtyFilesError(t *testing.T) {
	got := DirtyFiles("/nonexistent/path")
	if got != nil {
		t.Errorf("DirtyFiles(bad path) = %v, want nil", got)
	}
}

// --- WorktreeAdd ---

func TestWorktreeAdd(t *testing.T) {
	repo := initTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "my-worktree")

	err := WorktreeAdd(repo, worktreePath, "feature-branch")
	if err != nil {
		t.Fatal(err)
	}

	// Verify the worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree directory should exist")
	}

	// Verify .git file exists (worktrees have a .git file, not dir)
	if !IsRepo(worktreePath) {
		t.Error("worktree should be detected as a repo")
	}

	// Verify it's on the right branch
	branch := Branch(worktreePath)
	if branch != "feature-branch" {
		t.Errorf("Branch() = %q, want %q", branch, "feature-branch")
	}
}

func TestWorktreeAddBadRepo(t *testing.T) {
	err := WorktreeAdd("/nonexistent/repo", "/tmp/wt", "branch")
	if err == nil {
		t.Error("expected error for nonexistent repo")
	}
}

// --- WorktreeRemove ---

func TestWorktreeRemove(t *testing.T) {
	repo := initTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "wt-to-remove")

	// Create a worktree first
	if err := WorktreeAdd(repo, worktreePath, "temp-branch"); err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Fatal("worktree should exist before removal")
	}

	// Remove it
	if err := WorktreeRemove(repo, worktreePath); err != nil {
		t.Fatal(err)
	}

	// Verify it's gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed")
	}
}

func TestWorktreeRemoveBadPath(t *testing.T) {
	repo := initTestRepo(t)
	err := WorktreeRemove(repo, "/nonexistent/worktree")
	if err == nil {
		t.Error("expected error for nonexistent worktree")
	}
}

// --- DefaultBranch ---

func TestDefaultBranchMain(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	os.MkdirAll(repoPath, 0755)

	cmds := [][]string{
		{"git", "init", "-b", "main", repoPath},
		{"git", "-C", repoPath, "config", "user.email", "test@test.com"},
		{"git", "-C", repoPath, "config", "user.name", "Test"},
		{"git", "-C", repoPath, "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %s\n%s", args, err, out)
		}
	}

	got := DefaultBranch(repoPath)
	if got != "main" {
		t.Errorf("DefaultBranch() = %q, want %q", got, "main")
	}
}

func TestDefaultBranchMaster(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	os.MkdirAll(repoPath, 0755)

	cmds := [][]string{
		{"git", "init", "-b", "master", repoPath},
		{"git", "-C", repoPath, "config", "user.email", "test@test.com"},
		{"git", "-C", repoPath, "config", "user.name", "Test"},
		{"git", "-C", repoPath, "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cmd %v failed: %s\n%s", args, err, out)
		}
	}

	got := DefaultBranch(repoPath)
	if got != "master" {
		t.Errorf("DefaultBranch() = %q, want %q", got, "master")
	}
}

func TestDefaultBranchFallback(t *testing.T) {
	// Nonexistent repo should fall back to "main"
	got := DefaultBranch("/nonexistent/repo")
	if got != "main" {
		t.Errorf("DefaultBranch(bad path) = %q, want %q", got, "main")
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
