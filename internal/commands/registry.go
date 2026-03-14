package commands

// Command defines a cw command with its metadata for plugin generation.
type Command struct {
	Name        string
	Description string
	Params      map[string]ParamDef // nil means no params
	CLICommand  string              // internal CLI subcommand (e.g. "reload", "mode-switch"); empty for prompt-only commands
	PluginBody  string              // custom .md body for plugin generation; if empty, auto-generated from CLICommand
	Internal    bool                // internal/plumbing commands not exposed as user-facing plugins
}

// ParamDef describes a single command parameter.
type ParamDef struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Required    bool     `json:"-"`
}

var commands []Command

// Register adds a command to the global registry.
func Register(c Command) {
	commands = append(commands, c)
}

// All returns all registered commands.
func All() []Command {
	return commands
}

// Public returns only user-facing commands (excludes internal/plumbing commands).
func Public() []Command {
	var pub []Command
	for _, c := range commands {
		if !c.Internal {
			pub = append(pub, c)
		}
	}
	return pub
}
