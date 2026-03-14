package devlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Dir(dir)); err != nil {
		t.Fatalf("log dir should exist: %v", err)
	}
}

func TestCleanup(t *testing.T) {
	dir := t.TempDir()
	logDir := Dir(dir)
	os.MkdirAll(logDir, 0755)

	// Create some log files
	os.WriteFile(filepath.Join(logDir, "frontend.log"), []byte("log1"), 0644)
	os.WriteFile(filepath.Join(logDir, "backend.log"), []byte("log2"), 0644)

	if err := Cleanup(dir); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(logDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 files after cleanup, got %d", len(entries))
	}
}

func TestCleanupNoDir(t *testing.T) {
	dir := t.TempDir()
	// Should not error if log dir doesn't exist
	if err := Cleanup(dir); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateOversized(t *testing.T) {
	dir := t.TempDir()
	logDir := Dir(dir)
	os.MkdirAll(logDir, 0755)

	// Create a small file — should not be truncated
	smallContent := "line1\nline2\nline3\n"
	smallPath := filepath.Join(logDir, "small.log")
	os.WriteFile(smallPath, []byte(smallContent), 0644)

	// Create a large file that exceeds maxLogSize
	var b strings.Builder
	for i := 0; i < 200000; i++ {
		b.WriteString("this is a log line that repeats many times to create a large file\n")
	}
	largePath := filepath.Join(logDir, "large.log")
	os.WriteFile(largePath, []byte(b.String()), 0644)

	if err := TruncateOversized(dir); err != nil {
		t.Fatal(err)
	}

	// Small file should be unchanged
	got, _ := os.ReadFile(smallPath)
	if string(got) != smallContent {
		t.Error("small file should not be modified")
	}

	// Large file should be truncated to tailLines lines
	got, _ = os.ReadFile(largePath)
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != tailLines {
		t.Errorf("expected %d lines after truncation, got %d", tailLines, len(lines))
	}
}

func TestTruncateOversizedNoDir(t *testing.T) {
	dir := t.TempDir()
	if err := TruncateOversized(dir); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateFileSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("a\nb\nc\n"), 0644)

	// Should not truncate if fewer lines than limit
	if err := truncateFile(path, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "a\nb\nc\n" {
		t.Error("file should not be modified")
	}
}

func TestTruncateFileKeepsLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("1\n2\n3\n4\n5\n"), 0644)

	if err := truncateFile(path, 3); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "3\n4\n5\n" {
		t.Errorf("expected last 3 lines, got %q", string(got))
	}
}
