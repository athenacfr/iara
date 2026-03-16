package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadGlobalSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CW_DATA_DIR", dir)

	s := Settings{
		BypassPermissions:   true,
		AutoCompactLimit:    60,
		LoadSubprojectRules: true,
		DefaultMode:         "research",
		EnableHooks:         true,
	}
	if err := SaveGlobalSettings(s); err != nil {
		t.Fatal(err)
	}

	loaded := LoadGlobalSettings()
	if loaded.BypassPermissions != true {
		t.Error("expected BypassPermissions=true")
	}
	if loaded.AutoCompactLimit != 60 {
		t.Errorf("AutoCompactLimit = %d, want 60", loaded.AutoCompactLimit)
	}
	if loaded.LoadSubprojectRules != true {
		t.Error("expected LoadSubprojectRules=true")
	}
	if loaded.DefaultMode != "research" {
		t.Errorf("DefaultMode = %q, want %q", loaded.DefaultMode, "research")
	}
	if loaded.EnableHooks != true {
		t.Error("expected EnableHooks=true")
	}
}

func TestLoadGlobalSettingsDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CW_DATA_DIR", dir)

	// No settings file exists — should return default
	s := LoadGlobalSettings()
	if !s.BypassPermissions {
		t.Error("default should have BypassPermissions=true")
	}
	if !s.LoadSubprojectRules {
		t.Error("default should have LoadSubprojectRules=true")
	}
	if !s.EnableHooks {
		t.Error("default should have EnableHooks=true")
	}
}

func TestLoadGlobalSettingsCorrupted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CW_DATA_DIR", dir)

	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("not json"), 0644)

	s := LoadGlobalSettings()
	if !s.BypassPermissions {
		t.Error("corrupted file should return default with BypassPermissions=true")
	}
}

func TestSaveGlobalSettingsOverwrites(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CW_DATA_DIR", dir)

	SaveGlobalSettings(Settings{BypassPermissions: true})
	SaveGlobalSettings(Settings{BypassPermissions: false, AutoCompactLimit: 40})

	loaded := LoadGlobalSettings()
	if loaded.BypassPermissions != false {
		t.Error("expected BypassPermissions=false after overwrite")
	}
	if loaded.AutoCompactLimit != 40 {
		t.Errorf("AutoCompactLimit = %d, want 40", loaded.AutoCompactLimit)
	}
}

func TestLoadGlobalSettingsMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CW_DATA_DIR", dir)

	// Write old-format settings (only bypass_permissions and auto_compact_limit)
	old := []byte(`{"bypass_permissions": false, "auto_compact_limit": 50}`)
	os.WriteFile(filepath.Join(dir, "settings.json"), old, 0644)

	s := LoadGlobalSettings()
	if s.BypassPermissions != false {
		t.Error("expected BypassPermissions=false from file")
	}
	if s.AutoCompactLimit != 50 {
		t.Errorf("AutoCompactLimit = %d, want 50", s.AutoCompactLimit)
	}
	// New fields should get defaults since they start from DefaultSettings()
	if !s.LoadSubprojectRules {
		t.Error("migration should default LoadSubprojectRules=true")
	}
	if !s.EnableHooks {
		t.Error("migration should default EnableHooks=true")
	}
}

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if !s.BypassPermissions {
		t.Error("default BypassPermissions should be true")
	}
	if !s.LoadSubprojectRules {
		t.Error("default LoadSubprojectRules should be true")
	}
	if !s.EnableHooks {
		t.Error("default EnableHooks should be true")
	}
	if s.DefaultMode != "" {
		t.Errorf("default DefaultMode should be empty, got %q", s.DefaultMode)
	}
	if s.AutoCompactLimit != 0 {
		t.Errorf("default AutoCompactLimit should be 0, got %d", s.AutoCompactLimit)
	}
}

func TestGlobalSettingsPath(t *testing.T) {
	t.Setenv("CW_DATA_DIR", "/test/data")
	got := globalSettingsPath()
	if got != "/test/data/settings.json" {
		t.Errorf("globalSettingsPath = %q, want %q", got, "/test/data/settings.json")
	}
}
