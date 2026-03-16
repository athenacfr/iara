package widget

import (
	"fmt"
	"strings"

	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
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

// IsDivider returns true if the item implements FzfDivider and is a divider.
func IsDivider(item FzfItem) bool {
	if d, ok := item.(FzfDivider); ok {
		return d.IsDivider()
	}
	return false
}

// FzfConfirmMsg is sent when the user presses enter.
type FzfConfirmMsg struct {
	Item  FzfItem
	Items []FzfItem
}

// FzfCancelMsg is sent when the user presses esc with no query.
type FzfCancelMsg struct{}

// FzfListConfig holds configuration for the list.
type FzfListConfig struct {
	MultiSelect  bool
	PreviewFunc  func(item FzfItem, width, height int) string
	RenderItem   func(item FzfItem, index int, cursor bool, selected bool, matched []int, width int) string
	Placeholder  string
	PromptPrefix string
	ListWidthPct float64
	MaxLines     int // max lines per item (0 or 1 = single line, 2+ = multi-line wrap)
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
	matches   []fuzzy.Match
	query     string
	searching bool
	cursor    int
	offset    int
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
		if !IsDivider(item) {
			count++
		}
	}
	return count
}

func (m *FzfListModel) applyFilter() {
	if m.query == "" {
		m.matches = make([]fuzzy.Match, len(m.items))
		for i := range m.items {
			m.matches[i] = fuzzy.Match{Index: i}
		}
	} else {
		m.matches = fuzzy.FindFrom(m.query, fzfSource{items: m.items})
	}
	if m.cursor >= len(m.matches) {
		m.cursor = Max(0, len(m.matches)-1)
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
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

// HandleKey processes a key message. Returns the updated model, whether the key
// was consumed, and an optional command (FzfConfirmMsg, FzfCancelMsg).
func (m FzfListModel) HandleKey(keyStr string) (FzfListModel, bool, interface{}) {
	if keyStr == "s" && !m.searching {
		m.searching = true
		return m, true, nil
	}

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
			return m.handleConfirm()
		case "up", "down":
			return m.handleNav(keyStr)
		default:
			if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
				m.query += keyStr
				m.applyFilter()
				return m, true, nil
			}
			return m, false, nil
		}
	}

	// Number keys 1-9 jump to the Nth non-divider item
	if len(keyStr) == 1 && keyStr[0] >= '1' && keyStr[0] <= '9' {
		targetNum := int(keyStr[0] - '0')
		count := 0
		for i, match := range m.matches {
			if !IsDivider(m.items[match.Index]) {
				count++
				if count == targetNum {
					m.cursor = i
					m.clampScroll()
					return m, true, nil
				}
			}
		}
		return m, true, nil
	}

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
		m.cursor = Max(0, len(m.matches)-1)
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
			m.cursor = Max(0, len(m.matches)-1)
		}
		m.skipDividers(-1)
	case "toggle":
		if m.cfg.MultiSelect && m.cursor < len(m.matches) {
			origIdx := m.matches[m.cursor].Index
			if !IsDivider(m.items[origIdx]) {
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

func (m *FzfListModel) skipDividers(dir int) {
	for m.cursor >= 0 && m.cursor < len(m.matches) {
		if !IsDivider(m.items[m.matches[m.cursor].Index]) {
			return
		}
		m.cursor += dir
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.matches) {
		m.cursor = Max(0, len(m.matches)-1)
	}
}

func (m FzfListModel) handleConfirm() (FzfListModel, bool, interface{}) {
	if len(m.matches) == 0 {
		return m, true, nil
	}
	item := m.SelectedItem()
	if item != nil && IsDivider(item) {
		return m, true, nil
	}
	items := m.SelectedItems()
	return m, true, FzfConfirmMsg{Item: item, Items: items}
}

// View renders the fzf-like list.
func (m FzfListModel) View() string {
	if len(m.items) == 0 {
		prompt := m.renderPrompt(m.width)
		return prompt + "\n\n" + style.DimStyle.Render(m.cfg.Placeholder)
	}

	listWidth := m.width
	previewWidth := 0
	hasPreview := m.cfg.PreviewFunc != nil && m.width > 60

	if hasPreview {
		listWidth = int(float64(m.width) * m.cfg.ListWidthPct)
		previewWidth = m.width - listWidth - 3
	}

	visible := m.visibleHeight()
	var listLines []string
	totalLines := 0

	// Count non-dividers before offset for correct numbering
	displayNum := 0
	for i := 0; i < m.offset && i < len(m.matches); i++ {
		if !IsDivider(m.items[m.matches[i].Index]) {
			displayNum++
		}
	}

	for i := m.offset; i < len(m.matches); i++ {
		match := m.matches[i]
		isCursor := i == m.cursor
		isSelected := m.selected[match.Index]
		item := m.items[match.Index]

		isDividerItem := IsDivider(item)
		if !isDividerItem {
			displayNum++
		}
		itemDisplayNum := displayNum
		if isDividerItem {
			itemDisplayNum = 0
		}

		var rendered string
		if m.cfg.RenderItem != nil {
			rendered = m.cfg.RenderItem(item, itemDisplayNum, isCursor, isSelected, match.MatchedIndexes, listWidth-1)
		} else {
			rendered = m.defaultRenderItem(item, itemDisplayNum, isCursor, isSelected, match.MatchedIndexes)
			if listWidth > 0 {
				rendered = Truncate(rendered, listWidth-1)
			}
		}

		itemLines := strings.Count(rendered, "\n") + 1
		if totalLines+itemLines > visible {
			break
		}

		if isCursor && m.cfg.RenderItem == nil {
			pad := listWidth - 1 - lipgloss.Width(rendered)
			if pad > 0 {
				rendered += strings.Repeat(" ", pad)
			}
			rendered = style.FzfSelectedLine.Render(rendered)
		}

		listLines = append(listLines, rendered)
		totalLines += itemLines
	}

	for totalLines < visible {
		listLines = append(listLines, "")
		totalLines++
	}

	list := strings.Join(listLines, "\n")

	prompt := m.renderPrompt(listWidth)
	listPane := prompt + "\n" + list

	if !hasPreview {
		return listPane
	}

	previewContent := ""
	if item := m.SelectedItem(); item != nil {
		previewContent = m.cfg.PreviewFunc(item, previewWidth-4, visible+2)
	}
	previewPane := style.PreviewStyle.
		Width(previewWidth).
		Height(visible + 2).
		Render(previewContent)

	borderLines := make([]string, visible+2)
	for i := range borderLines {
		borderLines[i] = style.FzfBorderStyle.Render("│")
	}
	border := strings.Join(borderLines, "\n")

	leftPane := lipgloss.NewStyle().Width(listWidth).Render(listPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, border, previewPane)
}

func (m FzfListModel) renderPrompt(width int) string {
	if width < 10 {
		width = 30
	}
	innerWidth := width - 4

	var content string
	if m.searching {
		content = style.FzfSearchInput.Render(m.query) + "█"
	} else if m.query != "" {
		content = style.DimStyle.Render(m.query)
	} else {
		content = style.KeyStyle.Render("s") + style.DimStyle.Render("earch...")
	}

	pad := innerWidth - lipgloss.Width(content)
	if pad > 0 {
		content += strings.Repeat(" ", pad)
	}

	border := style.FzfBorderStyle
	top := border.Render("┌" + strings.Repeat("─", innerWidth+2) + "┐")
	mid := border.Render("│") + " " + content + " " + border.Render("│")
	bot := border.Render("└" + strings.Repeat("─", innerWidth+2) + "┘")

	return top + "\n" + mid + "\n" + bot
}

func (m FzfListModel) defaultRenderItem(item FzfItem, displayNum int, isCursor, isSelected bool, matched []int) string {
	if IsDivider(item) {
		return style.FzfDividerStyle.Render("── " + item.FilterValue() + " ──")
	}

	text := item.FilterValue()

	prefix := "  "
	if isCursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}
	if m.cfg.MultiSelect {
		if isSelected {
			prefix += style.FzfMarkerSelected.Render("● ")
		} else {
			prefix += style.FzfMarkerNormal.Render("○ ")
		}
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "
	highlighted := HighlightMatches(text, matched)

	return prefix + numStr + highlighted
}

// HighlightMatches highlights matched character positions in the text.
func HighlightMatches(text string, matched []int) string {
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
			result.WriteString(style.FzfMatchStyle.Render(string(ch)))
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// Truncate truncates a string to maxWidth, appending "..." if needed.
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 || lipgloss.Width(s) <= maxWidth {
		return s
	}
	return xansi.Truncate(s, maxWidth, "...")
}

// Max returns the larger of a or b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
