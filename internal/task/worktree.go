package task

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/git"
)

// SetupWorktree creates git worktrees for each repo and sets up the Claude context files.
func SetupWorktree(projectDir string, t Task, repoNames []string) error {
	wtBase := WorktreeBase(projectDir, t.Name)

	// Create worktrees for each repo
	for _, repo := range repoNames {
		repoPath := filepath.Join(projectDir, repo)
		wtPath := filepath.Join(wtBase, repo)
		if err := git.WorktreeAdd(repoPath, wtPath, t.Branch); err != nil {
			return fmt.Errorf("worktree add %s: %w", repo, err)
		}
	}

	// Create .claude/rules/ directory
	rulesDir := filepath.Join(wtBase, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}

	// Symlink PROJECT.md -> project CLAUDE.md (relative path)
	symlinkPath := filepath.Join(rulesDir, "PROJECT.md")
	symlinkTarget := filepath.Join("..", "..", "..", "..", "CLAUDE.md")
	if err := os.Symlink(symlinkTarget, symlinkPath); err != nil {
		return fmt.Errorf("symlink PROJECT.md: %w", err)
	}

	// Write task CLAUDE.md
	claudeMD := filepath.Join(wtBase, "CLAUDE.md")
	content := fmt.Sprintf("# Task: %s\n\n%s\n", t.Name, t.Description)
	if err := os.WriteFile(claudeMD, []byte(content), 0644); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}

	return nil
}

// RemoveWorktree removes git worktrees for each repo and cleans up directories.
func RemoveWorktree(projectDir string, t Task, repoNames []string) error {
	wtBase := WorktreeBase(projectDir, t.Name)

	// Remove worktrees for each repo
	for _, repo := range repoNames {
		repoPath := filepath.Join(projectDir, repo)
		wtPath := filepath.Join(wtBase, repo)
		if err := git.WorktreeRemove(repoPath, wtPath); err != nil {
			return fmt.Errorf("worktree remove %s: %w", repo, err)
		}
	}

	// Remove the worktree base directory entirely
	if err := os.RemoveAll(wtBase); err != nil {
		return fmt.Errorf("remove worktree dir: %w", err)
	}

	// If .worktrees/ is now empty, remove it
	worktreesDir := filepath.Join(projectDir, ".worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err == nil && len(entries) == 0 {
		os.Remove(worktreesDir)
	}

	return nil
}
