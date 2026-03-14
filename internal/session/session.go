package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ahtwr/cw/internal/paths"
)

// Session represents a cw-managed session stored in <project>/.cw/sessions/.
type Session struct {
	ID               string `json:"id"`
	ClaudeSessionID  string `json:"claude_session_id"`
	Mode             string `json:"mode"`
	SkipPermissions  bool   `json:"skip_permissions"`
	AutoCompactLimit int    `json:"auto_compact_limit"`
	Summary          string `json:"summary"`
	StartedAt        string `json:"started_at"`
	LastActive       string `json:"last_active"`
	Status           string `json:"status"` // "active" or "completed"
}

// sessionsDir returns the .cw/sessions/ directory for a project.
func sessionsDir(projectDir string) string {
	return filepath.Join(projectDir, ".cw", "sessions")
}

// sessionPath returns the file path for a session.
func sessionPath(projectDir, id string) string {
	return filepath.Join(sessionsDir(projectDir), id+".json")
}

// New creates a new session with the given parameters.
func New(id, mode string, skipPerms bool, autoCompactLimit int) *Session {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Session{
		ID:               id,
		Mode:             mode,
		SkipPermissions:  skipPerms,
		AutoCompactLimit: autoCompactLimit,
		StartedAt:        now,
		LastActive:       now,
		Status:           "active",
	}
}

// Save writes the session to <projectDir>/.cw/sessions/<id>.json.
func (s *Session) Save(projectDir string) error {
	dir := sessionsDir(projectDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return os.WriteFile(sessionPath(projectDir, s.ID), data, 0644)
}

// Load reads a session from disk.
func Load(projectDir, id string) (*Session, error) {
	data, err := os.ReadFile(sessionPath(projectDir, id))
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &s, nil
}

// List returns all sessions for a project, sorted by last_active descending.
func List(projectDir string) ([]Session, error) {
	dir := sessionsDir(projectDir)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(f.Name(), ".json")
		s, err := Load(projectDir, id)
		if err != nil {
			continue
		}
		sessions = append(sessions, *s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActive > sessions[j].LastActive
	})

	return sessions, nil
}

// Touch updates the session's LastActive timestamp and saves.
func (s *Session) Touch(projectDir string) error {
	s.LastActive = time.Now().UTC().Format(time.RFC3339)
	return s.Save(projectDir)
}

// FindClaudeSessionID discovers the Claude session ID by finding the most
// recently modified JSONL file in Claude's session storage for the given
// project directory. This reads directory listings only, not file contents.
func FindClaudeSessionID(projectDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	encoded := strings.ReplaceAll(projectDir, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", encoded)

	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var newestID string
	var newestTime time.Time
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestID = strings.TrimSuffix(f.Name(), ".jsonl")
		}
	}
	return newestID
}

// Delete removes a session file.
func Delete(projectDir, id string) error {
	return os.Remove(sessionPath(projectDir, id))
}

// ProjectSessionsDir returns the .cw/sessions/ path for a project name.
func ProjectSessionsDir(name string) string {
	return sessionsDir(filepath.Join(paths.ProjectsDir(), name))
}

// RelativeTime returns a human-readable relative time string.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return itoa(m) + "m ago"
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return itoa(h) + "h ago"
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return itoa(days) + "d ago"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ParseTime parses an RFC3339 timestamp string.
func ParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
