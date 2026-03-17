package dev

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ahtwr/iara/internal/devlog"
)

// trackedProcess holds a running long-running process and its metadata.
type trackedProcess struct {
	subproject string
	cmd        string
	port       int
	process    *os.Process
}

// Launch starts the dev supervisor. It runs one-shot commands sequentially,
// then launches long-running commands in parallel, and waits for them or a signal.
func Launch(taskDir, projectDir string) error {
	cfg, err := LoadConfig(taskDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Version != ConfigVersion {
		return fmt.Errorf("outdated config (version %d, current %d) — delete and re-discover", cfg.Version, ConfigVersion)
	}

	if err := devlog.EnsureDir(taskDir); err != nil {
		return fmt.Errorf("ensure log dir: %w", err)
	}

	// Check and resolve port conflicts.
	if err := resolvePorts(cfg, taskDir); err != nil {
		return fmt.Errorf("resolve ports: %w", err)
	}

	logDir := devlog.Dir(taskDir)

	// Collect all commands with their subproject reference.
	type cmdEntry struct {
		sp  *Subproject
		cmd Command
	}
	var entries []cmdEntry
	for i := range cfg.Subprojects {
		sp := &cfg.Subprojects[i]
		for _, cmd := range sp.Commands {
			entries = append(entries, cmdEntry{sp: sp, cmd: cmd})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].cmd.Priority < entries[j].cmd.Priority
	})

	// Group by priority and execute each group.
	// Commands with the same priority run in parallel; groups run sequentially.
	var tracked []trackedProcess
	for len(entries) > 0 {
		prio := entries[0].cmd.Priority
		// Collect all entries at this priority level.
		var group []cmdEntry
		for len(entries) > 0 && entries[0].cmd.Priority == prio {
			group = append(group, entries[0])
			entries = entries[1:]
		}

		// Execute the group: one-shots in parallel (wait for all), long-running started in parallel (don't wait).
		var oneShots []cmdEntry
		var longRunning []cmdEntry
		for _, e := range group {
			if e.cmd.Type == "one-shot" {
				oneShots = append(oneShots, e)
			} else {
				longRunning = append(longRunning, e)
			}
		}

		// Run one-shots in parallel and wait for all to complete.
		if len(oneShots) > 0 {
			errs := make([]error, len(oneShots))
			var wg sync.WaitGroup
			for i, e := range oneShots {
				wg.Add(1)
				go func(idx int, entry cmdEntry) {
					defer wg.Done()
					errs[idx] = runOneShot(projectDir, logDir, entry.sp, entry.cmd)
				}(i, e)
			}
			wg.Wait()
			for _, err := range errs {
				if err != nil {
					killAll(tracked)
					return err
				}
			}
		}

		// Start long-running commands in parallel.
		for _, e := range longRunning {
			tp, err := startLongRunning(projectDir, logDir, e.sp, e.cmd)
			if err != nil {
				killAll(tracked)
				return fmt.Errorf("start %s/%s: %w", e.sp.Path, e.cmd.Cmd, err)
			}
			tracked = append(tracked, tp)
		}
	}

	// Write PID file.
	myPID := os.Getpid()
	if err := os.WriteFile(PIDPath(taskDir), []byte(strconv.Itoa(myPID)+"\n"), 0644); err != nil {
		killAll(tracked)
		return fmt.Errorf("write pid file: %w", err)
	}

	// Build and write initial status.
	status := buildStatus(myPID, cfg, tracked)
	if err := writeStatus(taskDir, status); err != nil {
		killAll(tracked)
		return fmt.Errorf("write status: %w", err)
	}

	printSummary(cfg, tracked, taskDir)

	// Wait for children or signal.
	return supervise(taskDir, tracked, cfg)
}

// resolvePorts checks each subproject port for conflicts and finds alternatives.
func resolvePorts(cfg *Config, taskDir string) error {
	usedPorts := make(map[int]bool)
	changed := false

	for i := range cfg.Subprojects {
		sp := &cfg.Subprojects[i]
		if sp.Port == 0 {
			continue
		}
		origPort := sp.Port
		for isPortInUse(sp.Port) || usedPorts[sp.Port] {
			fmt.Printf("Warning: port %d (%s) is in use, trying %d\n", sp.Port, sp.Path, sp.Port+1)
			sp.Port++
		}
		usedPorts[sp.Port] = true
		if sp.Port != origPort {
			changed = true
		}
	}

	if changed {
		if err := SaveConfig(taskDir, cfg); err != nil {
			return fmt.Errorf("save updated config: %w", err)
		}
	}
	return nil
}

// isPortInUse checks if a TCP port has a listener using lsof.
func isPortInUse(port int) bool {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-t")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// logFileName returns the log file name for a subproject, replacing / with -.
func logFileName(subprojectPath string) string {
	name := strings.ReplaceAll(subprojectPath, "/", "-")
	return name + ".log"
}

// buildShellCmd constructs the shell command string, adding venv activation if needed.
func buildShellCmd(sp *Subproject, cmd Command) string {
	if sp.Venv != "" {
		return fmt.Sprintf("source %s/bin/activate && %s", sp.Venv, cmd.Cmd)
	}
	return cmd.Cmd
}

// runOneShot runs a one-shot command and waits for it to complete.
func runOneShot(projectDir, logDir string, sp *Subproject, cmd Command) error {
	fullCmd := buildShellCmd(sp, cmd)
	workDir := filepath.Join(projectDir, sp.Path)

	logFile, err := os.OpenFile(
		filepath.Join(logDir, logFileName(sp.Path)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open log for %s: %w", sp.Path, err)
	}
	defer logFile.Close()

	c := exec.Command("bash", "-c", fullCmd)
	c.Dir = workDir
	c.Stdout = logFile
	c.Stderr = logFile
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	fmt.Printf("Running [%s] %s ... ", sp.Path, cmd.Cmd)
	if err := c.Run(); err != nil {
		fmt.Println("FAILED")
		// Print last 20 lines of the log.
		printLogTail(filepath.Join(logDir, logFileName(sp.Path)), 20)
		return fmt.Errorf("one-shot command failed [%s] %s: %w", sp.Path, cmd.Cmd, err)
	}
	fmt.Println("OK")
	return nil
}

// startLongRunning starts a long-running command and returns a tracked process.
func startLongRunning(projectDir, logDir string, sp *Subproject, cmd Command) (trackedProcess, error) {
	fullCmd := buildShellCmd(sp, cmd)
	workDir := filepath.Join(projectDir, sp.Path)

	logFile, err := os.OpenFile(
		filepath.Join(logDir, logFileName(sp.Path)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return trackedProcess{}, fmt.Errorf("open log for %s: %w", sp.Path, err)
	}

	c := exec.Command("bash", "-c", fullCmd)
	c.Dir = workDir
	c.Stdout = logFile
	c.Stderr = logFile
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := c.Start(); err != nil {
		logFile.Close()
		return trackedProcess{}, err
	}

	// Close the log file handle in the parent; the child has its own fd.
	logFile.Close()

	return trackedProcess{
		subproject: sp.Path,
		cmd:        cmd.Cmd,
		port:       sp.Port,
		process:    c.Process,
	}, nil
}

// buildStatus constructs the Status struct from the current state.
func buildStatus(supervisorPID int, cfg *Config, tracked []trackedProcess) *Status {
	s := &Status{
		PID:     supervisorPID,
		Started: time.Now().Format(time.RFC3339),
	}

	// Add one-shot commands as completed.
	for _, sp := range cfg.Subprojects {
		for _, cmd := range sp.Commands {
			if cmd.Type != "one-shot" {
				continue
			}
			s.Processes = append(s.Processes, ProcessStatus{
				Subproject: sp.Path,
				Cmd:        cmd.Cmd,
				Type:       cmd.Type,
				Port:       sp.Port,
				Status:     "completed",
			})
		}
	}

	// Add long-running commands.
	for _, tp := range tracked {
		s.Processes = append(s.Processes, ProcessStatus{
			Subproject: tp.subproject,
			Cmd:        tp.cmd,
			Type:       "long-running",
			PID:        tp.process.Pid,
			Port:       tp.port,
			Status:     "running",
		})
	}

	return s
}

// writeStatus writes the status to dev-status.json.
func writeStatus(taskDir string, s *Status) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(StatusPath(taskDir), data, 0644)
}

// printSummary prints the summary table of running dev commands.
func printSummary(cfg *Config, tracked []trackedProcess, taskDir string) {
	fmt.Println()
	fmt.Println("Dev commands running:")
	fmt.Println()
	fmt.Printf("  %-14s %-22s %-14s %-8s %s\n", "Subproject", "Command", "Type", "Port", "Status")
	fmt.Printf("  %s\n", strings.Repeat("\u2500", 70))

	for _, tp := range tracked {
		portStr := ""
		if tp.port > 0 {
			portStr = fmt.Sprintf(":%d", tp.port)
		}
		fmt.Printf("  %-14s %-22s %-14s %-8s %s\n",
			tp.subproject, truncate(tp.cmd, 22), "long-running", portStr, "\u2713 running")
	}

	// Print URLs for processes with ports.
	var urls []string
	for _, tp := range tracked {
		if tp.port > 0 {
			urls = append(urls, fmt.Sprintf("  %s \u2192 http://localhost:%d", tp.subproject, tp.port))
		}
	}
	if len(urls) > 0 {
		fmt.Println()
		fmt.Println("  URLs:")
		for _, u := range urls {
			fmt.Println("  ", u)
		}
	}

	fmt.Println()
	fmt.Printf("  Logs: %s\n", devlog.Dir(taskDir))
	fmt.Println()
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// supervise waits for all tracked processes or a termination signal.
func supervise(taskDir string, tracked []trackedProcess, cfg *Config) error {
	if len(tracked) == 0 {
		os.Remove(PIDPath(taskDir))
		return nil
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	type exitResult struct {
		index int
		err   error
	}
	exitCh := make(chan exitResult, len(tracked))

	var wg sync.WaitGroup
	for i, tp := range tracked {
		wg.Add(1)
		go func(idx int, proc *os.Process) {
			defer wg.Done()
			state, err := proc.Wait()
			if err != nil {
				exitCh <- exitResult{index: idx, err: err}
				return
			}
			if !state.Success() {
				exitCh <- exitResult{index: idx, err: fmt.Errorf("exited with %s", state.String())}
				return
			}
			exitCh <- exitResult{index: idx}
		}(i, tp.process)
	}

	// Channel that closes when all children are done.
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	// Update status as children exit.
	go func() {
		for res := range exitCh {
			updateProcessStatus(taskDir, tracked[res.index], res.err)
		}
	}()

	select {
	case <-doneCh:
		// All children exited on their own.
	case <-sigCh:
		fmt.Println("\nReceived signal, stopping dev commands...")
		terminateAll(tracked)
		<-doneCh // wait for all wait goroutines to finish
	}

	close(exitCh)
	// Give consumer goroutine time to drain remaining updates.
	time.Sleep(100 * time.Millisecond)
	os.Remove(PIDPath(taskDir))
	return nil
}

// updateProcessStatus updates a single process in the status file.
func updateProcessStatus(taskDir string, tp trackedProcess, exitErr error) {
	data, err := os.ReadFile(StatusPath(taskDir))
	if err != nil {
		return
	}
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return
	}

	for i := range s.Processes {
		p := &s.Processes[i]
		if p.PID == tp.process.Pid {
			if exitErr != nil {
				p.Status = "failed"
				p.Error = exitErr.Error()
			} else {
				p.Status = "completed"
			}
			break
		}
	}

	writeStatus(taskDir, &s)
}

// killAll sends SIGTERM to all tracked process groups.
func killAll(tracked []trackedProcess) {
	for _, tp := range tracked {
		syscall.Kill(-tp.process.Pid, syscall.SIGTERM)
	}
}

// terminateAll sends SIGTERM to all process groups, waits briefly, then SIGKILLs survivors.
func terminateAll(tracked []trackedProcess) {
	for _, tp := range tracked {
		syscall.Kill(-tp.process.Pid, syscall.SIGTERM)
	}

	// Give processes up to 3 seconds to exit gracefully.
	time.Sleep(3 * time.Second)

	for _, tp := range tracked {
		// Check if still alive.
		if err := syscall.Kill(tp.process.Pid, 0); err == nil {
			syscall.Kill(-tp.process.Pid, syscall.SIGKILL)
		}
	}
}

// printLogTail prints the last n lines of a log file.
func printLogTail(path string, n int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	fmt.Println("--- last lines of log ---")
	for _, line := range lines[start:] {
		fmt.Println(line)
	}
	fmt.Println("---")
}
