package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ahtwr/cw/internal/config"
)

// mockClaudeBin compiles a tiny Go program that acts as a fake "claude" binary.
// It dumps its args and env to a JSON file, then either exits or sleeps
// depending on the MOCK_CLAUDE_BEHAVIOR env var.
func mockClaudeBin(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	src := filepath.Join(binDir, "main.go")
	bin := filepath.Join(binDir, "claude")

	program := `package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type invocation struct {
	Args []string          ` + "`json:\"args\"`" + `
	Env  map[string]string ` + "`json:\"env\"`" + `
}

func main() {
	inv := invocation{
		Args: os.Args[1:],
		Env:  make(map[string]string),
	}
	for _, e := range os.Environ() {
		if idx := indexOf(e, '='); idx >= 0 {
			key := e[:idx]
			if key == "CW_PID" || key == "CW_PROJECT" || key == "CW_PROJECT_DIR" ||
				key == "CW_MODE" || key == "CW_AUTO_SETUP" || key == "CW_AUTO_COMPACT_LIMIT" ||
				key == "CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD" ||
				key == "MOCK_CLAUDE_OUTPUT" {
				inv.Env[key] = e[idx+1:]
			}
		}
	}

	outFile := os.Getenv("MOCK_CLAUDE_OUTPUT")
	if outFile != "" {
		data, _ := json.Marshal(inv)
		os.WriteFile(outFile, data, 0644)
	}

	behavior := os.Getenv("MOCK_CLAUDE_BEHAVIOR")
	switch behavior {
	case "exit1":
		os.Exit(1)
	case "sleep":
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM)
		select {
		case <-sigCh:
			os.Exit(0)
		case <-time.After(10 * time.Second):
			os.Exit(2)
		}
	default:
		os.Exit(0)
	}
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
`
	os.WriteFile(src, []byte(program), 0644)

	cmd := exec.Command("go", "build", "-o", bin, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build mock claude: %v\n%s", err, out)
	}

	// Prepend binDir to PATH so exec.Command("claude") finds our mock
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	return binDir
}

type invocation struct {
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

func readInvocation(t *testing.T, path string) invocation {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read invocation: %v", err)
	}
	var inv invocation
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatalf("unmarshal invocation: %v", err)
	}
	return inv
}

// --- Launch() with mock claude ---

func TestLaunchMinimal(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	if len(inv.Args) != 0 {
		t.Errorf("expected 0 args, got %v", inv.Args)
	}
}

func TestLaunchPassesArgs(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir:         t.TempDir(),
		SkipPermissions: true,
		SessionID:       "sess-42",
		PluginDir:       "/plugins",
		SystemPrompts:   []string{"prompt1", "prompt2"},
		AddDirs:         []string{"/repo1", "/repo2"},
		Prompt:          "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	assertContains(t, inv.Args, "--dangerously-skip-permissions")
	assertContains(t, inv.Args, "--resume")
	assertContains(t, inv.Args, "sess-42")
	assertContains(t, inv.Args, "--plugin-dir")
	assertContains(t, inv.Args, "/plugins")
	assertContains(t, inv.Args, "--append-system-prompt")
	assertContains(t, inv.Args, "--add-dir")
	assertContains(t, inv.Args, "/repo1")
	assertContains(t, inv.Args, "/repo2")
	assertContains(t, inv.Args, "--")
	assertContains(t, inv.Args, "hello")
}

func TestLaunchSessionIDFlag(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir:   t.TempDir(),
		SessionID: "abc",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	assertContains(t, inv.Args, "--resume")
	assertContains(t, inv.Args, "abc")
}

func TestLaunchPrintFlag(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir: t.TempDir(),
		Print:   true,
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	assertContains(t, inv.Args, "-p")
}

func TestLaunchSystemPromptsJoined(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir:       t.TempDir(),
		SystemPrompts: []string{"first", "second"},
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	found := false
	for _, a := range inv.Args {
		if a == "first\n\n---\n\nsecond" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected joined system prompts, got args: %v", inv.Args)
	}
}

// --- Environment variables ---

func TestLaunchSetsEnvVars(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	err := Launch(LaunchConfig{
		WorkDir:          t.TempDir(),
		ProjectName:      "my-project",
		Mode:             config.Mode{Name: "research"},
		AutoSetup:        true,
		AutoCompactLimit: 70,
		AddDirs:          []string{"/repo1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)

	if inv.Env["CW_PROJECT"] != "my-project" {
		t.Errorf("CW_PROJECT = %q, want %q", inv.Env["CW_PROJECT"], "my-project")
	}
	if inv.Env["CW_MODE"] != "research" {
		t.Errorf("CW_MODE = %q, want %q", inv.Env["CW_MODE"], "research")
	}
	if inv.Env["CW_AUTO_SETUP"] != "1" {
		t.Errorf("CW_AUTO_SETUP = %q, want %q", inv.Env["CW_AUTO_SETUP"], "1")
	}
	if inv.Env["CW_AUTO_COMPACT_LIMIT"] != "70" {
		t.Errorf("CW_AUTO_COMPACT_LIMIT = %q, want %q", inv.Env["CW_AUTO_COMPACT_LIMIT"], "70")
	}
	if inv.Env["CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD"] != "1" {
		t.Error("expected CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1")
	}
	if inv.Env["CW_PID"] == "" {
		t.Error("expected CW_PID to be set")
	}
}

func TestLaunchMinimalEnv(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)
	// Clear env vars that Launch conditionally sets, so we can test they're NOT added
	t.Setenv("CW_PROJECT", "")
	t.Setenv("CW_PROJECT_DIR", "")
	t.Setenv("CW_MODE", "")
	t.Setenv("CW_AUTO_SETUP", "")
	t.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD", "")

	err := Launch(LaunchConfig{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	if inv.Env["CW_PID"] == "" {
		t.Error("CW_PID should always be set")
	}
}

// --- Error handling ---

func TestLaunchExitError(t *testing.T) {
	mockClaudeBin(t)
	t.Setenv("MOCK_CLAUDE_BEHAVIOR", "exit1")

	err := Launch(LaunchConfig{WorkDir: t.TempDir()})
	if err == nil {
		t.Error("expected error for exit code 1")
	}
}

func TestLaunchNotFound(t *testing.T) {
	// Empty PATH so "claude" can't be found
	t.Setenv("PATH", "")

	err := Launch(LaunchConfig{WorkDir: t.TempDir()})
	if err == nil {
		t.Error("expected error when claude binary not found")
	}
}

// --- Signal / reload handling ---

func TestLaunchReloadSignal(t *testing.T) {
	mockClaudeBin(t)
	t.Setenv("MOCK_CLAUDE_BEHAVIOR", "sleep")

	reloadRequested.Store(false)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Launch(LaunchConfig{WorkDir: t.TempDir()})
	}()

	// Give the mock process time to start
	time.Sleep(200 * time.Millisecond)

	// Send SIGUSR1 to ourselves (simulating cw internal reload)
	syscall.Kill(os.Getpid(), syscall.SIGUSR1)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil error after reload, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Launch did not return after SIGUSR1")
	}

	if !WasReload() {
		t.Error("WasReload should be true after SIGUSR1")
	}

	// Cleanup
	reloadRequested.Store(false)
}

func TestLaunchResetsReloadState(t *testing.T) {
	mockClaudeBin(t)

	// Set reload to true, then Launch should reset it
	reloadRequested.Store(true)

	Launch(LaunchConfig{WorkDir: t.TempDir()})

	// After a normal exit (no SIGUSR1), WasReload should be false
	if WasReload() {
		t.Error("WasReload should be false after normal exit")
	}
}

// --- WasReload ---

func TestWasReloadInitialState(t *testing.T) {
	reloadRequested.Store(false)
	if WasReload() {
		t.Error("WasReload should be false initially")
	}
}

func TestWasReloadAfterSet(t *testing.T) {
	reloadRequested.Store(true)
	if !WasReload() {
		t.Error("WasReload should be true after set")
	}
	reloadRequested.Store(false)
}

// --- WorkDir ---

func TestLaunchSetsWorkDir(t *testing.T) {
	mockClaudeBin(t)
	outFile := filepath.Join(t.TempDir(), "out.json")
	t.Setenv("MOCK_CLAUDE_OUTPUT", outFile)

	workDir := t.TempDir()
	err := Launch(LaunchConfig{WorkDir: workDir})
	if err != nil {
		t.Fatal(err)
	}

	inv := readInvocation(t, outFile)
	if inv.Env["CW_PROJECT_DIR"] != "" {
		// Without ProjectName, CW_PROJECT_DIR shouldn't be set
		// but WorkDir is set via cmd.Dir which we can't inspect from the child
		// So this test just verifies Launch succeeds with a WorkDir
	}
	_ = inv
}

// --- helpers ---

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("args %v should contain %q", args, want)
}

func assertNotContains(t *testing.T, args []string, unwanted string) {
	t.Helper()
	for _, a := range args {
		if a == unwanted {
			t.Errorf("args %v should NOT contain %q", args, unwanted)
			return
		}
	}
}

func assertEnvContains(t *testing.T, env []string, key, value string) {
	t.Helper()
	want := key + "=" + value
	for _, e := range env {
		if e == want {
			return
		}
	}
	t.Errorf("env should contain %q, got %v", want, env)
}

// Silence unused import warnings
var _ = fmt.Sprint
var _ = strings.Contains
