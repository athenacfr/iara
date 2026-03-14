package commands

import (
	"testing"
)

func TestRegisteredCommands(t *testing.T) {
	// The init() function registers all commands
	all := All()
	if len(all) == 0 {
		t.Fatal("expected commands to be registered via init()")
	}
}

func TestRequiredCommandsExist(t *testing.T) {
	required := []string{
		"compact-and-continue",
		"new-session",
		"reload",
		"open-project",
		"switch-mode",
		"switch-permissions",
		"save-metadata",
		"mode",
		"permissions",
		"help",
		"worktree",
		"new-intention",
		"yolo",
		"yolo-start",
		"yolo-stop",
	}

	all := All()
	names := make(map[string]bool)
	for _, c := range all {
		names[c.Name] = true
	}

	for _, name := range required {
		if !names[name] {
			t.Errorf("missing required command: %s", name)
		}
	}
}

func TestCLICommandsHaveCLICommand(t *testing.T) {
	// Commands that map to internal CLI should have CLICommand set
	cliCommands := []string{"compact-and-continue", "new-session", "reload", "open-project", "switch-mode", "switch-permissions", "save-metadata", "yolo-start", "yolo-stop"}

	byName := commandsByName()
	for _, name := range cliCommands {
		cmd, ok := byName[name]
		if !ok {
			t.Errorf("command %q not found", name)
			continue
		}
		if cmd.CLICommand == "" {
			t.Errorf("command %q should have CLICommand set", name)
		}
	}
}

func TestPromptCommandsHavePluginBody(t *testing.T) {
	// Prompt-only commands should have PluginBody set
	promptCommands := []string{"mode", "permissions", "help", "worktree", "new-intention", "yolo"}

	byName := commandsByName()
	for _, name := range promptCommands {
		cmd, ok := byName[name]
		if !ok {
			t.Errorf("command %q not found", name)
			continue
		}
		if cmd.PluginBody == "" {
			t.Errorf("command %q should have PluginBody set", name)
		}
	}
}

func TestAllCommandsHaveDescription(t *testing.T) {
	for _, cmd := range All() {
		if cmd.Description == "" {
			t.Errorf("command %q has empty description", cmd.Name)
		}
	}
}

func TestAllCommandsHaveName(t *testing.T) {
	for _, cmd := range All() {
		if cmd.Name == "" {
			t.Error("found command with empty name")
		}
	}
}

func TestInternalCommandsMarked(t *testing.T) {
	internal := []string{"switch-mode", "switch-permissions", "save-metadata", "yolo-start", "yolo-stop"}
	byName := commandsByName()

	for _, name := range internal {
		cmd, ok := byName[name]
		if !ok {
			t.Errorf("command %q not found", name)
			continue
		}
		if !cmd.Internal {
			t.Errorf("command %q should be marked Internal", name)
		}
	}
}

func TestPublicExcludesInternal(t *testing.T) {
	pub := Public()
	for _, cmd := range pub {
		if cmd.Internal {
			t.Errorf("Public() returned internal command: %s", cmd.Name)
		}
	}

	// Verify internal commands are not in public list
	pubNames := make(map[string]bool)
	for _, c := range pub {
		pubNames[c.Name] = true
	}
	for _, name := range []string{"switch-mode", "switch-permissions", "save-metadata", "yolo-start", "yolo-stop"} {
		if pubNames[name] {
			t.Errorf("internal command %q should not be in Public()", name)
		}
	}
}

func TestPublicCommandsNotInternal(t *testing.T) {
	publicNames := []string{"compact-and-continue", "new-session", "reload", "open-project", "mode", "permissions", "help", "worktree", "yolo", "new-intention"}
	byName := commandsByName()

	for _, name := range publicNames {
		cmd, ok := byName[name]
		if !ok {
			t.Errorf("command %q not found", name)
			continue
		}
		if cmd.Internal {
			t.Errorf("command %q should NOT be marked Internal", name)
		}
	}
}

func TestNoDuplicateCommandNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, cmd := range All() {
		if seen[cmd.Name] {
			t.Errorf("duplicate command name: %s", cmd.Name)
		}
		seen[cmd.Name] = true
	}
}

func TestSwitchModeHasParams(t *testing.T) {
	byName := commandsByName()
	cmd := byName["switch-mode"]

	if cmd.Params == nil {
		t.Fatal("switch-mode should have params")
	}
	modeDef, ok := cmd.Params["mode"]
	if !ok {
		t.Fatal("switch-mode should have 'mode' param")
	}
	if !modeDef.Required {
		t.Error("mode param should be required")
	}
	if len(modeDef.Enum) == 0 {
		t.Error("mode param should have enum values")
	}
}

func TestSwitchPermissionsHasParams(t *testing.T) {
	byName := commandsByName()
	cmd := byName["switch-permissions"]

	if cmd.Params == nil {
		t.Fatal("switch-permissions should have params")
	}
	valueDef, ok := cmd.Params["value"]
	if !ok {
		t.Fatal("switch-permissions should have 'value' param")
	}
	if !valueDef.Required {
		t.Error("value param should be required")
	}
	if len(valueDef.Enum) != 2 {
		t.Errorf("expected 2 enum values (bypass, normal), got %d", len(valueDef.Enum))
	}
}

func TestSaveMetadataHasParams(t *testing.T) {
	byName := commandsByName()
	cmd := byName["save-metadata"]

	if cmd.Params == nil {
		t.Fatal("save-metadata should have params")
	}
	jsonDef, ok := cmd.Params["json"]
	if !ok {
		t.Fatal("save-metadata should have 'json' param")
	}
	if !jsonDef.Required {
		t.Error("json param should be required")
	}
}

func TestRegisterAddsCommand(t *testing.T) {
	before := len(All())
	Register(Command{
		Name:        "test_command",
		Description: "A test command",
	})
	after := len(All())

	if after != before+1 {
		t.Errorf("Register should add 1 command: before=%d, after=%d", before, after)
	}
}

// helper
func commandsByName() map[string]Command {
	m := make(map[string]Command)
	for _, c := range All() {
		m[c.Name] = c
	}
	return m
}
