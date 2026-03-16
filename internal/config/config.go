package config

type Mode struct {
	Name        string
	Description string
	Agent       string // agent name passed via --agent (e.g., "researcher", "reviewer")
}

var Modes []Mode

// GetMode looks up a mode by name. Returns the mode and true if found.
func GetMode(name string) (Mode, bool) {
	for _, m := range Modes {
		if m.Name == name {
			return m, true
		}
	}
	return Mode{}, false
}

func InitModes() {
	Modes = []Mode{
		{Name: "code", Description: "Write features, fix bugs (default)"},
		{Name: "research", Description: "Explore codebase, read-only", Agent: "researcher"},
		{Name: "review", Description: "Review code changes", Agent: "reviewer"},
		{Name: "none", Description: "No preset behavior"},
	}
}
