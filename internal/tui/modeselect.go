package tui

import (
	"strings"

	"github.com/ahtwr/cw/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// modeItem wraps a mode for the fzf list.
type modeItem struct {
	config.Mode
}

func (m modeItem) FilterValue() string { return m.Name + " " + m.Description }

type modeSelectModel struct {
	fzfList       FzfListModel
	width, height int
}

func newModeSelectModel() modeSelectModel {
	items := make([]FzfItem, len(config.Modes))
	for i, m := range config.Modes {
		items[i] = modeItem{m}
	}
	fzf := NewFzfList(items, FzfListConfig{
		RenderItem:   renderModeItem,
		PreviewFunc:  modePreview,
		Placeholder:  "No modes",
		ListWidthPct: 0.4,
	})
	return modeSelectModel{fzfList: fzf}
}

func (m modeSelectModel) Init() tea.Cmd {
	return nil
}

func (m *modeSelectModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.fzfList.SetSize(w, h-3)
}

func (m modeSelectModel) update(msg tea.Msg) (modeSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		newList, consumed, result := m.fzfList.HandleKey(msg.String())
		m.fzfList = newList

		if result != nil {
			switch r := result.(type) {
			case FzfConfirmMsg:
				mi := r.Item.(modeItem)
				return m, func() tea.Msg {
					return modeSelectedMsg{mode: mi.Mode}
				}
			case FzfCancelMsg:
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenProjectList}
				}
			}
		}

		if consumed {
			return m, nil
		}
	}
	return m, nil
}

func (m modeSelectModel) view() string {
	header := titleStyle.Render("MODE") + "\n"
	return header + "\n" + m.fzfList.View()
}

func renderModeItem(item FzfItem, index int, cursor, selected bool, matched []int) string {
	mi := item.(modeItem)

	prefix := "  "
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}

	name := highlightMatches(mi.Name, matched)
	return prefix + name
}

func modePreview(item FzfItem, width, height int) string {
	mi := item.(modeItem)

	var lines []string
	lines = append(lines, titleStyle.Render(mi.Name))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(mi.Description))

	if mi.Flag != "" {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Custom system prompt"))
	}

	return strings.Join(lines, "\n")
}
