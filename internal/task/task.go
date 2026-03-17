package task

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Task represents a task within a project, each with its own worktree and sessions.
type Task struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Branch      string `json:"branch"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	LastActive  string `json:"last_active"`
}

// newUUID generates a UUID v4 using crypto/rand.
func newUUID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// New creates a new Task with a generated UUID, timestamps, and status "active".
func New(name, description, branch string) Task {
	now := time.Now().UTC().Format(time.RFC3339)
	return Task{
		ID:          newUUID(),
		Name:        name,
		Description: description,
		Branch:      branch,
		Status:      "active",
		CreatedAt:   now,
		LastActive:  now,
	}
}

// taskPath returns the path to a task's JSON file.
func taskPath(projectDir, taskID string) string {
	return filepath.Join(projectDir, ".iara", "tasks", taskID, "task.json")
}

// Save writes the task to <projectDir>/.iara/tasks/<id>/task.json.
func Save(projectDir string, t Task) error {
	dir := filepath.Dir(taskPath(projectDir, t.ID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create task dir: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	return os.WriteFile(taskPath(projectDir, t.ID), data, 0644)
}

// Load reads a task from disk.
func Load(projectDir, taskID string) (Task, error) {
	data, err := os.ReadFile(taskPath(projectDir, taskID))
	if err != nil {
		return Task{}, err
	}

	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return Task{}, fmt.Errorf("unmarshal task: %w", err)
	}
	return t, nil
}

// List returns all tasks in the project, sorted by LastActive descending.
// The "default" directory is skipped.
func List(projectDir string) ([]Task, error) {
	tasksDir := filepath.Join(projectDir, ".iara", "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tasks []Task
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.EqualFold(e.Name(), "default") {
			continue
		}

		t, err := Load(projectDir, e.Name())
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].LastActive > tasks[j].LastActive
	})

	return tasks, nil
}

// Touch updates the task's LastActive timestamp and saves.
func Touch(projectDir, taskID string) error {
	t, err := Load(projectDir, taskID)
	if err != nil {
		return err
	}
	t.LastActive = time.Now().UTC().Format(time.RFC3339)
	return Save(projectDir, t)
}

// SetStatus loads a task, changes its status, and saves.
func SetStatus(projectDir, taskID, status string) error {
	t, err := Load(projectDir, taskID)
	if err != nil {
		return err
	}
	t.Status = status
	return Save(projectDir, t)
}

// SessionsDir returns the sessions directory for a task.
func SessionsDir(projectDir, taskID string) string {
	return filepath.Join(projectDir, ".iara", "tasks", taskID, "sessions")
}

// DefaultSessionsDir returns the sessions directory for the default (non-task) context.
func DefaultSessionsDir(projectDir string) string {
	return filepath.Join(projectDir, ".iara", "tasks", "default", "sessions")
}

// TaskDir returns the directory for a task.
func TaskDir(projectDir, taskID string) string {
	return filepath.Join(projectDir, ".iara", "tasks", taskID)
}

// WorktreeBase returns the worktree base directory for a task.
func WorktreeBase(projectDir, taskSlug string) string {
	return filepath.Join(projectDir, ".worktrees", taskSlug)
}

// Delete removes a task's entire directory from disk.
// It does not remove worktrees — call RemoveWorktree first if needed.
func Delete(projectDir, taskID string) error {
	dir := TaskDir(projectDir, taskID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return os.RemoveAll(dir)
}

// MigrateSessionsIfNeeded moves legacy .iara/sessions/ to .iara/tasks/default/sessions/
// if the old directory exists and the new one does not. This is a one-time migration.
func MigrateSessionsIfNeeded(projectDir string) error {
	oldDir := filepath.Join(projectDir, ".iara", "sessions")
	newDir := DefaultSessionsDir(projectDir)

	// Check if migration is needed
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil // no old sessions
	}
	if _, err := os.Stat(newDir); err == nil {
		return nil // already migrated
	}

	// Create parent directory and move
	if err := os.MkdirAll(filepath.Dir(newDir), 0755); err != nil {
		return fmt.Errorf("create default task dir: %w", err)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("migrate sessions: %w", err)
	}
	return nil
}
