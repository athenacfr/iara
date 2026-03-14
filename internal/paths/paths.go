package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the cw data directory for embedded files, modes, hooks, etc.
// Override with CW_DATA_DIR env var.
// Defaults: macOS ~/Library/Application Support/cw, Linux/WSL ~/.local/share/cw
func DataDir() string {
	if dir := os.Getenv("CW_DATA_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "cw")
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "cw")
	}
	return filepath.Join(home, ".local", "share", "cw")
}

// ProjectsDir returns the directory where cw projects live.
// Override with CW_PROJECTS_DIR env var.
// Default: ~/cw/projects
func ProjectsDir() string {
	if dir := os.Getenv("CW_PROJECTS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "cw", "projects")
}

// EnsureProjectsDir creates the projects directory if it doesn't exist.
func EnsureProjectsDir() error {
	return os.MkdirAll(ProjectsDir(), 0755)
}

// EnvsDir returns the directory for global env files.
// Override with CW_ENVS_DIR env var.
// Default: ~/cw/envs
func EnvsDir() string {
	if dir := os.Getenv("CW_ENVS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "cw", "envs")
}

// ModeOverrideFile returns the path to the sideband file used to communicate
// mode switches from a Claude session to the parent cw process during reload.
func ModeOverrideFile() string {
	return filepath.Join(DataDir(), "mode-override")
}

// PermissionsOverrideFile returns the path to the sideband file used to communicate
// permission switches from a Claude session to the parent cw process during reload.
func PermissionsOverrideFile() string {
	return filepath.Join(DataDir(), "permissions-override")
}

// NewSessionFile returns the path to the sideband file that signals the reload
// loop to start a fresh session instead of resuming the previous one.
func NewSessionFile() string {
	return filepath.Join(DataDir(), "new-session")
}

// AutoCompactFile returns the path to the sideband file that signals the reload
// loop to restart with /compact as the initial prompt.
func AutoCompactFile() string {
	return filepath.Join(DataDir(), "auto-compact")
}

// CompactContextFile returns the path to the sideband file that stores
// task context extracted from the session JSONL before auto-compacting.
// Used to continue the task after compact completes.
func CompactContextFile() string {
	return filepath.Join(DataDir(), "compact-context")
}

// YoloActiveFile returns the path to the sideband file that signals the reload
// loop to enter yolo execution mode. Contains the absolute path to the plan file.
func YoloActiveFile() string {
	return filepath.Join(DataDir(), "yolo-active")
}

// BinDir returns the directory where the cw binary is installed.
// Override with CW_BIN_DIR env var.
// Defaults: macOS /usr/local/bin, Linux/WSL ~/.local/bin
func BinDir() string {
	if dir := os.Getenv("CW_BIN_DIR"); dir != "" {
		return dir
	}
	if runtime.GOOS == "darwin" {
		return "/usr/local/bin"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}
