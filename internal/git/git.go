package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func IsRepo(path string) bool {
	info, err := os.Stat(path + "/.git")
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func Branch(repoPath string) string {
	out, err := run(repoPath, "branch", "--show-current")
	if err != nil {
		return "detached"
	}
	b := strings.TrimSpace(out)
	if b == "" {
		return "detached"
	}
	return b
}

func RecentCommits(repoPath string, n int) []string {
	out, err := run(repoPath, "log", "--oneline", "-"+strconv.Itoa(n))
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func LastCommitTime(repoPath string) (time.Time, error) {
	out, err := run(repoPath, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}, err
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

func DirtyFiles(repoPath string) []string {
	out, err := run(repoPath, "status", "--porcelain")
	if err != nil {
		return nil
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func Clone(url, destPath string) error {
	cmd := exec.Command("git", "clone", url, destPath)
	return cmd.Run()
}

func Init(path string) error {
	cmd := exec.Command("git", "init", path)
	return cmd.Run()
}

func run(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}

// CloneProgress reports the status of a single repo clone.
type CloneProgress struct {
	Repo string
	Done bool
	Err  error
}

// ParallelClone clones multiple repos into the project directory concurrently.
// repos are in "owner/repo" format. Uses gh repo clone.
func ParallelClone(projectPath string, repos []string) <-chan CloneProgress {
	ch := make(chan CloneProgress, len(repos))
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			ch <- CloneProgress{Repo: r}

			cmd := exec.Command("gh", "repo", "clone", r, filepath.Join(projectPath, repoName(r)))
			if out, err := cmd.CombinedOutput(); err != nil {
				ch <- CloneProgress{Repo: r, Done: true, Err: fmt.Errorf("%s: %s", err, out)}
			} else {
				ch <- CloneProgress{Repo: r, Done: true}
			}
		}(repo)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

// PullProgress reports the status of a single repo pull.
type PullProgress struct {
	Repo string
	Done bool
	Err  error
}

// PullAll fetches and pulls the default branch for each repo in parallel.
// repos are directory names (not owner/repo). Errors are silently ignored.
func PullAll(projectPath string, repos []string) <-chan PullProgress {
	ch := make(chan PullProgress, len(repos))
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			ch <- PullProgress{Repo: r}

			dir := filepath.Join(projectPath, r)

			defaultBranch := detectDefaultBranch(dir)

			// Fetch latest
			exec.Command("git", "-C", dir, "fetch", "origin", defaultBranch).Run()

			// Update default branch (works even if on another branch)
			exec.Command("git", "-C", dir, "branch", "-f", defaultBranch, "origin/"+defaultBranch).Run()

			// Always report success
			ch <- PullProgress{Repo: r, Done: true}
		}(repo)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

// RepoInfo holds git status for a single repo.
type RepoInfo struct {
	Name          string
	Branch        string
	Clean         bool
	Ahead         int
	Behind        int
	RecentCommits []string
}

// GetRepoInfo gathers git status info for a single repo.
func GetRepoInfo(projectPath, repo string) RepoInfo {
	dir := filepath.Join(projectPath, repo)
	info := RepoInfo{Name: repo}

	// Current branch
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		info.Branch = strings.TrimSpace(string(out))
	}

	// Clean or dirty
	out, _ := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	info.Clean = len(strings.TrimSpace(string(out))) == 0

	// Ahead/behind
	if abOut, err := exec.Command("git", "-C", dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output(); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(abOut)))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &info.Ahead)
			fmt.Sscanf(parts[1], "%d", &info.Behind)
		}
	}

	// Recent commits (last 3)
	if logOut, err := exec.Command("git", "-C", dir, "log", "--oneline", "-3", "--format=%s").Output(); err == nil {
		lines := strings.TrimSpace(string(logOut))
		if lines != "" {
			info.RecentCommits = strings.Split(lines, "\n")
		}
	}

	return info
}

// GetAllRepoInfo gathers git info for all repos in parallel.
func GetAllRepoInfo(projectPath string, repos []string) []RepoInfo {
	infos := make([]RepoInfo, len(repos))
	var wg sync.WaitGroup

	for i, repo := range repos {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()
			infos[idx] = GetRepoInfo(projectPath, r)
		}(i, repo)
	}

	wg.Wait()
	return infos
}

// WorktreeAdd creates a new git worktree at worktreePath with a new branch.
func WorktreeAdd(repoPath, worktreePath, branchName string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}

// WorktreeRemove removes a git worktree.
func WorktreeRemove(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}

// DefaultBranch detects the default branch for a repo.
// Tries origin/HEAD first, then checks for main/master branches, falls back to "main".
func DefaultBranch(repoPath string) string {
	// Try symbolic-ref for origin/HEAD
	if out, err := run(repoPath, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		ref := strings.TrimSpace(out)
		// Extract branch name from "refs/remotes/origin/main"
		if parts := strings.SplitN(ref, "refs/remotes/origin/", 2); len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
	}

	// Check if main branch exists
	if _, err := run(repoPath, "rev-parse", "--verify", "main"); err == nil {
		return "main"
	}

	// Check if master branch exists
	if _, err := run(repoPath, "rev-parse", "--verify", "master"); err == nil {
		return "master"
	}

	return "main"
}

// detectDefaultBranch returns "main" or "master" for a repo.
func detectDefaultBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "refs/heads/main").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return "main"
	}
	return "master"
}

// repoName extracts the repo name from "owner/repo".
func repoName(nameWithOwner string) string {
	for i := len(nameWithOwner) - 1; i >= 0; i-- {
		if nameWithOwner[i] == '/' {
			return nameWithOwner[i+1:]
		}
	}
	return nameWithOwner
}
