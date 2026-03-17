package dev

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ahtwr/iara/internal/devlog"
)

// Stop sends SIGTERM to the supervisor process and waits for it to exit.
// If the process doesn't exit within 5 seconds, it sends SIGKILL.
func Stop(taskDir string) error {
	pidData, err := os.ReadFile(PIDPath(taskDir))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dev supervisor running (pid file not found)")
		}
		return fmt.Errorf("read pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return fmt.Errorf("invalid pid file: %w", err)
	}

	// Check if process exists.
	if err := syscall.Kill(pid, 0); err != nil {
		os.Remove(PIDPath(taskDir))
		return fmt.Errorf("supervisor process %d not running", pid)
	}

	// Send SIGTERM.
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to %d: %w", pid, err)
	}

	// Wait up to 5 seconds for the process to exit.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			// Process is gone.
			os.Remove(PIDPath(taskDir))
			fmt.Println("Dev commands stopped.")
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Still alive, send SIGKILL.
	syscall.Kill(pid, syscall.SIGKILL)
	os.Remove(PIDPath(taskDir))
	fmt.Println("Dev commands stopped.")
	return nil
}

// Restart stops the supervisor, clears logs, and re-launches.
func Restart(taskDir, projectDir string) error {
	// Stop if running (ignore errors — may not be running).
	_ = Stop(taskDir)

	// Clear log files.
	logDir := devlog.Dir(taskDir)
	entries, err := os.ReadDir(logDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
				os.Remove(filepath.Join(logDir, e.Name()))
			}
		}
	}

	// Re-launch.
	return Launch(taskDir, projectDir)
}

// PrintStatus reads and displays the current dev supervisor status.
func PrintStatus(taskDir string) error {
	data, err := os.ReadFile(StatusPath(taskDir))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dev status found (is the supervisor running?)")
		}
		return fmt.Errorf("read status: %w", err)
	}

	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse status: %w", err)
	}

	fmt.Printf("Dev supervisor PID: %d (started %s)\n\n", s.PID, s.Started)
	fmt.Printf("  %-14s %-22s %-14s %-8s %s\n", "Subproject", "Command", "Type", "Port", "Status")
	fmt.Printf("  %s\n", strings.Repeat("\u2500", 70))

	logDir := devlog.Dir(taskDir)

	for _, p := range s.Processes {
		// Check if the process is actually alive.
		status := p.Status
		if p.PID > 0 && status == "running" {
			if err := syscall.Kill(p.PID, 0); err != nil {
				status = "stopped"
			}
		}

		portStr := ""
		if p.Port > 0 {
			portStr = fmt.Sprintf(":%d", p.Port)
		}

		statusIcon := "\u2713"
		if status == "failed" || status == "stopped" {
			statusIcon = "\u2717"
		}

		fmt.Printf("  %-14s %-22s %-14s %-8s %s %s\n",
			p.Subproject, truncate(p.Cmd, 22), p.Type, portStr, statusIcon, status)

		// Print last 10 lines of log for failed processes.
		if status == "failed" {
			logPath := logFilePath(logDir, p.Subproject)
			lines := tailFile(logPath, 10)
			if len(lines) > 0 {
				fmt.Println("    --- log tail ---")
				for _, line := range lines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Println("    ---")
			}
		}
	}

	fmt.Println()
	return nil
}

// PrintLogs tails log files for dev command output.
// If subproject is empty, it tails all log files.
func PrintLogs(taskDir, subproject string, lines int) error {
	if lines <= 0 {
		lines = 50
	}

	logDir := devlog.Dir(taskDir)

	if subproject != "" {
		logPath := logFilePath(logDir, subproject)
		return printLogSection(subproject, logPath, lines)
	}

	// Tail all log files.
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no log directory found")
		}
		return fmt.Errorf("read log dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".log")
		logPath := logDir + "/" + e.Name()
		if err := printLogSection(name, logPath, lines); err != nil {
			fmt.Printf("Error reading %s: %v\n", name, err)
		}
	}

	return nil
}

// logFilePath returns the full path to a subproject's log file.
func logFilePath(logDir, subproject string) string {
	return logDir + "/" + logFileName(subproject)
}

// printLogSection prints a header and the last N lines of a log file.
func printLogSection(name, path string, n int) error {
	lines := tailFile(path, n)
	if lines == nil {
		return fmt.Errorf("could not read log for %s", name)
	}

	fmt.Printf("=== %s ===\n", name)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
	return nil
}

// tailFile returns the last n lines of a file.
func tailFile(path string, n int) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := strings.TrimRight(string(data), "\n")
	if content == "" {
		return []string{}
	}
	lines := strings.Split(content, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
