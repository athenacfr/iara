package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- DataDir ---

func TestDataDirDefault(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "")
	dir := DataDir()
	if dir == "" {
		t.Error("DataDir should not be empty")
	}
	if !strings.Contains(dir, "cw") {
		t.Errorf("DataDir = %q, expected to contain 'cw'", dir)
	}
}

func TestDataDirOverride(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/custom/data")
	dir := DataDir()
	if dir != "/custom/data" {
		t.Errorf("DataDir = %q, want %q", dir, "/custom/data")
	}
}

func TestDataDirXDG(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	dir := DataDir()
	if dir != "/xdg/data/cw" {
		t.Errorf("DataDir with XDG = %q, want %q", dir, "/xdg/data/cw")
	}
}

// --- ProjectsDir ---

func TestProjectsDirDefault(t *testing.T) {
	t.Setenv("CW_PROJECTS_DIR", "")
	dir := ProjectsDir()
	if dir == "" {
		t.Error("ProjectsDir should not be empty")
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "cw", "projects")
	if dir != expected {
		t.Errorf("ProjectsDir = %q, want %q", dir, expected)
	}
}

func TestProjectsDirOverride(t *testing.T) {
	t.Setenv("CW_PROJECTS_DIR", "/custom/projects")
	dir := ProjectsDir()
	if dir != "/custom/projects" {
		t.Errorf("ProjectsDir = %q, want %q", dir, "/custom/projects")
	}
}

// --- EnvsDir ---

func TestEnvsDirDefault(t *testing.T) {
	t.Setenv("CW_ENVS_DIR", "")
	dir := EnvsDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "cw", "envs")
	if dir != expected {
		t.Errorf("EnvsDir = %q, want %q", dir, expected)
	}
}

func TestEnvsDirOverride(t *testing.T) {
	t.Setenv("CW_ENVS_DIR", "/custom/envs")
	dir := EnvsDir()
	if dir != "/custom/envs" {
		t.Errorf("EnvsDir = %q, want %q", dir, "/custom/envs")
	}
}

// --- BinDir ---

func TestBinDirDefault(t *testing.T) {
	t.Setenv("CW_BIN_DIR", "")
	dir := BinDir()
	if dir == "" {
		t.Error("BinDir should not be empty")
	}
}

func TestBinDirOverride(t *testing.T) {
	t.Setenv("CW_BIN_DIR", "/custom/bin")
	dir := BinDir()
	if dir != "/custom/bin" {
		t.Errorf("BinDir = %q, want %q", dir, "/custom/bin")
	}
}

// --- Sideband files ---

func TestModeOverrideFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := ModeOverrideFile()
	if got != "/data/mode-override" {
		t.Errorf("ModeOverrideFile = %q, want %q", got, "/data/mode-override")
	}
}

func TestPermissionsOverrideFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := PermissionsOverrideFile()
	if got != "/data/permissions-override" {
		t.Errorf("PermissionsOverrideFile = %q, want %q", got, "/data/permissions-override")
	}
}

func TestNewSessionFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := NewSessionFile()
	if got != "/data/new-session" {
		t.Errorf("NewSessionFile = %q, want %q", got, "/data/new-session")
	}
}

func TestAutoCompactFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := AutoCompactFile()
	if got != "/data/auto-compact" {
		t.Errorf("AutoCompactFile = %q, want %q", got, "/data/auto-compact")
	}
}

// --- EnsureProjectsDir ---

func TestEnsureProjectsDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CW_PROJECTS_DIR", filepath.Join(tmp, "projects"))

	err := EnsureProjectsDir()
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(tmp, "projects"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestEnsureProjectsDirIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CW_PROJECTS_DIR", filepath.Join(tmp, "projects"))

	if err := EnsureProjectsDir(); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectsDir(); err != nil {
		t.Fatal(err)
	}
}

// --- CompactContextFile ---

func TestCompactContextFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := CompactContextFile()
	if got != "/data/compact-context" {
		t.Errorf("CompactContextFile = %q, want %q", got, "/data/compact-context")
	}
}

// --- DataDir darwin branch (can't test on linux, but test XDG fallthrough) ---

func TestDataDirNoXDG(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "")
	dir := DataDir()
	// On linux with no XDG, should fall through to ~/.local/share/cw
	home, _ := os.UserHomeDir()
	if dir == "" {
		t.Error("DataDir should not be empty")
	}
	// Should end with /cw
	if !strings.HasSuffix(dir, "/cw") {
		t.Errorf("DataDir = %q, expected to end with /cw", dir)
	}
	_ = home
}

// --- YoloActiveFile ---

func TestYoloActiveFile(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/data")
	got := YoloActiveFile()
	if got != "/data/yolo-active" {
		t.Errorf("YoloActiveFile = %q, want %q", got, "/data/yolo-active")
	}
}

// --- BinDir no override, no darwin ---

func TestBinDirNoOverride(t *testing.T) {
	t.Setenv("CW_BIN_DIR", "")
	dir := BinDir()
	if dir == "" {
		t.Error("BinDir should not be empty")
	}
}
