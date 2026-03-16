package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ahtwr/cw/internal/commands"
	"github.com/ahtwr/cw/internal/paths"
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

// HooksDir returns the path to the extracted hooks directory.
func HooksDir() string {
	return filepath.Join(installDir, "hooks")
}

// AgentsDir returns the path to the extracted agents directory.
func AgentsDir() string {
	return filepath.Join(installDir, "agents")
}

// Install extracts embedded files to the platform-specific data directory.
func Install() error {
	return installToDir(paths.DataDir())
}

func installToDir(dest string) error {
	installDir = dest

	// Build set of embedded file paths
	embedded := make(map[string]bool)
	if err := fs.WalkDir(embeddedFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel("files", path)
			embedded[rel] = true
		}
		return nil
	}); err != nil {
		return err
	}

	// Write all embedded files
	if err := fs.WalkDir(embeddedFS, "files", func(path string, d fs.DirEntry, err error) error {
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
	}); err != nil {
		return err
	}

	// Auto-generate .md plugin stubs from command registry
	generated, err := generatePluginsFromCommands(dest)
	if err != nil {
		return err
	}
	for _, p := range generated {
		rel, _ := filepath.Rel(dest, p)
		embedded[rel] = true
	}

	// Clean up files on disk that are no longer embedded
	managedDirs := []string{"plugins", "hooks", "agents"}
	for _, dir := range managedDirs {
		dirPath := filepath.Join(dest, dir)
		filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dest, path)
			if !embedded[rel] {
				os.Remove(path)
			}
			return nil
		})
	}

	return nil
}

// generatePluginsFromCommands creates .md plugin stubs from the command registry.
func generatePluginsFromCommands(dest string) ([]string, error) {
	commandsDir := filepath.Join(dest, "plugins", "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		return nil, err
	}

	var generated []string
	for _, cmd := range commands.Public() {
		mdPath := filepath.Join(commandsDir, cmd.Name+".md")

		var body string
		if cmd.PluginBody != "" {
			body = cmd.PluginBody
		} else if cmd.CLICommand != "" {
			body = fmt.Sprintf("Run `cw internal %s` using the Bash tool. Do not say anything else.", cmd.CLICommand)
		} else {
			continue
		}

		md := fmt.Sprintf("---\ndescription: %s\n---\n\n%s\n", cmd.Description, body)
		if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
			return nil, err
		}
		generated = append(generated, mdPath)
	}

	return generated, nil
}
