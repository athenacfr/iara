package screen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/session"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/ahtwr/cw/internal/tui/widget"
	tea "github.com/charmbracelet/bubbletea"
)

type sessionItem struct {
	entry sessionEntry
}

func (s sessionItem) FilterValue() string { return s.entry.label }

type sessionEntry struct {
	label   string
	relTime string
	session *session.Session
	kind    int // 0=new, 1=list sessions, 2=resume specific session
}

type sessionsLoadedMsg struct {
	sessions []session.Session
	err      error
}

type LauncherModel struct {
	fzfList         widget.FzfListModel
	modes           []config.Mode
	modeIndex       int
	skipPermissions bool
	sessionsDir     string
	taskName        string
	width, height   int
}

func NewLauncherModel(bypass bool, sessionsDir string) LauncherModel {
	return newLauncherModel(bypass, sessionsDir, "")
}

func NewLauncherModelWithDefault(bypass bool, sessionsDir string, defaultMode string) LauncherModel {
	return newLauncherModel(bypass, sessionsDir, defaultMode)
}

func NewLauncherModelForTask(bypass bool, sessionsDir string, defaultMode string, taskName string) LauncherModel {
	m := newLauncherModel(bypass, sessionsDir, defaultMode)
	m.taskName = taskName
	return m
}

func newLauncherModel(bypass bool, sessionsDir string, defaultMode string) LauncherModel {
	entries := []sessionEntry{
		{label: "+ New Session", kind: 0},
		{label: "List sessions", kind: 1},
	}

	items := make([]widget.FzfItem, len(entries))
	for i, e := range entries {
		items[i] = sessionItem{entry: e}
	}

	fzf := widget.NewFzfList(items, widget.FzfListConfig{
		RenderItem:   renderSessionItem,
		PreviewFunc:  sessionPreview,
		Placeholder:  "No sessions",
		ListWidthPct: 0.5,
	})

	modeIdx := 0
	if defaultMode != "" {
		for i, m := range config.Modes {
			if m.Name == defaultMode {
				modeIdx = i
				break
			}
		}
	}

	return LauncherModel{
		fzfList:         fzf,
		modes:           config.Modes,
		modeIndex:       modeIdx,
		skipPermissions: bypass,
		sessionsDir:     sessionsDir,
	}
}

func (m LauncherModel) LoadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := session.List(m.sessionsDir)
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func (m LauncherModel) Init() tea.Cmd {
	return nil
}

func (m *LauncherModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.fzfList.SetSize(w, h-7)
}

func (m LauncherModel) Update(msg tea.Msg) (LauncherModel, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		if msg.err != nil || len(msg.sessions) == 0 {
			return m, nil
		}
		entries := []sessionEntry{
			{label: "+ New Session", kind: 0},
		}
		// Sessions come sorted by LastActive desc; assign numbers by StartedAt asc
		numberMap := sessionNumbers(msg.sessions)
		for i := range msg.sessions {
			s := msg.sessions[i]
			relTime := session.RelativeTime(session.ParseTime(s.LastActive))
			var label string
			if s.Summary != "" {
				label = s.Summary
			} else {
				num := numberMap[s.ID]
				label = fmt.Sprintf("Session #%d", num)
			}
			entries = append(entries, sessionEntry{label: label, relTime: relTime, session: &s, kind: 2})
		}
		items := make([]widget.FzfItem, len(entries))
		for i, e := range entries {
			items[i] = sessionItem{entry: e}
		}
		m.fzfList.SetItems(items)
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		if !m.fzfList.IsSearching() {
			switch key {
			case "tab":
				m.skipPermissions = !m.skipPermissions
				return m, nil
			case "left", "h":
				if m.modeIndex > 0 {
					m.modeIndex--
				}
				return m, nil
			case "right", "l":
				if m.modeIndex < len(m.modes)-1 {
					m.modeIndex++
				}
				return m, nil
			}
		}

		newList, consumed, result := m.fzfList.HandleKey(key)
		m.fzfList = newList

		if result != nil {
			switch r := result.(type) {
			case widget.FzfConfirmMsg:
				si := r.Item.(sessionItem)
				mode := m.modes[m.modeIndex]
				return m, func() tea.Msg {
					return shared.ModeSelectedMsg{
						Mode:            mode,
						SkipPermissions: m.skipPermissions,
						SessionKind:     si.entry.kind,
						ResumeSessionID: sessionEntryID(si.entry),
					}
				}
			case widget.FzfCancelMsg:
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenTaskSelect}
				}
			}
		}

		if consumed {
			return m, nil
		}
	}
	return m, nil
}

func sessionEntryID(entry sessionEntry) string {
	if entry.session != nil {
		return entry.session.ID
	}
	return ""
}

func (m LauncherModel) View() string {
	var b strings.Builder

	if m.taskName != "" {
		b.WriteString(style.AccentStyle.Render("TASK: " + m.taskName))
		b.WriteString("\n\n")
	}

	b.WriteString(style.TitleStyle.Render("MODE"))
	b.WriteString("\n")

	b.WriteString("  ")
	for i, mode := range m.modes {
		if i > 0 {
			b.WriteString("    ")
		}
		if i == m.modeIndex {
			b.WriteString(style.AccentStyle.Bold(true).Render(mode.Name))
		} else {
			b.WriteString(style.DimStyle.Render(mode.Name))
		}
	}
	b.WriteString("\n")

	if m.modeIndex < len(m.modes) {
		b.WriteString("  ")
		b.WriteString(style.DimStyle.Render(m.modes[m.modeIndex].Description))
	}
	b.WriteString("\n\n")

	b.WriteString(style.TitleStyle.Render("SESSIONS"))
	b.WriteString("\n")
	b.WriteString(m.fzfList.View())
	b.WriteString("\n")

	var permLabel string
	if m.skipPermissions {
		permLabel = style.SuccessStyle.Render("skip permissions")
	} else {
		permLabel = style.DimStyle.Render("normal permissions")
	}
	keybar := style.RenderKeybar(
		style.KeyBind{Key: "←→", Desc: "mode"},
		style.KeyBind{Key: "↑↓", Desc: "session"},
		style.KeyBind{Key: "tab", Desc: "permissions"},
		style.KeyBind{Key: "enter", Desc: "launch"},
	) + "  " + permLabel
	b.WriteString(keybar)

	return b.String()
}

func renderSessionItem(item widget.FzfItem, displayNum int, cursor, selected bool, matched []int, width int) string {
	si := item.(sessionItem)

	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "

	if si.entry.kind == 0 {
		return prefix + numStr + style.TreeAddStyle.Render(si.entry.label)
	}

	plain := si.entry.label
	timeSuffix := "  " + style.DimStyle.Render(si.entry.relTime)
	timePlainLen := 2 + len(si.entry.relTime)

	numWidth := len(fmt.Sprintf("%d.", displayNum)) + 1
	prefixWidth := 2 + numWidth
	contentWidth := width - prefixWidth

	// Reserve space for the time suffix
	labelWidth := contentWidth - timePlainLen
	if labelWidth <= 0 || contentWidth <= 0 {
		return prefix + numStr + widget.HighlightMatches(plain, matched) + timeSuffix
	}

	if len(plain) <= labelWidth {
		return prefix + numStr + widget.HighlightMatches(plain, matched) + timeSuffix
	}

	// Truncate the label, then append time
	truncated := widget.Truncate(widget.HighlightMatches(plain, matched), labelWidth)
	return prefix + numStr + truncated + timeSuffix
}

// sessionNumbers assigns sequential numbers to sessions based on StartedAt order.
// Returns a map of session ID → number (1-based, oldest = #1).
func sessionNumbers(sessions []session.Session) map[string]int {
	type idTime struct {
		id   string
		time string
	}
	sorted := make([]idTime, len(sessions))
	for i, s := range sessions {
		sorted[i] = idTime{id: s.ID, time: s.StartedAt}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].time < sorted[j].time
	})
	m := make(map[string]int, len(sorted))
	for i, s := range sorted {
		m[s.id] = i + 1
	}
	return m
}

func sessionPreview(item widget.FzfItem, width, height int) string {
	si := item.(sessionItem)

	var lines []string

	switch si.entry.kind {
	case 0:
		lines = append(lines, style.TitleStyle.Render("New Session"))
		lines = append(lines, "")
		lines = append(lines, style.DimStyle.Render("Start a fresh conversation with"))
		lines = append(lines, style.DimStyle.Render("updated system prompts and config."))
	case 1:
		lines = append(lines, style.TitleStyle.Render("List Sessions"))
		lines = append(lines, "")
		lines = append(lines, style.DimStyle.Render("Launch Claude with /resume to"))
		lines = append(lines, style.DimStyle.Render("browse and resume a session."))
	case 2:
		if si.entry.session != nil {
			s := si.entry.session
			lines = append(lines, style.TitleStyle.Render("Session"))
			lines = append(lines, "")
			if s.Summary != "" {
				lines = append(lines, s.Summary)
			}
			lines = append(lines, "")
			lines = append(lines, style.DimStyle.Render(session.RelativeTime(session.ParseTime(s.LastActive))))
			lines = append(lines, "")
			lines = append(lines, style.DimStyle.Render("ID: "+s.ID[:min(12, len(s.ID))]))
		}
	}

	return strings.Join(lines, "\n")
}
