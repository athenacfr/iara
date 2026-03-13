package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- extractText ---

func TestExtractTextString(t *testing.T) {
	got := extractText("hello world")
	if got != "hello world" {
		t.Errorf("extractText(string) = %q, want %q", got, "hello world")
	}
}

func TestExtractTextArray(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "first message"},
		map[string]any{"type": "image", "url": "http://example.com"},
	}
	got := extractText(content)
	if got != "first message" {
		t.Errorf("extractText(array) = %q, want %q", got, "first message")
	}
}

func TestExtractTextEmptyArray(t *testing.T) {
	got := extractText([]any{})
	if got != "" {
		t.Errorf("extractText(empty array) = %q, want empty", got)
	}
}

func TestExtractTextNil(t *testing.T) {
	got := extractText(nil)
	if got != "" {
		t.Errorf("extractText(nil) = %q, want empty", got)
	}
}

func TestExtractTextArrayNoText(t *testing.T) {
	content := []any{
		map[string]any{"type": "image", "url": "http://example.com"},
	}
	got := extractText(content)
	if got != "" {
		t.Errorf("extractText(array with no text) = %q, want empty", got)
	}
}

func TestExtractTextNumber(t *testing.T) {
	got := extractText(42)
	if got != "" {
		t.Errorf("extractText(int) = %q, want empty", got)
	}
}

// --- stripTags ---

func TestStripTagsSimple(t *testing.T) {
	got := stripTags("<b>hello</b>")
	if got != "hello" {
		t.Errorf("stripTags = %q, want %q", got, "hello")
	}
}

func TestStripTagsNested(t *testing.T) {
	got := stripTags("<div><span>text</span></div>")
	if got != "text" {
		t.Errorf("stripTags nested = %q, want %q", got, "text")
	}
}

func TestStripTagsNoTags(t *testing.T) {
	got := stripTags("plain text")
	if got != "plain text" {
		t.Errorf("stripTags no tags = %q, want %q", got, "plain text")
	}
}

func TestStripTagsUnclosed(t *testing.T) {
	// Unclosed angle bracket should be preserved
	got := stripTags("a < b and c > d")
	// '<' starts a tag, '>' closes it, so "a  d"
	// Actually: "a " then tag " b and c " is consumed, then " d"
	if !strings.Contains(got, "a") {
		t.Errorf("stripTags unclosed = %q, should contain 'a'", got)
	}
}

func TestStripTagsEmpty(t *testing.T) {
	got := stripTags("")
	if got != "" {
		t.Errorf("stripTags empty = %q, want empty", got)
	}
}

func TestStripTagsSelfClosing(t *testing.T) {
	got := stripTags("before<br/>after")
	if got != "beforeafter" {
		t.Errorf("stripTags self-closing = %q, want %q", got, "beforeafter")
	}
}

// --- cleanSummary ---

func TestCleanSummaryNormal(t *testing.T) {
	got := cleanSummary("Hello, can you help me with this?", 200)
	if got != "Hello, can you help me with this?" {
		t.Errorf("cleanSummary normal = %q", got)
	}
}

func TestCleanSummaryTruncation(t *testing.T) {
	got := cleanSummary("Hello world", 5)
	if got != "Hello..." {
		t.Errorf("cleanSummary truncated = %q, want %q", got, "Hello...")
	}
}

func TestCleanSummarySystemPrefix(t *testing.T) {
	tests := []string{
		"<local-command-caveat>some content",
		"<system-reminder>some content",
		"<command-message>some content",
		"<command-name>some content",
		"<command-args>some content",
		"<local-command-stdout>some content",
	}
	for _, input := range tests {
		got := cleanSummary(input, 200)
		if got != "" {
			t.Errorf("cleanSummary(%q) = %q, want empty", input, got)
		}
	}
}

func TestCleanSummaryWithTags(t *testing.T) {
	got := cleanSummary("Please <b>fix</b> the <code>bug</code>", 200)
	if got != "Please fix the bug" {
		t.Errorf("cleanSummary with tags = %q, want %q", got, "Please fix the bug")
	}
}

func TestCleanSummaryWhitespace(t *testing.T) {
	got := cleanSummary("  lots   of   spaces  ", 200)
	if got != "lots of spaces" {
		t.Errorf("cleanSummary whitespace = %q, want %q", got, "lots of spaces")
	}
}

func TestCleanSummaryEmpty(t *testing.T) {
	got := cleanSummary("", 200)
	if got != "" {
		t.Errorf("cleanSummary empty = %q, want empty", got)
	}
}

func TestCleanSummaryOnlyTags(t *testing.T) {
	got := cleanSummary("<div><span></span></div>", 200)
	if got != "" {
		t.Errorf("cleanSummary only tags = %q, want empty", got)
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

func TestRelativeTimeOneMinute(t *testing.T) {
	got := RelativeTime(time.Now().Add(-90 * time.Second))
	if got != "1m ago" {
		t.Errorf("RelativeTime(90s ago) = %q, want %q", got, "1m ago")
	}
}

func TestRelativeTimeMinutes(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{2, "2m ago"},
		{5, "5m ago"},
		{10, "10m ago"},
		{15, "15m ago"},
		{30, "30m ago"},
		{45, "45m ago"},
		{59, "59m ago"},
	}
	for _, tt := range tests {
		got := RelativeTime(time.Now().Add(-time.Duration(tt.minutes) * time.Minute))
		if got != tt.want {
			t.Errorf("RelativeTime(%dm ago) = %q, want %q", tt.minutes, got, tt.want)
		}
	}
}

func TestRelativeTimeOneHour(t *testing.T) {
	got := RelativeTime(time.Now().Add(-90 * time.Minute))
	if got != "1h ago" {
		t.Errorf("RelativeTime(90m ago) = %q, want %q", got, "1h ago")
	}
}

func TestRelativeTimeHours(t *testing.T) {
	tests := []struct {
		hours int
		want  string
	}{
		{2, "2h ago"},
		{5, "5h ago"},
		{12, "12h ago"},
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
		{2, "2d ago"},
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
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
	}
	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- parseSession ---

func TestParseSessionValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []map[string]any{
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": "Help me fix this bug"},
			"timestamp": "2025-01-15T10:30:00Z",
		},
	}

	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.ID != "test-id" {
		t.Errorf("ID = %q, want %q", s.ID, "test-id")
	}
	if s.Summary != "Help me fix this bug" {
		t.Errorf("Summary = %q, want %q", s.Summary, "Help me fix this bug")
	}
}

func TestParseSessionSkipsSystemMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []map[string]any{
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": "<system-reminder>config info</system-reminder>"},
			"timestamp": "2025-01-15T10:30:00Z",
		},
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": "Real user message here"},
			"timestamp": "2025-01-15T10:30:01Z",
		},
	}

	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.Summary != "Real user message here" {
		t.Errorf("Summary = %q, want %q", s.Summary, "Real user message here")
	}
}

func TestParseSessionEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	s, err := parseSession(path, "empty-id")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Errorf("expected nil session for empty file, got %+v", s)
	}
}

func TestParseSessionInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	os.WriteFile(path, []byte("not json\n{also bad\n"), 0644)

	s, err := parseSession(path, "bad-id")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Errorf("expected nil session for invalid JSON, got %+v", s)
	}
}

func TestParseSessionAssistantOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []map[string]any{
		{
			"type":      "assistant",
			"message":   map[string]any{"role": "assistant", "content": "I can help with that"},
			"timestamp": "2025-01-15T10:30:00Z",
		},
	}

	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Errorf("expected nil for assistant-only session, got %+v", s)
	}
}

func TestParseSessionArrayContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []map[string]any{
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "Content from array"},
				},
			},
			"timestamp": "2025-01-15T10:30:00Z",
		},
	}

	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.Summary != "Content from array" {
		t.Errorf("Summary = %q, want %q", s.Summary, "Content from array")
	}
}

func TestParseSessionMissingFile(t *testing.T) {
	_, err := parseSession("/nonexistent/path.jsonl", "missing")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- List ---

func TestListNoSessions(t *testing.T) {
	sessions, err := List("/tmp/nonexistent-project-dir-" + itoa(int(time.Now().UnixNano())))
	if err != nil {
		t.Fatal(err)
	}
	if sessions != nil {
		t.Errorf("expected nil for no sessions, got %d", len(sessions))
	}
}

// --- extractUserText ---

func TestExtractUserTextString(t *testing.T) {
	got := extractUserText("plain text")
	if got != "plain text" {
		t.Errorf("extractUserText(string) = %q, want %q", got, "plain text")
	}
}

func TestExtractUserTextArrayWithText(t *testing.T) {
	content := []any{
		map[string]any{"type": "tool_result", "content": "tool output"},
		map[string]any{"type": "text", "text": "user input"},
	}
	got := extractUserText(content)
	if got != "user input" {
		t.Errorf("extractUserText(array) = %q, want %q", got, "user input")
	}
}

func TestExtractUserTextArrayToolResultOnly(t *testing.T) {
	content := []any{
		map[string]any{"type": "tool_result", "content": "tool output"},
	}
	got := extractUserText(content)
	if got != "" {
		t.Errorf("extractUserText(tool_result only) = %q, want empty", got)
	}
}

func TestExtractUserTextNil(t *testing.T) {
	got := extractUserText(nil)
	if got != "" {
		t.Errorf("extractUserText(nil) = %q, want empty", got)
	}
}

// --- extractAssistantText ---

func TestExtractAssistantTextMultipleBlocks(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "First part."},
		map[string]any{"type": "tool_use", "name": "Read"},
		map[string]any{"type": "text", "text": "Second part."},
	}
	got := extractAssistantText(content)
	if got != "First part. Second part." {
		t.Errorf("extractAssistantText = %q, want %q", got, "First part. Second part.")
	}
}

func TestExtractAssistantTextToolUseOnly(t *testing.T) {
	content := []any{
		map[string]any{"type": "tool_use", "name": "Read"},
	}
	got := extractAssistantText(content)
	if got != "" {
		t.Errorf("extractAssistantText(tool_use only) = %q, want empty", got)
	}
}

func TestExtractAssistantTextString(t *testing.T) {
	// Assistant content is always an array, string should return ""
	got := extractAssistantText("plain string")
	if got != "" {
		t.Errorf("extractAssistantText(string) = %q, want empty", got)
	}
}

func TestExtractAssistantTextWhitespaceOnly(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "   "},
	}
	got := extractAssistantText(content)
	if got != "" {
		t.Errorf("extractAssistantText(whitespace) = %q, want empty", got)
	}
}

// --- extractMessages ---

func TestExtractMessagesBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	entries := []map[string]any{
		{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": "Hello"},
		},
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "Hi there!"},
				},
			},
		},
	}
	writeJSONL(t, path, entries)

	msgs := extractMessages(path, 5)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Text != "Hello" {
		t.Errorf("msg[0] = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Text != "Hi there!" {
		t.Errorf("msg[1] = %+v", msgs[1])
	}
}

func TestExtractMessagesLimitsToMaxPairs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	var entries []map[string]any
	for i := 0; i < 10; i++ {
		entries = append(entries, map[string]any{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": "msg " + itoa(i)},
		})
		entries = append(entries, map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"role":    "assistant",
				"content": []any{map[string]any{"type": "text", "text": "reply " + itoa(i)}},
			},
		})
	}
	writeJSONL(t, path, entries)

	msgs := extractMessages(path, 2) // max 2 pairs = 4 messages
	if len(msgs) != 4 {
		t.Errorf("expected 4 messages (2 pairs), got %d", len(msgs))
	}
}

func TestExtractMessagesMissingFile(t *testing.T) {
	msgs := extractMessages("/nonexistent/path.jsonl", 5)
	if msgs != nil {
		t.Errorf("expected nil for missing file, got %d msgs", len(msgs))
	}
}

func TestExtractMessagesSkipsSystemMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	entries := []map[string]any{
		{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": "<system-reminder>internal</system-reminder>"},
		},
		{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": "Real message"},
		},
	}
	writeJSONL(t, path, entries)

	msgs := extractMessages(path, 5)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (system skipped), got %d", len(msgs))
	}
	if msgs[0].Text != "Real message" {
		t.Errorf("msg = %q, want %q", msgs[0].Text, "Real message")
	}
}

// --- List (full coverage) ---

func TestListWithSessions(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-list-sessions"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Create two session files
	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Hello"}, "timestamp": "2025-01-15T10:00:00Z"},
	}
	writeJSONL(t, filepath.Join(sessionDir, "sess-a.jsonl"), entries)

	entries2 := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "World"}, "timestamp": "2025-01-15T11:00:00Z"},
	}
	writeJSONL(t, filepath.Join(sessionDir, "sess-b.jsonl"), entries2)

	// Create a non-jsonl file (should be ignored)
	os.WriteFile(filepath.Join(sessionDir, "notes.txt"), []byte("ignore"), 0644)
	// Create a directory (should be ignored)
	os.MkdirAll(filepath.Join(sessionDir, "subdir"), 0755)

	sessions, err := List(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Should be sorted by timestamp descending
	if sessions[0].Timestamp.Before(sessions[1].Timestamp) {
		t.Error("sessions should be sorted most recent first")
	}
}

func TestListSkipsUnparseableSessions(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-list-unparseable"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Empty session file (should be skipped - returns nil session)
	os.WriteFile(filepath.Join(sessionDir, "empty.jsonl"), []byte(""), 0644)

	// Valid session
	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Valid"}, "timestamp": "2025-01-15T10:00:00Z"},
	}
	writeJSONL(t, filepath.Join(sessionDir, "valid.jsonl"), entries)

	sessions, err := List(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session (empty skipped), got %d", len(sessions))
	}
}

// --- parseSession: timestamp with no summary ---

func TestParseSessionTimestampButNoSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// User message with only system content (cleanSummary returns "")
	// but timestamp is captured
	entries := []map[string]any{
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": "<system-reminder>stuff</system-reminder>"},
			"timestamp": "2025-01-15T10:30:00Z",
		},
	}
	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	// firstTS is set but no real summary, so should return "(no summary)"
	if s == nil {
		t.Fatal("expected non-nil session with (no summary)")
	}
	if s.Summary != "(no summary)" {
		t.Errorf("Summary = %q, want %q", s.Summary, "(no summary)")
	}
}

// --- ExtractRecentContext ---

func TestExtractRecentContextNoSessions(t *testing.T) {
	got := ExtractRecentContext("/nonexistent-path-for-test", 5)
	if got != "" {
		t.Errorf("expected empty for no sessions, got %q", got)
	}
}

func TestExtractRecentContextWithSession(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-extract-context"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Fix the bug"}},
		{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "I'll look into it"}}}},
	}
	writeJSONL(t, filepath.Join(sessionDir, "sess.jsonl"), entries)

	got := ExtractRecentContext(projectDir, 5)
	if got == "" {
		t.Fatal("expected non-empty context")
	}
	if !strings.Contains(got, "Auto-compact was triggered") {
		t.Error("expected auto-compact header")
	}
	if !strings.Contains(got, "User: Fix the bug") {
		t.Error("expected user message in context")
	}
	if !strings.Contains(got, "Assistant: I'll look into it") {
		t.Error("expected assistant message in context")
	}
}

func TestExtractRecentContextEmptyDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-extract-empty"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// No .jsonl files
	got := ExtractRecentContext(projectDir, 5)
	if got != "" {
		t.Errorf("expected empty for dir with no jsonl files, got %q", got)
	}
}

func TestExtractRecentContextEmptyMessages(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-extract-empty-msgs"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Session with only system messages (extractMessages returns empty)
	os.WriteFile(filepath.Join(sessionDir, "sess.jsonl"), []byte("{\"bad\":true}\n"), 0644)

	got := ExtractRecentContext(projectDir, 5)
	if got != "" {
		t.Errorf("expected empty for session with no user messages, got %q", got)
	}
}

// --- parseSession: user message with empty text ---

func TestParseSessionEmptyTextContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []map[string]any{
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": ""},
			"timestamp": "2025-01-15T10:30:00Z",
		},
		{
			"type":      "user",
			"message":   map[string]any{"role": "user", "content": "Real message"},
			"timestamp": "2025-01-15T10:30:01Z",
		},
	}
	writeJSONL(t, path, entries)

	s, err := parseSession(path, "test-id")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.Summary != "Real message" {
		t.Errorf("Summary = %q, want %q", s.Summary, "Real message")
	}
}

// --- List: Stat error in loop ---

func TestListStatError(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-list-stat-error"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Create a valid session
	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Hello"}, "timestamp": "2025-01-15T10:00:00Z"},
	}
	writeJSONL(t, filepath.Join(sessionDir, "good.jsonl"), entries)

	// Create a directory named like a session (IsDir check)
	os.MkdirAll(filepath.Join(sessionDir, "dir.jsonl"), 0755)

	sessions, err := List(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	// Directory should be skipped
	if len(sessions) != 1 {
		t.Errorf("expected 1 session (dir skipped), got %d", len(sessions))
	}
}

// --- extractMessages: invalid JSON lines ---

func TestExtractMessagesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := "not json\n{\"type\":\"user\",\"message\":{\"role\":\"user\",\"content\":\"Valid\"}}\n"
	os.WriteFile(path, []byte(content), 0644)

	msgs := extractMessages(path, 5)
	if len(msgs) != 1 {
		t.Errorf("expected 1 message (invalid JSON skipped), got %d", len(msgs))
	}
}

// --- ExtractRecentContext: dir with non-jsonl and directory entries ---

func TestExtractRecentContextSkipsNonJsonl(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-extract-skip-nonjsonl"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Non-jsonl file
	os.WriteFile(filepath.Join(sessionDir, "notes.txt"), []byte("ignore"), 0644)
	// Directory
	os.MkdirAll(filepath.Join(sessionDir, "subdir"), 0755)

	// Valid session
	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Hello"}},
	}
	writeJSONL(t, filepath.Join(sessionDir, "valid.jsonl"), entries)

	got := ExtractRecentContext(projectDir, 5)
	if !strings.Contains(got, "Hello") {
		t.Errorf("expected Hello in context, got %q", got)
	}
}

func TestExtractRecentContextPicksNewest(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-extract-newest"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Old session
	entries1 := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Old message"}},
	}
	writeJSONL(t, filepath.Join(sessionDir, "old.jsonl"), entries1)

	// Wait so modification time differs
	time.Sleep(50 * time.Millisecond)

	// New session
	entries2 := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "New message"}},
	}
	writeJSONL(t, filepath.Join(sessionDir, "new.jsonl"), entries2)

	got := ExtractRecentContext(projectDir, 5)
	if !strings.Contains(got, "New message") {
		t.Errorf("expected newest session content, got %q", got)
	}
}

// --- extractMessages: long assistant text truncation ---

func TestExtractMessagesTruncatesLongAssistant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	longText := strings.Repeat("a", 600)
	entries := []map[string]any{
		{
			"type": "assistant",
			"message": map[string]any{
				"role":    "assistant",
				"content": []any{map[string]any{"type": "text", "text": longText}},
			},
		},
	}
	writeJSONL(t, path, entries)

	msgs := extractMessages(path, 5)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Text) != 503 { // 500 + "..."
		t.Errorf("expected truncated to 503 chars, got %d", len(msgs[0].Text))
	}
	if !strings.HasSuffix(msgs[0].Text, "...") {
		t.Error("expected '...' suffix")
	}
}

// --- extractUserText: array with non-map items ---

func TestExtractUserTextArrayNonMap(t *testing.T) {
	content := []any{"not a map", 42}
	got := extractUserText(content)
	if got != "" {
		t.Errorf("expected empty for non-map items, got %q", got)
	}
}

// --- extractAssistantText: array with non-map items ---

func TestExtractAssistantTextArrayNonMap(t *testing.T) {
	content := []any{"not a map", 42}
	got := extractAssistantText(content)
	if got != "" {
		t.Errorf("expected empty for non-map items, got %q", got)
	}
}

// --- extractAssistantText: nil ---

func TestExtractAssistantTextNil(t *testing.T) {
	got := extractAssistantText(nil)
	if got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
}

// --- newestJSONL ---

func TestNewestJSONL(t *testing.T) {
	dir := t.TempDir()

	// Create older file
	writeJSONL(t, filepath.Join(dir, "old.jsonl"), []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "old"}},
	})
	time.Sleep(50 * time.Millisecond)

	// Create newer file
	writeJSONL(t, filepath.Join(dir, "new.jsonl"), []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "new"}},
	})

	got := newestJSONL(dir)
	if !strings.HasSuffix(got, "new.jsonl") {
		t.Errorf("newestJSONL = %q, want new.jsonl", got)
	}
}

func TestNewestJSONLEmpty(t *testing.T) {
	dir := t.TempDir()
	got := newestJSONL(dir)
	if got != "" {
		t.Errorf("newestJSONL(empty dir) = %q, want empty", got)
	}
}

func TestNewestJSONLMissing(t *testing.T) {
	got := newestJSONL("/nonexistent/path")
	if got != "" {
		t.Errorf("newestJSONL(missing) = %q, want empty", got)
	}
}

// --- extractRichMessages ---

func TestExtractRichMessagesBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	entries := []map[string]any{
		{"type": "user", "message": map[string]any{"role": "user", "content": "Fix the bug"}},
		{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "text", "text": "I'll fix it."},
		}}},
	}
	writeJSONL(t, path, entries)

	msgs := extractRichMessages(path, 20, 2000)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Text != "Fix the bug" {
		t.Errorf("msg[0] = %q", msgs[0].Text)
	}
	if msgs[1].Text != "I'll fix it." {
		t.Errorf("msg[1] = %q", msgs[1].Text)
	}
}

func TestExtractRichMessagesIncludesToolUse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	entries := []map[string]any{
		{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "text", "text": "Let me read that."},
			map[string]any{"type": "tool_use", "name": "Read", "input": map[string]any{"file_path": "/foo/bar.go"}},
			map[string]any{"type": "text", "text": "Found the issue."},
		}}},
	}
	writeJSONL(t, path, entries)

	msgs := extractRichMessages(path, 20, 2000)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Text, "[Used tool: Read /foo/bar.go]") {
		t.Errorf("expected tool summary, got %q", msgs[0].Text)
	}
	if !strings.Contains(msgs[0].Text, "Let me read that.") {
		t.Errorf("expected text content, got %q", msgs[0].Text)
	}
}

func TestExtractRichMessagesLimitsCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	var entries []map[string]any
	for i := 0; i < 30; i++ {
		entries = append(entries, map[string]any{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": "msg " + itoa(i)},
		})
	}
	writeJSONL(t, path, entries)

	msgs := extractRichMessages(path, 5, 2000)
	if len(msgs) != 5 {
		t.Errorf("expected 5 messages, got %d", len(msgs))
	}
	// Should be the last 5
	if msgs[0].Text != "msg 25" {
		t.Errorf("expected msg 25, got %q", msgs[0].Text)
	}
}

func TestExtractRichMessagesMissingFile(t *testing.T) {
	msgs := extractRichMessages("/nonexistent/path.jsonl", 20, 2000)
	if msgs != nil {
		t.Errorf("expected nil for missing file, got %d msgs", len(msgs))
	}
}

// --- extractRichAssistantText ---

func TestExtractRichAssistantTextToolUseBash(t *testing.T) {
	content := []any{
		map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]any{"command": "go test ./..."}},
	}
	got := extractRichAssistantText(content, 2000)
	if got != "[Used tool: Bash go test ./...]" {
		t.Errorf("got %q", got)
	}
}

func TestExtractRichAssistantTextNonArray(t *testing.T) {
	got := extractRichAssistantText("not an array", 2000)
	if got != "" {
		t.Errorf("expected empty for non-array, got %q", got)
	}
}

func TestExtractRichAssistantTextTruncates(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": strings.Repeat("x", 3000)},
	}
	got := extractRichAssistantText(content, 100)
	if len(got) != 103 { // 100 + "..."
		t.Errorf("expected 103 chars, got %d", len(got))
	}
}

// --- buildRawContext ---

func TestBuildRawContextCapsTotalLength(t *testing.T) {
	home, _ := os.UserHomeDir()
	projectDir := "/test-build-raw-context"
	encoded := strings.ReplaceAll(projectDir, "/", "-")
	sessionDir := filepath.Join(home, ".claude", "projects", encoded)
	os.MkdirAll(sessionDir, 0755)
	defer os.RemoveAll(sessionDir)

	// Create a session with lots of long messages
	var entries []map[string]any
	longText := strings.Repeat("a", 2000)
	for i := 0; i < 20; i++ {
		entries = append(entries, map[string]any{
			"type":    "user",
			"message": map[string]any{"role": "user", "content": longText},
		})
	}
	writeJSONL(t, filepath.Join(sessionDir, "sess.jsonl"), entries)

	got := buildRawContext(projectDir)
	if len(got) > 10000 {
		t.Errorf("expected capped at 10000 chars, got %d", len(got))
	}
}

func TestBuildRawContextEmpty(t *testing.T) {
	got := buildRawContext("/nonexistent-project-for-test")
	if got != "" {
		t.Errorf("expected empty for missing project, got %q", got)
	}
}

// --- AnalyzeContext ---

func TestAnalyzeContextNoClaude(t *testing.T) {
	// Set PATH to empty so claude won't be found
	t.Setenv("PATH", t.TempDir())
	_, err := AnalyzeContext("/some/dir")
	if err == nil {
		t.Error("expected error when claude not in PATH")
	}
	if !strings.Contains(err.Error(), "claude not in PATH") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAnalyzeContextNoSession(t *testing.T) {
	// claude is in PATH but no session data exists
	_, err := AnalyzeContext("/nonexistent-project-for-analyze-test")
	if err == nil {
		t.Error("expected error for missing session")
	}
	if !strings.Contains(err.Error(), "no session context") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- helpers ---

func writeJSONL(t *testing.T, path string, entries []map[string]any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(data)
		f.Write([]byte("\n"))
	}
}
