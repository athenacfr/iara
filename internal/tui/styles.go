package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	subtitleStyle = lipgloss.NewStyle().Faint(true)
	keyStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	dimStyle      = lipgloss.NewStyle().Faint(true)
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// fzf-specific styles
	fzfPromptStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	fzfSearchInput    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	fzfSearchActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	fzfMatchStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	fzfSelectedLine   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	fzfCounterStyle   = lipgloss.NewStyle().Faint(true)
	fzfBorderStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	fzfMarkerSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fzfMarkerNormal   = lipgloss.NewStyle().Faint(true)
	fzfCursorPrefix   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	fzfDividerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)

	// Tree view styles
	treeBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	treeAddStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)

	previewStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

type keyBind struct {
	key  string
	desc string
}

func renderKeybar(bindings ...keyBind) string {
	var parts []string
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.key)+" "+dimStyle.Render(b.desc))
	}
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += "  "
		}
		s += p
	}
	return s
}
