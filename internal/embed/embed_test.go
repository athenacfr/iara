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
		"plugins/commands/worktree.md",
		"plugins/commands/help.md",
		"plugins/commands/mode.md",
		"plugins/commands/new-intention.md",
		"modes/research.md",
		"modes/review.md",
		"hooks/pre-write-guard.sh",
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
	if got := ModesDir(); got != filepath.Join(tmp, "modes") {
		t.Errorf("ModesDir() = %q, want %q", got, filepath.Join(tmp, "modes"))
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
