package config

import (
	"testing"
)

func TestInitModes(t *testing.T) {
	InitModes()

	if len(Modes) == 0 {
		t.Fatal("expected modes to be initialized")
	}

	// Should contain the standard modes
	names := make(map[string]bool)
	for _, m := range Modes {
		names[m.Name] = true
	}

	required := []string{"code", "research", "review", "none"}
	for _, name := range required {
		if !names[name] {
			t.Errorf("missing mode: %s", name)
		}
	}
}

func TestGetModeFound(t *testing.T) {
	InitModes()

	m, ok := GetMode("code")
	if !ok {
		t.Fatal("expected to find 'code' mode")
	}
	if m.Name != "code" {
		t.Errorf("Name = %q, want %q", m.Name, "code")
	}
}

func TestGetModeNotFound(t *testing.T) {
	InitModes()

	_, ok := GetMode("nonexistent")
	if ok {
		t.Error("expected false for nonexistent mode")
	}
}

func TestGetModeEmptyName(t *testing.T) {
	InitModes()

	_, ok := GetMode("")
	if ok {
		t.Error("expected false for empty mode name")
	}
}

func TestGetModeResearchHasAgent(t *testing.T) {
	InitModes()

	m, ok := GetMode("research")
	if !ok {
		t.Fatal("expected to find 'research' mode")
	}
	if m.Agent != "researcher" {
		t.Errorf("Agent = %q, want %q", m.Agent, "researcher")
	}
}

func TestGetModeReviewHasAgent(t *testing.T) {
	InitModes()

	m, ok := GetMode("review")
	if !ok {
		t.Fatal("expected to find 'review' mode")
	}
	if m.Agent != "reviewer" {
		t.Errorf("Agent = %q, want %q", m.Agent, "reviewer")
	}
}

func TestGetModeCodeHasNoAgent(t *testing.T) {
	InitModes()

	m, ok := GetMode("code")
	if !ok {
		t.Fatal("expected to find 'code' mode")
	}
	if m.Agent != "" {
		t.Errorf("code mode should have no Agent, got %q", m.Agent)
	}
}

func TestGetModeNoneHasNoAgent(t *testing.T) {
	InitModes()

	m, ok := GetMode("none")
	if !ok {
		t.Fatal("expected to find 'none' mode")
	}
	if m.Agent != "" {
		t.Errorf("none mode should have no Agent, got %q", m.Agent)
	}
}

func TestInitModesOverwritesPrevious(t *testing.T) {
	InitModes()
	first := len(Modes)

	InitModes()
	second := len(Modes)

	if first != second {
		t.Errorf("InitModes should overwrite, not append: %d vs %d", first, second)
	}
}
