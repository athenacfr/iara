package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New("test-id", "code", false, 70)

	if s.ID != "test-id" {
		t.Errorf("ID = %q, want %q", s.ID, "test-id")
	}
	if s.Mode != "code" {
		t.Errorf("Mode = %q, want %q", s.Mode, "code")
	}
	if s.SkipPermissions {
		t.Error("expected SkipPermissions = false")
	}
	if s.AutoCompactLimit != 70 {
		t.Errorf("AutoCompactLimit = %d, want 70", s.AutoCompactLimit)
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
	dir := t.TempDir()

	s := New("save-test", "research", true, 50)
	s.ClaudeSessionID = "claude-abc-123"
	s.Summary = "Test session"

	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, ".cw", "sessions", "save-test.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file not created: %v", err)
	}

	loaded, err := Load(dir, "save-test")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "save-test" {
		t.Errorf("loaded ID = %q", loaded.ID)
	}
	if loaded.ClaudeSessionID != "claude-abc-123" {
		t.Errorf("loaded ClaudeSessionID = %q", loaded.ClaudeSessionID)
	}
	if loaded.Mode != "research" {
		t.Errorf("loaded Mode = %q", loaded.Mode)
	}
	if !loaded.SkipPermissions {
		t.Error("expected loaded SkipPermissions = true")
	}
	if loaded.AutoCompactLimit != 50 {
		t.Errorf("loaded AutoCompactLimit = %d", loaded.AutoCompactLimit)
	}
	if loaded.Summary != "Test session" {
		t.Errorf("loaded Summary = %q", loaded.Summary)
	}
	if loaded.Status != "active" {
		t.Errorf("loaded Status = %q", loaded.Status)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	s1 := New("session-1", "code", false, 0)
	s1.LastActive = "2025-01-15T10:00:00Z"
	s1.Summary = "First session"
	s1.Save(dir)

	s2 := New("session-2", "research", false, 0)
	s2.LastActive = "2025-01-15T12:00:00Z"
	s2.Summary = "Second session"
	s2.Save(dir)

	sessions, err := List(dir)
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
	dir := t.TempDir()
	sessions, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sessions != nil {
		t.Errorf("expected nil for no sessions, got %d", len(sessions))
	}
}

func TestListSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, ".cw", "sessions")
	os.MkdirAll(sessDir, 0755)

	os.WriteFile(filepath.Join(sessDir, "notes.txt"), []byte("ignore"), 0644)
	os.MkdirAll(filepath.Join(sessDir, "subdir"), 0755)

	s := New("valid", "code", false, 0)
	s.Save(dir)

	sessions, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestTouch(t *testing.T) {
	dir := t.TempDir()

	s := New("touch-test", "code", false, 0)
	s.LastActive = "2020-01-01T00:00:00Z"
	s.Save(dir)

	before := s.LastActive

	time.Sleep(10 * time.Millisecond)
	if err := s.Touch(dir); err != nil {
		t.Fatal(err)
	}

	if s.LastActive == before {
		t.Error("expected LastActive to be updated")
	}

	loaded, _ := Load(dir, "touch-test")
	if loaded.LastActive == before {
		t.Error("expected persisted LastActive to be updated")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()

	s := New("delete-me", "code", false, 0)
	s.Save(dir)

	if _, err := Load(dir, "delete-me"); err != nil {
		t.Fatal(err)
	}

	if err := Delete(dir, "delete-me"); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(dir, "delete-me"); err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteMissing(t *testing.T) {
	dir := t.TempDir()
	err := Delete(dir, "nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent session")
	}
}

func TestFindClaudeSessionID(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-find-claude-id"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	claudeDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(claudeDir, 0755)
	defer os.RemoveAll(claudeDir)

	os.WriteFile(filepath.Join(claudeDir, "abc-123.jsonl"), []byte("{}"), 0644)

	got := FindClaudeSessionID(projectDir)
	if got != "abc-123" {
		t.Errorf("FindClaudeSessionID = %q, want %q", got, "abc-123")
	}
}

func TestFindClaudeSessionIDEmpty(t *testing.T) {
	got := FindClaudeSessionID("/nonexistent-project-for-test-" + itoa(int(time.Now().UnixNano())))
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestJSONRoundtrip(t *testing.T) {
	s := &Session{
		ID:               "round-trip",
		ClaudeSessionID:  "claude-xyz",
		Mode:             "debug",
		SkipPermissions:  true,
		AutoCompactLimit: 80,
		Summary:          "Test roundtrip",
		StartedAt:        "2025-01-15T10:00:00Z",
		LastActive:       "2025-01-15T11:00:00Z",
		Status:           "active",
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.ID != s.ID || loaded.ClaudeSessionID != s.ClaudeSessionID ||
		loaded.Mode != s.Mode || loaded.SkipPermissions != s.SkipPermissions ||
		loaded.AutoCompactLimit != s.AutoCompactLimit || loaded.Summary != s.Summary ||
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
