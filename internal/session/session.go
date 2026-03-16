package session

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/ahtwr/cw/internal/paths"
)

// Session represents a cw-managed session stored in <project>/.cw/sessions/.
// The ID is a UUID that is shared with Claude via --session-id on first launch,
// so both cw and Claude use the same identifier for the session.
type Session struct {
	ID              string `json:"id"`
	Mode            string `json:"mode"`
	SkipPermissions bool   `json:"skip_permissions"`
	Summary         string `json:"summary"`
	StartedAt       string `json:"started_at"`
	LastActive      string `json:"last_active"`
	Status          string `json:"status"` // "active" or "completed"
}

// sessionPath returns the file path for a session within the given sessions directory.
func sessionPath(sessionsDir, id string) string {
	return filepath.Join(sessionsDir, id+".json")
}

// New creates a new session with the given parameters.
func New(id, mode string, skipPerms bool) *Session {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Session{
		ID:              id,
		Mode:            mode,
		SkipPermissions: skipPerms,
		StartedAt:       now,
		LastActive:      now,
		Status:          "active",
	}
}

// Save writes the session to <sessionsDir>/<id>.json.
func (s *Session) Save(sessionsDir string) error {
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return os.WriteFile(sessionPath(sessionsDir, s.ID), data, 0644)
}

// Load reads a session from disk.
func Load(sessionsDir, id string) (*Session, error) {
	data, err := os.ReadFile(sessionPath(sessionsDir, id))
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &s, nil
}

// List returns all sessions in the given sessions directory, sorted by last_active descending.
func List(sessionsDir string) ([]Session, error) {
	files, err := os.ReadDir(sessionsDir)
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
		s, err := Load(sessionsDir, id)
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
func (s *Session) Touch(sessionsDir string) error {
	s.LastActive = time.Now().UTC().Format(time.RFC3339)
	return s.Save(sessionsDir)
}

// Delete removes a session file.
func Delete(sessionsDir, id string) error {
	return os.Remove(sessionPath(sessionsDir, id))
}

// ProjectSessionsDir returns the .cw/sessions/ path for a project name.
func ProjectSessionsDir(name string) string {
	return filepath.Join(paths.ProjectsDir(), name, ".cw", "sessions")
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

// GenerateSummary uses Claude to generate a short title for a session
// based on the conversation content. Returns "" if generation fails
// (the TUI falls back to "Session #N").
func GenerateSummary(sessionID, workDir string) string {
	excerpt := extractConversationExcerpt(sessionID, workDir)
	if excerpt == "" {
		return ""
	}

	prompt := "Summarize this coding session in under 10 words. Output ONLY the title, nothing else. No quotes, no punctuation at the end.\n\n<conversation>\n" + excerpt + "</conversation>"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", "--", prompt)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return ""
	}

	title := strings.TrimSpace(stdout.String())

	// Sanity check: reject empty or absurdly long output
	if title == "" || utf8.RuneCountInString(title) > 120 {
		return ""
	}

	return title
}

// GenerateSummaryAsync spawns a detached background process to generate the
// session summary. The process survives terminal close and CLI interruption
// by using its own process group and redirecting stdio to /dev/null.
// Use this when no spinner or in-process coordination is needed.
func GenerateSummaryAsync(sessionID, workDir, sessionsDir string) {
	cwBin, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(cwBin, "internal", "summarize", sessionID, workDir, sessionsDir)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // new session — survives terminal close
	}
	cmd.Start() // fire and forget
}

// GenerateSummaryBackground runs summary generation in-process on a goroutine
// and saves the result. Returns a channel that is closed when done, so callers
// can coordinate (e.g. keep a spinner alive until both compact and summary finish).
func GenerateSummaryBackground(sessionID, workDir, sessionsDir string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runSummarize(sessionID, workDir, sessionsDir)
	}()
	return done
}

// RunSummarize is the handler for "cw internal summarize". It generates
// a summary for the given session and saves it to disk.
func RunSummarize(sessionID, workDir, sessionsDir string) {
	runSummarize(sessionID, workDir, sessionsDir)
}

func runSummarize(sessionID, workDir, sessionsDir string) {
	s, err := Load(sessionsDir, sessionID)
	if err != nil || s.Summary != "" {
		return
	}

	summary := GenerateSummary(sessionID, workDir)
	if summary == "" {
		return
	}

	// Re-load to avoid clobbering concurrent writes
	s, err = Load(sessionsDir, sessionID)
	if err != nil {
		return
	}
	if s.Summary == "" {
		s.Summary = summary
		s.Save(sessionsDir)
	}
}

// extractConversationExcerpt reads the Claude JSONL file and builds a
// compact transcript of the first few user+assistant exchanges.
func extractConversationExcerpt(sessionID, workDir string) string {
	jsonlPath := claudeJSONLPath(sessionID, workDir)
	if jsonlPath == "" {
		return ""
	}

	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	const maxPairs = 5
	const maxMsgRunes = 200

	var parts []string
	userCount := 0

	for scanner.Scan() {
		if userCount >= maxPairs {
			break
		}

		line := scanner.Bytes()

		var rec struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil {
			continue
		}

		role := rec.Message.Role
		if role != "user" && role != "assistant" {
			continue
		}
		// Only count records with matching type field
		if rec.Type != role {
			continue
		}

		text := extractMessageText(rec.Message.Content)
		text = strings.TrimSpace(text)

		// Skip command/system messages
		if text == "" || strings.HasPrefix(text, "<") {
			continue
		}

		// Truncate long messages
		if utf8.RuneCountInString(text) > maxMsgRunes {
			runes := []rune(text)
			text = string(runes[:maxMsgRunes]) + "..."
		}

		if role == "user" {
			parts = append(parts, "User: "+text)
			userCount++
		} else {
			parts = append(parts, "Assistant: "+text)
		}
	}

	return strings.Join(parts, "\n")
}

// claudeJSONLPath returns the path to Claude's JSONL file for a session.
// Claude stores sessions at ~/.claude/projects/<encoded-workdir>/<session-id>.jsonl
// where encoded-workdir has "/" replaced by "-".
func claudeJSONLPath(sessionID, workDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	encoded := strings.ReplaceAll(workDir, string(filepath.Separator), "-")
	path := filepath.Join(home, ".claude", "projects", encoded, sessionID+".jsonl")

	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// extractMessageText gets the text content from a user message.
// Content can be a string or an array of content blocks.
func extractMessageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		for _, block := range v {
			if m, ok := block.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					return t
				}
			}
		}
	}
	return ""
}
