package embed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:files
var embeddedFS embed.FS

var installDir string

// Dir returns the base install directory.
func Dir() string {
	return installDir
}

// PluginDir returns the path to the extracted plugins directory.
func PluginDir() string {
	return filepath.Join(installDir, "plugins")
}

// ModesDir returns the path to the extracted modes directory.
func ModesDir() string {
	return filepath.Join(installDir, "modes")
}

// HooksDir returns the path to the extracted hooks directory.
func HooksDir() string {
	return filepath.Join(installDir, "hooks")
}

// Install extracts embedded files to ~/.local/share/cw/.
func Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return installToDir(filepath.Join(home, ".local", "share", "cw"))
}

func installToDir(dest string) error {
	installDir = dest

	return fs.WalkDir(embeddedFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel("files", path)
		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o755)
	})
}
