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
// Default: ~/development/
func ProjectsDir() string {
	if dir := os.Getenv("CW_PROJECTS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "development")
}

// EnsureProjectsDir creates the projects directory if it doesn't exist.
func EnsureProjectsDir() error {
	return os.MkdirAll(ProjectsDir(), 0755)
}

// EnvsDir returns the directory for global env files.
// Override with CW_ENVS_DIR env var.
// Default: ~/development/dotenvs
func EnvsDir() string {
	if dir := os.Getenv("CW_ENVS_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "development", "dotenvs")
}

// BinDir returns the directory where the cw binary is installed.
// Default: ~/.local/bin on all platforms.
func BinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}
