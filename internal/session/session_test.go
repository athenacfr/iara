package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New("test-id", "code", false)

	if s.ID != "test-id" {
		t.Errorf("ID = %q, want %q", s.ID, "test-id")
	}
	if s.Mode != "code" {
		t.Errorf("Mode = %q, want %q", s.Mode, "code")
	}
	if s.SkipPermissions {
		t.Error("expected SkipPermissions = false")
	}
	if s.Status != "active" {
		t.Errorf("Status = %q, want %q", s.Status, "active")
	}
	if s.StartedAt == "" {
		t.Error("expected StartedAt to be set")
	}
	if s.LastActive == "" {
		t.Error("expected LastActive to be set")
	}
}

func TestSaveAndLoad(t *testing.T) {
	sessDir := t.TempDir()

	s := New("save-test", "research", true)
	s.Summary = "Test session"

	if err := s.Save(sessDir); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(sessDir, "save-test.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file not created: %v", err)
	}

	loaded, err := Load(sessDir, "save-test")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "save-test" {
		t.Errorf("loaded ID = %q", loaded.ID)
	}
	if loaded.Mode != "research" {
		t.Errorf("loaded Mode = %q", loaded.Mode)
	}
	if !loaded.SkipPermissions {
		t.Error("expected loaded SkipPermissions = true")
	}
	if loaded.Summary != "Test session" {
		t.Errorf("loaded Summary = %q", loaded.Summary)
	}
	if loaded.Status != "active" {
		t.Errorf("loaded Status = %q", loaded.Status)
	}
}

func TestLoadMissing(t *testing.T) {
	sessDir := t.TempDir()
	_, err := Load(sessDir, "nonexistent")
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestList(t *testing.T) {
	sessDir := t.TempDir()

	s1 := New("session-1", "code", false)
	s1.LastActive = "2025-01-15T10:00:00Z"
	s1.Summary = "First session"
	s1.Save(sessDir)

	s2 := New("session-2", "research", false)
	s2.LastActive = "2025-01-15T12:00:00Z"
	s2.Summary = "Second session"
	s2.Save(sessDir)

	sessions, err := List(sessDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Sorted by last_active descending
	if sessions[0].ID != "session-2" {
		t.Errorf("expected session-2 first, got %q", sessions[0].ID)
	}
	if sessions[1].ID != "session-1" {
		t.Errorf("expected session-1 second, got %q", sessions[1].ID)
	}
}

func TestListEmpty(t *testing.T) {
	// Use a non-existent subdirectory so ReadDir returns IsNotExist
	sessDir := filepath.Join(t.TempDir(), "sessions")
	sessions, err := List(sessDir)
	if err != nil {
		t.Fatal(err)
	}
	if sessions != nil {
		t.Errorf("expected nil for no sessions, got %d", len(sessions))
	}
}

func TestListSkipsNonJSON(t *testing.T) {
	sessDir := t.TempDir()

	os.WriteFile(filepath.Join(sessDir, "notes.txt"), []byte("ignore"), 0644)
	os.MkdirAll(filepath.Join(sessDir, "subdir"), 0755)

	s := New("valid", "code", false)
	s.Save(sessDir)

	sessions, err := List(sessDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestTouch(t *testing.T) {
	sessDir := t.TempDir()

	s := New("touch-test", "code", false)
	s.LastActive = "2020-01-01T00:00:00Z"
	s.Save(sessDir)

	before := s.LastActive

	time.Sleep(10 * time.Millisecond)
	if err := s.Touch(sessDir); err != nil {
		t.Fatal(err)
	}

	if s.LastActive == before {
		t.Error("expected LastActive to be updated")
	}

	loaded, _ := Load(sessDir, "touch-test")
	if loaded.LastActive == before {
		t.Error("expected persisted LastActive to be updated")
	}
}

func TestDelete(t *testing.T) {
	sessDir := t.TempDir()

	s := New("delete-me", "code", false)
	s.Save(sessDir)

	if _, err := Load(sessDir, "delete-me"); err != nil {
		t.Fatal(err)
	}

	if err := Delete(sessDir, "delete-me"); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(sessDir, "delete-me"); err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteMissing(t *testing.T) {
	sessDir := t.TempDir()
	err := Delete(sessDir, "nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent session")
	}
}

func TestJSONRoundtrip(t *testing.T) {
	s := &Session{
		ID:              "round-trip",
		Mode:            "debug",
		SkipPermissions: true,
		Summary:         "Test roundtrip",
		StartedAt:       "2025-01-15T10:00:00Z",
		LastActive:      "2025-01-15T11:00:00Z",
		Status:          "active",
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.ID != s.ID ||
		loaded.Mode != s.Mode || loaded.SkipPermissions != s.SkipPermissions ||
		loaded.Summary != s.Summary ||
		loaded.StartedAt != s.StartedAt || loaded.LastActive != s.LastActive ||
		loaded.Status != s.Status {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", loaded, *s)
	}
}

// --- RelativeTime ---

func TestRelativeTimeZero(t *testing.T) {
	got := RelativeTime(time.Time{})
	if got != "unknown" {
		t.Errorf("RelativeTime(zero) = %q, want %q", got, "unknown")
	}
}

func TestRelativeTimeJustNow(t *testing.T) {
	got := RelativeTime(time.Now().Add(-30 * time.Second))
	if got != "just now" {
		t.Errorf("RelativeTime(30s ago) = %q, want %q", got, "just now")
	}
}

func TestRelativeTimeMinutes(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{1, "1m ago"},
		{5, "5m ago"},
		{30, "30m ago"},
		{59, "59m ago"},
	}
	for _, tt := range tests {
		got := RelativeTime(time.Now().Add(-time.Duration(tt.minutes)*time.Minute - 30*time.Second))
		if got != tt.want {
			t.Errorf("RelativeTime(%dm ago) = %q, want %q", tt.minutes, got, tt.want)
		}
	}
}

func TestRelativeTimeHours(t *testing.T) {
	tests := []struct {
		hours int
		want  string
	}{
		{1, "1h ago"},
		{5, "5h ago"},
		{23, "23h ago"},
	}
	for _, tt := range tests {
		got := RelativeTime(time.Now().Add(-time.Duration(tt.hours) * time.Hour))
		if got != tt.want {
			t.Errorf("RelativeTime(%dh ago) = %q, want %q", tt.hours, got, tt.want)
		}
	}
}

func TestRelativeTimeDays(t *testing.T) {
	tests := []struct {
		days int
		want string
	}{
		{1, "1d ago"},
		{7, "7d ago"},
		{30, "30d ago"},
	}
	for _, tt := range tests {
		got := RelativeTime(time.Now().Add(-time.Duration(tt.days) * 24 * time.Hour))
		if got != tt.want {
			t.Errorf("RelativeTime(%dd ago) = %q, want %q", tt.days, got, tt.want)
		}
	}
}

// --- itoa ---

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
	}
	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- ParseTime ---

func TestParseTime(t *testing.T) {
	got := ParseTime("2025-01-15T10:30:00Z")
	if got.IsZero() {
		t.Error("expected non-zero time")
	}
	if got.Hour() != 10 || got.Minute() != 30 {
		t.Errorf("got %v", got)
	}
}

func TestParseTimeInvalid(t *testing.T) {
	got := ParseTime("not-a-time")
	if !got.IsZero() {
		t.Errorf("expected zero time for invalid input, got %v", got)
	}
}

// --- extractConversationExcerpt ---

func TestExcerptFromJSONL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/home/testuser/projects/myproject"
	sessionID := "test-session-123"

	encoded := "-home-testuser-projects-myproject"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	lines := []string{
		`{"type":"progress","slug":"calm-exploring-turtle"}`,
		`{"type":"user","message":{"role":"user","content":"<command-message>cw:yolo</command-message>"}}`,
		`{"type":"user","message":{"role":"user","content":"fix the login page"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"I'll fix the login page now."}}`,
		`{"type":"user","message":{"role":"user","content":"also update the tests"}}`,
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	if !strings.Contains(got, "User: fix the login page") {
		t.Errorf("expected user message in excerpt, got %q", got)
	}
	if !strings.Contains(got, "Assistant: I'll fix the login page now.") {
		t.Errorf("expected assistant message in excerpt, got %q", got)
	}
	if !strings.Contains(got, "User: also update the tests") {
		t.Errorf("expected second user message in excerpt, got %q", got)
	}
	// Command messages should be skipped
	if strings.Contains(got, "cw:yolo") {
		t.Errorf("excerpt should not contain command messages, got %q", got)
	}
}

func TestExcerptSkipsCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/test/dir"
	sessionID := "cmd-session"

	encoded := "-test-dir"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":"<command-message>foo</command-message>"}}`,
		`{"type":"user","message":{"role":"user","content":"<local-command>bar</local-command>"}}`,
		`{"type":"user","message":{"role":"user","content":"hello world"}}`,
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	if got != "User: hello world" {
		t.Errorf("excerpt = %q, want %q", got, "User: hello world")
	}
}

func TestExcerptContentArray(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/test/arr"
	sessionID := "arr-session"

	encoded := "-test-arr"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	lines := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"array content here"}]}}`,
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	if got != "User: array content here" {
		t.Errorf("excerpt = %q, want %q", got, "User: array content here")
	}
}

func TestExcerptTruncatesLongMessages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/test/long"
	sessionID := "long-session"

	encoded := "-test-long"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	longMsg := strings.Repeat("x", 300)
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"` + longMsg + `"}}`,
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	// "User: " (6) + 200 runes + "..." (3) = 209
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncated message to end with '...', got %q", got)
	}
	// Should have exactly 200 x's
	content := strings.TrimPrefix(got, "User: ")
	content = strings.TrimSuffix(content, "...")
	if len(content) != 200 {
		t.Errorf("expected 200 rune content, got %d", len(content))
	}
}

func TestExcerptLimitsMessagePairs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/test/limit"
	sessionID := "limit-session"

	encoded := "-test-limit"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	// Create 10 user messages — should only include first 5
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"message %d"}}`, i))
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	count := strings.Count(got, "User: message")
	if count != 5 {
		t.Errorf("expected 5 user messages, got %d in:\n%s", count, got)
	}
	if strings.Contains(got, "message 5") {
		t.Errorf("should not contain message 5+, got:\n%s", got)
	}
}

func TestExcerptMissingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := extractConversationExcerpt("nonexistent", "/no/such/dir")
	if got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

func TestExcerptNoUserMessages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workDir := "/test/nousers"
	sessionID := "no-users"

	encoded := "-test-nousers"
	jsonlDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(jsonlDir, 0755)

	lines := []string{
		`{"type":"progress","slug":"test-slug"}`,
		`{"type":"system","message":"system init"}`,
	}
	jsonlPath := filepath.Join(jsonlDir, sessionID+".jsonl")
	os.WriteFile(jsonlPath, []byte(joinTestLines(lines)), 0644)

	got := extractConversationExcerpt(sessionID, workDir)
	if got != "" {
		t.Errorf("expected empty for no user messages, got %q", got)
	}
}

func joinTestLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}
