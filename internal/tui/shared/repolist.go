package shared

import (
	"fmt"
	"strings"

	"github.com/ahtwr/iara/internal/gh"
	"github.com/ahtwr/iara/internal/git"
	"github.com/ahtwr/iara/internal/tui/style"
	"github.com/ahtwr/iara/internal/tui/widget"
	tea "github.com/charmbracelet/bubbletea"
)

// MethodItem represents a method option in the method selection list.
type MethodItem struct{ Name string }

func (m MethodItem) FilterValue() string { return m.Name }

// OrgDividerItem is a non-selectable divider for org groups in repo lists.
type OrgDividerItem struct{ Name string }

func (o OrgDividerItem) FilterValue() string { return o.Name }
func (o OrgDividerItem) IsDivider() bool     { return true }

// RepoItem represents a GitHub repo in the fzf list.
type RepoItem struct {
	gh.Repo
	Org string
}

func (r RepoItem) FilterValue() string {
	return r.NameWithOwner + " " + r.Description + " " + r.PrimaryLanguage.Name
}

// AllReposLoadedMsg is sent when all GitHub repos have been fetched.
type AllReposLoadedMsg struct {
	Repos []widget.FzfItem
	Err   error
}

// AllClonesCompleteMsg is sent when all parallel clones finish.
type AllClonesCompleteMsg struct{ ProjectName string }

// AddCompleteMsg is sent when a single add operation completes.
type AddCompleteMsg struct{}

// LoadAllRepos fetches all repos from all GitHub orgs/users.
func LoadAllRepos() tea.Msg {
	groups, err := gh.FetchRepos()
	if err != nil {
		return AllReposLoadedMsg{Err: err}
	}

	var items []widget.FzfItem
	for _, g := range groups {
		items = append(items, OrgDividerItem{Name: g.Owner})
		for _, r := range g.Repos {
			items = append(items, RepoItem{Repo: r, Org: g.Owner})
		}
	}
	return AllReposLoadedMsg{Repos: items}
}

// ListenForCloneProgress returns a tea.Cmd that reads one clone progress event.
func ListenForCloneProgress(ch <-chan git.CloneProgress) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-ch
		if !ok {
			return widget.CloneAllDoneMsg{}
		}
		return widget.CloneTickMsg(progress)
	}
}

// RenderRepoItem renders a GitHub repo item in an fzf list.
func RenderRepoItem(item widget.FzfItem, displayNum int, cursor, selected bool, matched []int, width int) string {
	if widget.IsDivider(item) {
		d := item.(OrgDividerItem)
		return style.FzfDividerStyle.Render("  ── " + d.Name + " ──")
	}

	ri := item.(RepoItem)

	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "

	marker := style.FzfMarkerNormal.Render("○ ")
	if selected {
		marker = style.FzfMarkerSelected.Render("● ")
	}

	name := widget.HighlightMatches(ri.Name, matched)
	lang := ""
	if ri.PrimaryLanguage.Name != "" {
		lang = " " + style.DimStyle.Render("["+ri.PrimaryLanguage.Name+"]")
	}

	return prefix + numStr + marker + name + lang
}

// RepoPreview renders a preview pane for a GitHub repo item.
func RepoPreview(item widget.FzfItem, width, height int) string {
	ri, ok := item.(RepoItem)
	if !ok {
		return ""
	}
	var lines []string
	title := ri.NameWithOwner
	if title == "" {
		title = ri.Name
	}
	lines = append(lines, style.TitleStyle.Render(title))
	lines = append(lines, "")
	if ri.Description != "" {
		lines = append(lines, ri.Description)
		lines = append(lines, "")
	}
	if ri.PrimaryLanguage.Name != "" {
		lines = append(lines, style.DimStyle.Render("Language: ")+ri.PrimaryLanguage.Name)
	}
	if ri.StargazerCount > 0 {
		lines = append(lines, style.DimStyle.Render("Stars: ")+fmt.Sprintf("%d", ri.StargazerCount))
	}
	if ri.UpdatedAt != "" && len(ri.UpdatedAt) >= 10 {
		lines = append(lines, style.DimStyle.Render("Updated: ")+ri.UpdatedAt[:10])
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
