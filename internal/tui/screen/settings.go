package screen

import (
	"fmt"
	"strings"

	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	tea "github.com/charmbracelet/bubbletea"
)

type settingKind int

const (
	settingToggle settingKind = iota
	settingCycle
	settingSelect
)

type settingItem struct {
	label   string
	desc    string
	kind    settingKind
	key     string // settings field identifier
	options []string
}

var settingsList = []settingItem{
	{label: "Bypass Permissions", desc: "Skip permission prompts when launching Claude", kind: settingToggle, key: "bypass_permissions"},
	{label: "Auto Compact", desc: "Context % threshold for auto-compaction (0 = off)", kind: settingCycle, key: "auto_compact_limit", options: []string{"off", "40%", "50%", "60%", "70%", "80%"}},
	{label: "Load Subproject Rules", desc: "Load CLAUDE.md and rules from each repo via --add-dir", kind: settingToggle, key: "load_subproject_rules"},
	{label: "Default Mode", desc: "Default mode for new sessions", kind: settingSelect, key: "default_mode"},
	{label: "Enable Hooks", desc: "Sync pre-write guard and auto-compact hooks", kind: settingToggle, key: "enable_hooks"},
}

type SettingsModel struct {
	settings project.Settings
	cursor   int
	width    int
	height   int
}

func NewSettingsModel() SettingsModel {
	return SettingsModel{
		settings: project.LoadGlobalSettings(),
	}
}

func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m SettingsModel) Init() tea.Cmd { return nil }

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg {
				return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
			}
		case "j", "down":
			if m.cursor < len(settingsList)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter", " ", "right", "l":
			m.toggleCurrent(1)
			project.SaveGlobalSettings(m.settings)
			return m, nil
		case "left", "h":
			m.toggleCurrent(-1)
			project.SaveGlobalSettings(m.settings)
			return m, nil
		}
	}
	return m, nil
}

func (m *SettingsModel) toggleCurrent(dir int) {
	item := settingsList[m.cursor]
	switch item.key {
	case "bypass_permissions":
		m.settings.BypassPermissions = !m.settings.BypassPermissions
	case "auto_compact_limit":
		values := []int{0, 40, 50, 60, 70, 80}
		idx := 0
		for i, v := range values {
			if v == m.settings.AutoCompactLimit {
				idx = i
				break
			}
		}
		idx += dir
		if idx < 0 {
			idx = len(values) - 1
		} else if idx >= len(values) {
			idx = 0
		}
		m.settings.AutoCompactLimit = values[idx]
	case "load_subproject_rules":
		m.settings.LoadSubprojectRules = !m.settings.LoadSubprojectRules
	case "default_mode":
		modes := config.Modes
		if len(modes) == 0 {
			return
		}
		idx := 0
		for i, mode := range modes {
			if mode.Name == m.settings.DefaultMode {
				idx = i
				break
			}
		}
		idx += dir
		if idx < 0 {
			idx = len(modes) - 1
		} else if idx >= len(modes) {
			idx = 0
		}
		m.settings.DefaultMode = modes[idx].Name
	case "enable_hooks":
		m.settings.EnableHooks = !m.settings.EnableHooks
	}
}

func (m SettingsModel) valueString(item settingItem) string {
	switch item.key {
	case "bypass_permissions":
		if m.settings.BypassPermissions {
			return style.ErrorStyle.Render("bypass")
		}
		return style.SuccessStyle.Render("normal")
	case "auto_compact_limit":
		if m.settings.AutoCompactLimit == 0 {
			return style.DimStyle.Render("off")
		}
		return style.AccentStyle.Render(fmt.Sprintf("%d%%", m.settings.AutoCompactLimit))
	case "load_subproject_rules":
		if m.settings.LoadSubprojectRules {
			return style.SuccessStyle.Render("on")
		}
		return style.DimStyle.Render("off")
	case "default_mode":
		name := m.settings.DefaultMode
		if name == "" {
			name = "code"
		}
		return style.AccentStyle.Render(name)
	case "enable_hooks":
		if m.settings.EnableHooks {
			return style.SuccessStyle.Render("on")
		}
		return style.DimStyle.Render("off")
	}
	return ""
}

// Settings returns the current settings so callers can read them.
func (m SettingsModel) Settings() project.Settings {
	return m.settings
}

func (m SettingsModel) View() string {
	header := style.TitleStyle.Render("SETTINGS") + "\n\n"

	var rows []string
	for i, item := range settingsList {
		cursor := "  "
		if i == m.cursor {
			cursor = style.FzfCursorPrefix.Render("▶ ")
		}

		value := m.valueString(item)
		line := fmt.Sprintf("%s%-24s %s", cursor, item.label, value)
		if i == m.cursor {
			line += "\n" + strings.Repeat(" ", 26) + style.DimStyle.Render(item.desc)
		}
		rows = append(rows, line)
	}

	body := strings.Join(rows, "\n")

	keybar := style.RenderKeybar(
		style.KeyBind{Key: "↑↓", Desc: "navigate"},
		style.KeyBind{Key: "←→", Desc: "change"},
		style.KeyBind{Key: "esc", Desc: "back"},
	)

	return header + body + "\n\n" + keybar
}
