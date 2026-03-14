package screen

import (
	"fmt"
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
	width, height   int
}

func NewLauncherModel(bypass bool, projectDir string) LauncherModel {
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

	return LauncherModel{
		fzfList:         fzf,
		modes:           config.Modes,
		modeIndex:       0,
		skipPermissions: bypass,
	}
}

func (m LauncherModel) LoadSessions(projectDir string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := session.List(projectDir)
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
		for i := range msg.sessions {
			s := msg.sessions[i]
			label := fmt.Sprintf("%s  %s", session.RelativeTime(session.ParseTime(s.LastActive)), s.Summary)
			entries = append(entries, sessionEntry{label: label, session: &s, kind: 2})
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
						SessionID:       sessionEntryID(si.entry),
					}
				}
			case widget.FzfCancelMsg:
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
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

func renderSessionItem(item widget.FzfItem, index int, cursor, selected bool, matched []int, width int) string {
	si := item.(sessionItem)

	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}

	if si.entry.kind == 0 {
		return prefix + style.TreeAddStyle.Render(si.entry.label)
	}

	plain := si.entry.label
	prefixWidth := 2
	contentWidth := width - prefixWidth

	if contentWidth <= 0 || len(plain) <= contentWidth {
		return prefix + widget.HighlightMatches(plain, matched)
	}

	splitAt := contentWidth
	if idx := strings.LastIndex(plain[:splitAt], " "); idx > splitAt/2 {
		splitAt = idx + 1
	}

	line1 := plain[:splitAt]
	line2 := strings.TrimLeft(plain[splitAt:], " ")

	var matched1, matched2 []int
	for _, m := range matched {
		if m < splitAt {
			matched1 = append(matched1, m)
		} else if m-splitAt < len(plain[splitAt:]) {
			adj := m - splitAt + (len(plain[splitAt:]) - len(line2))
			if adj >= 0 && adj < len(line2) {
				matched2 = append(matched2, adj)
			}
		}
	}

	result := prefix + widget.HighlightMatches(line1, matched1)
	indent := strings.Repeat(" ", prefixWidth)
	wrapped := indent + widget.Truncate(widget.HighlightMatches(line2, matched2), contentWidth)
	return result + "\n" + wrapped
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
