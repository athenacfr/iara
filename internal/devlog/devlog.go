package devlog

import (
	"os"
	"path/filepath"
)

const (
	logDir     = ".cw/logs"
	maxLogSize = 10 * 1024 * 1024 // 10MB
	tailLines  = 5000
)

// Dir returns the log directory for a project.
func Dir(projectDir string) string {
	return filepath.Join(projectDir, logDir)
}

// EnsureDir creates the log directory if it doesn't exist.
func EnsureDir(projectDir string) error {
	return os.MkdirAll(Dir(projectDir), 0755)
}

// Cleanup removes all log files in the project's log directory.
func Cleanup(projectDir string) error {
	dir := Dir(projectDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		os.Remove(filepath.Join(dir, e.Name()))
	}
	return nil
}

// TruncateOversized checks each log file and truncates any that exceed maxLogSize
// by keeping only the last tailLines lines.
func TruncateOversized(projectDir string) error {
	dir := Dir(projectDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() <= maxLogSize {
			continue
		}
		if err := truncateFile(path, tailLines); err != nil {
			continue
		}
	}
	return nil
}

// truncateFile keeps only the last n lines of a file.
func truncateFile(path string, n int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := splitLines(data)
	if len(lines) <= n {
		return nil
	}

	// Keep last n lines
	kept := lines[len(lines)-n:]
	out := joinLines(kept)
	return os.WriteFile(path, out, 0644)
}

// splitLines splits data into lines, preserving line endings.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i+1])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// joinLines joins lines back into a single byte slice.
func joinLines(lines [][]byte) []byte {
	size := 0
	for _, l := range lines {
		size += len(l)
	}
	out := make([]byte, 0, size)
	for _, l := range lines {
		out = append(out, l...)
	}
	return out
}
