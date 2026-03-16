package embed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallToDir(t *testing.T) {
	tmp := t.TempDir()

	if err := installToDir(tmp); err != nil {
		t.Fatalf("installToDir failed: %v", err)
	}

	// Check that key files were extracted
	expectedFiles := []string{
		"plugins/.claude-plugin/plugin.json",
		"plugins/commands/help.md",
		"plugins/commands/mode.md",
		"plugins/commands/new-task.md",
		"hooks/pre-write-guard.sh",
		"agents/researcher.md",
		"agents/reviewer.md",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tmp, f)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected file %s to have content", f)
		}
	}
}

func TestInstallSetsDir(t *testing.T) {
	tmp := t.TempDir()

	if err := installToDir(tmp); err != nil {
		t.Fatal(err)
	}

	if got := Dir(); got != tmp {
		t.Errorf("Dir() = %q, want %q", got, tmp)
	}
	if got := PluginDir(); got != filepath.Join(tmp, "plugins") {
		t.Errorf("PluginDir() = %q, want %q", got, filepath.Join(tmp, "plugins"))
	}
	if got := HooksDir(); got != filepath.Join(tmp, "hooks") {
		t.Errorf("HooksDir() = %q, want %q", got, filepath.Join(tmp, "hooks"))
	}
}

func TestInstallIdempotent(t *testing.T) {
	tmp := t.TempDir()

	if err := installToDir(tmp); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if err := installToDir(tmp); err != nil {
		t.Fatalf("second install failed: %v", err)
	}
}

func TestPluginJSONContent(t *testing.T) {
	tmp := t.TempDir()

	if err := installToDir(tmp); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "plugins", ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("plugin.json is empty")
	}
	// Should contain the plugin name
	if !contains(content, "cw") {
		t.Error("plugin.json should contain 'cw'")
	}
}

// --- Install (via paths.DataDir) ---

func TestInstall(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CW_DATA_DIR", tmp)

	if err := Install(); err != nil {
		t.Fatal(err)
	}

	if Dir() != tmp {
		t.Errorf("Dir() = %q, want %q", Dir(), tmp)
	}

	// Verify some files exist
	if _, err := os.Stat(filepath.Join(tmp, "plugins", ".claude-plugin", "plugin.json")); err != nil {
		t.Error("expected plugin.json after Install()")
	}
}

// --- Cleanup of old files ---

func TestInstallCleansOldFiles(t *testing.T) {
	tmp := t.TempDir()

	// First install
	installToDir(tmp)

	// Add a stale file in a managed directory
	stale := filepath.Join(tmp, "plugins", "commands", "stale-old-command.md")
	os.WriteFile(stale, []byte("old"), 0644)

	// Re-install — stale file should be cleaned
	installToDir(tmp)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale file should be removed after re-install")
	}
}

func TestInstallCleansOldHooks(t *testing.T) {
	tmp := t.TempDir()

	installToDir(tmp)

	stale := filepath.Join(tmp, "hooks", "stale-hook.sh")
	os.WriteFile(stale, []byte("old"), 0644)

	installToDir(tmp)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale hook file should be removed")
	}
}

// --- generatePluginsFromCommands ---

func TestGeneratedCommandPlugins(t *testing.T) {
	tmp := t.TempDir()
	installToDir(tmp)

	commandsDir := filepath.Join(tmp, "plugins", "commands")

	// CLI commands should generate .md files with "cw internal" invocation
	cliGenerated := []string{"compact-and-continue.md", "new-session.md", "reload.md", "open-project.md"}
	for _, name := range cliGenerated {
		path := filepath.Join(commandsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("missing generated plugin: %s: %v", name, err)
			continue
		}
		content := string(data)
		if !contains(content, "cw internal") {
			t.Errorf("%s should contain 'cw internal'", name)
		}
	}

	// Prompt commands should have PluginBody content
	promptGenerated := []string{"mode.md", "permissions.md", "help.md", "new-task.md", "finish-task.md", "setup-project.md"}
	for _, name := range promptGenerated {
		path := filepath.Join(commandsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("missing generated plugin: %s: %v", name, err)
			continue
		}
		if len(data) < 50 {
			t.Errorf("%s seems too short (%d bytes)", name, len(data))
		}
	}
}

func TestGeneratedPluginsHaveFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	installToDir(tmp)

	commandsDir := filepath.Join(tmp, "plugins", "commands")
	entries, _ := os.ReadDir(commandsDir)

	for _, e := range entries {
		if e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(commandsDir, e.Name()))
		content := string(data)
		if !contains(content, "---") {
			t.Errorf("%s should have YAML frontmatter", e.Name())
		}
		if !contains(content, "description:") {
			t.Errorf("%s should have description in frontmatter", e.Name())
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
