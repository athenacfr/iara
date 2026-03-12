package tui

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// FzfItem is the interface that list items must implement.
type FzfItem interface {
	FilterValue() string
}

// FzfDivider is an optional interface. Items implementing it are rendered
// as non-selectable section headers.
type FzfDivider interface {
	IsDivider() bool
}

func isDivider(item FzfItem) bool {
	if d, ok := item.(FzfDivider); ok {
		return d.IsDivider()
	}
	return false
}

// FzfConfirmMsg is sent when the user presses enter.
type FzfConfirmMsg struct {
	Item  FzfItem
	Items []FzfItem // all selected items in multi-select mode
}

// FzfCancelMsg is sent when the user presses esc with no query.
type FzfCancelMsg struct{}

// FzfListConfig holds configuration for the list.
type FzfListConfig struct {
	MultiSelect  bool
	PreviewFunc  func(item FzfItem, width, height int) string
	RenderItem   func(item FzfItem, index int, cursor bool, selected bool, matched []int) string
	Placeholder  string
	PromptPrefix string
	ListWidthPct float64
}

// fzfSource wraps []FzfItem for sahilm/fuzzy.
type fzfSource struct {
	items []FzfItem
}

func (s fzfSource) String(i int) string { return s.items[i].FilterValue() }
func (s fzfSource) Len() int            { return len(s.items) }

// FzfListModel is the Bubble Tea model for an fzf-like list.
type FzfListModel struct {
	cfg       FzfListConfig
	items     []FzfItem
	matches   []fuzzy.Match // current filtered matches
	query     string
	searching bool // when false, printable keys are ignored
	cursor    int
	offset    int // scroll offset
	selected  map[int]bool
	width     int
	height    int
}

func NewFzfList(items []FzfItem, cfg FzfListConfig) FzfListModel {
	if cfg.PromptPrefix == "" {
		cfg.PromptPrefix = "> "
	}
	if cfg.ListWidthPct == 0 {
		cfg.ListWidthPct = 0.4
	}
	if cfg.Placeholder == "" {
		cfg.Placeholder = "No items"
	}

	m := FzfListModel{
		cfg:      cfg,
		items:    items,
		selected: make(map[int]bool),
	}
	m.applyFilter()
	m.skipDividers(1)
	return m
}

func (m *FzfListModel) SetItems(items []FzfItem) {
	m.items = items
	m.selected = make(map[int]bool)
	m.applyFilter()
}

func (m *FzfListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m FzfListModel) SelectedItem() FzfItem {
	if m.cursor >= 0 && m.cursor < len(m.matches) {
		return m.items[m.matches[m.cursor].Index]
	}
	return nil
}

func (m FzfListModel) SelectedItems() []FzfItem {
	var result []FzfItem
	for i, item := range m.items {
		if m.selected[i] {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		// If nothing explicitly selected, return cursor item
		if item := m.SelectedItem(); item != nil {
			result = append(result, item)
		}
	}
	return result
}

func (m FzfListModel) IsSearching() bool {
	return m.searching
}

func (m FzfListModel) TotalCount() int {
	count := 0
	for _, item := range m.items {
		if !isDivider(item) {
			count++
		}
	}
	return count
}

func (m *FzfListModel) applyFilter() {
	if m.query == "" {
		// Show all items in original order
		m.matches = make([]fuzzy.Match, len(m.items))
		for i := range m.items {
			m.matches[i] = fuzzy.Match{Index: i}
		}
	} else {
		m.matches = fuzzy.FindFrom(m.query, fzfSource{items: m.items})
	}
	if m.cursor >= len(m.matches) {
		m.cursor = max(0, len(m.matches)-1)
	}
	m.clampScroll()
}

func (m *FzfListModel) clampScroll() {
	visible := m.visibleHeight()
	if visible <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m FzfListModel) visibleHeight() int {
	h := m.height - 4 // search box (3 lines) + margin
	if h < 1 {
		h = 1
	}
	return h
}

// HandleKey processes a key message. Returns the updated model, whether the key
// was consumed, and an optional command (FzfConfirmMsg, FzfCancelMsg).
func (m FzfListModel) HandleKey(keyStr string) (FzfListModel, bool, interface{}) {
	// Toggle search
	if keyStr == "s" && !m.searching {
		m.searching = true
		return m, true, nil
	}

	// Search mode key handling
	if m.searching {
		switch keyStr {
		case "esc":
			m.searching = false
			m.query = ""
			m.applyFilter()
			return m, true, nil
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applyFilter()
			}
			return m, true, nil
		case "enter":
			m.searching = false
			// Fall through to confirm below
			return m.handleConfirm()
		case "up", "down":
			// Allow navigation even in search mode
			return m.handleNav(keyStr)
		default:
			// Only accept single printable chars
			if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
				m.query += keyStr
				m.applyFilter()
				return m, true, nil
			}
			return m, false, nil
		}
	}

	// Non-search mode
	switch keyStr {
	case "up", "k":
		return m.handleNav("up")
	case "down", "j":
		return m.handleNav("down")
	case "pgup":
		return m.handleNav("pgup")
	case "pgdown":
		return m.handleNav("pgdown")
	case "home", "g":
		m.cursor = 0
		m.clampScroll()
		return m, true, nil
	case "end", "G":
		m.cursor = max(0, len(m.matches)-1)
		m.clampScroll()
		return m, true, nil
	case " ":
		if m.cfg.MultiSelect {
			return m.handleNav("toggle")
		}
		return m, false, nil
	case "enter":
		return m.handleConfirm()
	case "esc":
		return m, true, FzfCancelMsg{}
	}

	return m, false, nil
}

func (m FzfListModel) handleNav(dir string) (FzfListModel, bool, interface{}) {
	switch dir {
	case "up":
		if m.cursor > 0 {
			m.cursor--
			m.skipDividers(-1)
		}
	case "down":
		if m.cursor < len(m.matches)-1 {
			m.cursor++
			m.skipDividers(1)
		}
	case "pgup":
		m.cursor -= m.visibleHeight()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.skipDividers(1)
	case "pgdown":
		m.cursor += m.visibleHeight()
		if m.cursor >= len(m.matches) {
			m.cursor = max(0, len(m.matches)-1)
		}
		m.skipDividers(-1)
	case "toggle":
		if m.cfg.MultiSelect && m.cursor < len(m.matches) {
			origIdx := m.matches[m.cursor].Index
			if !isDivider(m.items[origIdx]) {
				m.selected[origIdx] = !m.selected[origIdx]
			}
			if m.cursor < len(m.matches)-1 {
				m.cursor++
				m.skipDividers(1)
			}
		}
	}
	m.clampScroll()
	return m, true, nil
}

// skipDividers moves the cursor past any divider items in the given direction.
func (m *FzfListModel) skipDividers(dir int) {
	for m.cursor >= 0 && m.cursor < len(m.matches) {
		if !isDivider(m.items[m.matches[m.cursor].Index]) {
			return
		}
		m.cursor += dir
	}
	// If we went out of bounds, reverse
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.matches) {
		m.cursor = max(0, len(m.matches)-1)
	}
}

func (m FzfListModel) handleConfirm() (FzfListModel, bool, interface{}) {
	if len(m.matches) == 0 {
		return m, true, nil
	}
	item := m.SelectedItem()
	if item != nil && isDivider(item) {
		return m, true, nil
	}
	items := m.SelectedItems()
	return m, true, FzfConfirmMsg{Item: item, Items: items}
}

// View renders the fzf-like list.
func (m FzfListModel) View() string {
	if len(m.items) == 0 {
		prompt := m.renderPrompt(m.width)
		return prompt + "\n\n" + dimStyle.Render(m.cfg.Placeholder)
	}

	listWidth := m.width
	previewWidth := 0
	hasPreview := m.cfg.PreviewFunc != nil && m.width > 60

	if hasPreview {
		listWidth = int(float64(m.width) * m.cfg.ListWidthPct)
		previewWidth = m.width - listWidth - 3 // 3 for border + padding
	}

	// Render list items
	visible := m.visibleHeight()
	var listLines []string
	end := m.offset + visible
	if end > len(m.matches) {
		end = len(m.matches)
	}

	for i := m.offset; i < end; i++ {
		match := m.matches[i]
		isCursor := i == m.cursor
		isSelected := m.selected[match.Index]
		item := m.items[match.Index]

		var line string
		if m.cfg.RenderItem != nil {
			line = m.cfg.RenderItem(item, match.Index, isCursor, isSelected, match.MatchedIndexes)
		} else {
			line = m.defaultRenderItem(item, isCursor, isSelected, match.MatchedIndexes)
		}

		// Truncate non-cursor lines; let cursor line expand for inline hints
		if isCursor {
			// Pad to full width so background spans the entire row
			pad := listWidth - 1 - lipgloss.Width(line)
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			line = fzfSelectedLine.Render(line)
		} else if listWidth > 0 {
			line = truncate(line, listWidth-1)
		}

		listLines = append(listLines, line)
	}

	// Pad to fill visible height
	for len(listLines) < visible {
		listLines = append(listLines, "")
	}

	list := strings.Join(listLines, "\n")

	// Search input at top
	prompt := m.renderPrompt(listWidth)

	// Assemble list pane (search on top)
	listPane := prompt + "\n" + list

	if !hasPreview {
		return listPane
	}

	// Preview pane
	previewContent := ""
	if item := m.SelectedItem(); item != nil {
		previewContent = m.cfg.PreviewFunc(item, previewWidth-4, visible+2)
	}
	previewPane := previewStyle.
		Width(previewWidth).
		Height(visible + 2).
		Render(previewContent)

	// Border column
	borderLines := make([]string, visible+2)
	for i := range borderLines {
		borderLines[i] = fzfBorderStyle.Render("│")
	}
	border := strings.Join(borderLines, "\n")

	leftPane := lipgloss.NewStyle().Width(listWidth).Render(listPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, border, previewPane)
}

func (m FzfListModel) renderPrompt(width int) string {
	if width < 10 {
		width = 30
	}
	innerWidth := width - 4 // 2 for border + 2 for padding

	// Build inner content
	var content string
	if m.searching {
		content = fzfSearchInput.Render(m.query) + "█"
	} else if m.query != "" {
		content = dimStyle.Render(m.query)
	} else {
		content = keyStyle.Render("s") + dimStyle.Render("earch...")
	}

	// Pad content to fill inner width
	pad := innerWidth - lipgloss.Width(content)
	if pad > 0 {
		content += strings.Repeat(" ", pad)
	}

	// Draw box with gray borders
	border := fzfBorderStyle
	top := border.Render("┌" + strings.Repeat("─", innerWidth+2) + "┐")
	mid := border.Render("│") + " " + content + " " + border.Render("│")
	bot := border.Render("└" + strings.Repeat("─", innerWidth+2) + "┘")

	return top + "\n" + mid + "\n" + bot
}

func (m FzfListModel) defaultRenderItem(item FzfItem, isCursor, isSelected bool, matched []int) string {
	if isDivider(item) {
		return fzfDividerStyle.Render("── " + item.FilterValue() + " ──")
	}

	text := item.FilterValue()

	// Build prefix
	prefix := "  "
	if isCursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}
	if m.cfg.MultiSelect {
		if isSelected {
			prefix += fzfMarkerSelected.Render("● ")
		} else {
			prefix += fzfMarkerNormal.Render("○ ")
		}
	}

	// Highlight matched characters
	highlighted := highlightMatches(text, matched)

	line := prefix + highlighted
	return line
}

func highlightMatches(text string, matched []int) string {
	if len(matched) == 0 {
		return text
	}
	matchSet := make(map[int]bool, len(matched))
	for _, idx := range matched {
		matchSet[idx] = true
	}

	var result strings.Builder
	for i, ch := range text {
		if matchSet[i] {
			result.WriteString(fzfMatchStyle.Render(string(ch)))
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 || lipgloss.Width(s) <= maxWidth {
		return s
	}
	return xansi.Truncate(s, maxWidth, "")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
