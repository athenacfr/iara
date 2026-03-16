package screen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/paths"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/ahtwr/cw/internal/tui/widget"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Tree item types

type TreeProjectItem struct {
	Name      string
	RepoCount int
}

func (t TreeProjectItem) FilterValue() string { return t.Name }

type TreeRepoItem struct {
	ProjectName string
	Name        string
	Branch      string
	DirtyCount  int
	IsLast      bool
}

func (t TreeRepoItem) FilterValue() string { return t.Name }

type TreeAddItem struct {
	ProjectName string
}

func (t TreeAddItem) FilterValue() string { return "+ add repo to " + t.ProjectName }

type TreeNewProjectItem struct{}

func (t TreeNewProjectItem) FilterValue() string { return "+ new project" }

// Messages
type projectsLoadedMsg struct {
	projects []project.Project
	err      error
}
type projectDeletedMsg struct{ name string }
type projectRenamedMsg2 struct{ oldName, newName string }

type ProjectExplorerModel struct {
	fzfList       widget.FzfListModel
	width, height int
	loading       bool
	err           error

	renaming    bool
	renameInput textinput.Model
	renameErr   string
	renameProj  string

	confirmDelete bool
	deleteTarget  string
	deleteProject string

	expandedProjects map[string]bool

	BypassPerms      bool
	AutoCompactLimit int

	projects []project.Project
}

func NewProjectExplorerModel() ProjectExplorerModel {
	s := project.LoadGlobalSettings()
	return ProjectExplorerModel{
		loading:          true,
		expandedProjects: make(map[string]bool),
		BypassPerms:      s.BypassPermissions,
		AutoCompactLimit: s.AutoCompactLimit,
	}
}

func (m ProjectExplorerModel) Init() tea.Cmd {
	return loadProjects
}

func loadProjects() tea.Msg {
	projects, err := project.List()
	return projectsLoadedMsg{projects: projects, err: err}
}

func deleteProject(name string) tea.Cmd {
	return func() tea.Msg {
		project.Delete(name)
		return projectDeletedMsg{name: name}
	}
}

func removeRepoFromProject(projectName, repoName string) tea.Cmd {
	return func() tea.Msg {
		project.RemoveRepo(projectName, repoName)
		return projectDeletedMsg{name: repoName}
	}
}

func (m *ProjectExplorerModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.fzfList.SetSize(w, h-3)
}

func (m ProjectExplorerModel) selectedProjectName() string {
	item := m.fzfList.SelectedItem()
	if item == nil {
		return ""
	}
	switch it := item.(type) {
	case TreeProjectItem:
		return it.Name
	case TreeRepoItem:
		return it.ProjectName
	case TreeAddItem:
		return it.ProjectName
	}
	return ""
}

func (m *ProjectExplorerModel) buildTreeItems(projects []project.Project) []widget.FzfItem {
	var items []widget.FzfItem
	for _, p := range projects {
		full, err := project.Get(p.Name)
		if err != nil {
			full = &p
		}
		items = append(items, TreeProjectItem{
			Name:      full.Name,
			RepoCount: len(full.Repos),
		})
		if m.expandedProjects[full.Name] {
			for i, r := range full.Repos {
				items = append(items, TreeRepoItem{
					ProjectName: full.Name,
					Name:        r.Name,
					Branch:      r.Branch,
					DirtyCount:  len(r.DirtyFiles),
					IsLast:      i == len(full.Repos)-1,
				})
			}
			items = append(items, TreeAddItem{ProjectName: full.Name})
		}
	}
	items = append(items, TreeNewProjectItem{})
	return items
}

func (m ProjectExplorerModel) Update(msg tea.Msg) (ProjectExplorerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.projects = msg.projects
		items := m.buildTreeItems(msg.projects)
		m.fzfList = widget.NewFzfList(items, widget.FzfListConfig{
			PreviewFunc:  treePreview,
			RenderItem:   RenderTreeItem,
			Placeholder:  "No projects. Press 'n' to create one.",
			ListWidthPct: 0.4,
		})
		m.fzfList.SetSize(m.width, m.height-4)
		return m, nil

	case projectDeletedMsg:
		m.confirmDelete = false
		m.deleteTarget = ""
		m.deleteProject = ""
		return m, loadProjects

	case projectRenamedMsg2:
		m.renaming = false
		m.renameProj = ""
		m.renameErr = ""
		return m, loadProjects

	case tea.KeyMsg:
		if m.renaming {
			return m.updateRename(msg)
		}

		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y", "enter":
				proj := m.deleteProject
				target := m.deleteTarget
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteProject = ""
				if target == proj {
					return m, deleteProject(proj)
				}
				return m, removeRepoFromProject(proj, target)
			case "n", "N", "esc":
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteProject = ""
			default:
				return m, nil
			}
			return m, nil
		}

		newList, consumed, result := m.fzfList.HandleKey(msg.String())
		m.fzfList = newList

		if result != nil {
			switch result.(type) {
			case widget.FzfConfirmMsg:
				return m.handleConfirm()
			case widget.FzfCancelMsg:
				return m, tea.Quit
			}
		}

		if consumed {
			return m, nil
		}

		if !m.fzfList.IsSearching() {
			switch msg.String() {
			case "n":
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenProjectWizard}
				}
			case "t", " ":
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = !m.expandedProjects[projName]
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "right", "l":
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = true
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "left", "h":
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = false
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "r":
				if item := m.fzfList.SelectedItem(); item != nil {
					if pi, ok := item.(TreeProjectItem); ok {
						m.renaming = true
						m.renameProj = pi.Name
						m.renameErr = ""
						ti := textinput.New()
						ti.SetValue(pi.Name)
						ti.Focus()
						ti.CharLimit = 50
						m.renameInput = ti
						return m, textinput.Blink
					}
				}
				return m, nil
			case "p":
				m.BypassPerms = !m.BypassPerms
				project.SaveGlobalSettings(project.Settings{
					BypassPermissions: m.BypassPerms,
					AutoCompactLimit:  m.AutoCompactLimit,
				})
				return m, nil
			case "c":
				switch m.AutoCompactLimit {
				case 0:
					m.AutoCompactLimit = 40
				case 40:
					m.AutoCompactLimit = 50
				case 50:
					m.AutoCompactLimit = 60
				case 60:
					m.AutoCompactLimit = 70
				case 70:
					m.AutoCompactLimit = 80
				default:
					m.AutoCompactLimit = 0
				}
				project.SaveGlobalSettings(project.Settings{
					BypassPermissions: m.BypassPerms,
					AutoCompactLimit:  m.AutoCompactLimit,
				})
				return m, nil
			case ",": // settings
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenSettings}
				}
			case "d":
				if item := m.fzfList.SelectedItem(); item != nil {
					switch it := item.(type) {
					case TreeProjectItem:
						m.confirmDelete = true
						m.deleteTarget = it.Name
						m.deleteProject = it.Name
						return m, nil
					case TreeRepoItem:
						m.confirmDelete = true
						m.deleteTarget = it.Name
						m.deleteProject = it.ProjectName
						return m, nil
					}
				}
				return m, nil
			case "q":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m ProjectExplorerModel) handleConfirm() (ProjectExplorerModel, tea.Cmd) {
	item := m.fzfList.SelectedItem()
	if item == nil {
		return m, nil
	}

	switch it := item.(type) {
	case TreeProjectItem:
		full, err := project.Get(it.Name)
		if err == nil {
			return m, func() tea.Msg {
				return shared.ProjectSelectedMsg{Project: full}
			}
		}
		return m, nil

	case TreeRepoItem:
		return m, func() tea.Msg {
			return shared.RepoSelectedMsg{ProjectName: it.ProjectName, RepoName: it.Name}
		}

	case TreeAddItem:
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenAddRepo, ProjectName: it.ProjectName}
		}

	case TreeNewProjectItem:
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenProjectWizard}
		}
	}
	return m, nil
}

func (m ProjectExplorerModel) updateRename(msg tea.KeyMsg) (ProjectExplorerModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := strings.TrimSpace(m.renameInput.Value())
		if newName == "" {
			m.renameErr = "Name cannot be empty"
			return m, nil
		}
		if newName == m.renameProj {
			m.renaming = false
			m.renameProj = ""
			return m, nil
		}
		newPath := filepath.Join(paths.ProjectsDir(), newName)
		if _, err := os.Stat(newPath); err == nil {
			m.renameErr = "A project with that name already exists"
			return m, nil
		}
		oldName := m.renameProj
		if err := project.Rename(oldName, newName); err != nil {
			m.renameErr = "Rename failed: " + err.Error()
			return m, nil
		}
		return m, func() tea.Msg {
			return projectRenamedMsg2{oldName: oldName, newName: newName}
		}
	case "esc":
		m.renaming = false
		m.renameProj = ""
		m.renameErr = ""
		return m, nil
	default:
		m.renameErr = ""
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

func (m ProjectExplorerModel) View() string {
	if m.loading {
		return "\n  Loading projects..."
	}
	if m.err != nil {
		return style.ErrorStyle.Render(fmt.Sprintf("\n  Error: %v", m.err))
	}

	header := style.TitleStyle.Render("PROJECTS") + "\n"

	if m.renaming {
		kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "confirm"}, style.KeyBind{Key: "esc", Desc: "cancel"})
		body := "\n  Rename project:\n\n  " + m.renameInput.View()
		if m.renameErr != "" {
			body += "\n\n  " + style.ErrorStyle.Render(m.renameErr)
		}
		return header + kb + "\n" + body
	}

	if m.confirmDelete {
		var warning string
		if m.deleteTarget == m.deleteProject {
			warning = "\n" + style.ErrorStyle.Render("  Delete project '"+m.deleteTarget+"'?") + "\n" +
				style.DimStyle.Render("  This will remove the project and all cloned repos.") + "\n\n" +
				"  " + style.KeyStyle.Render("y") + style.DimStyle.Render(" confirm  ") +
				style.KeyStyle.Render("n") + style.DimStyle.Render(" cancel")
		} else {
			warning = "\n" + style.ErrorStyle.Render("  Remove '"+m.deleteTarget+"' from "+m.deleteProject+"?") + "\n" +
				style.DimStyle.Render("  This will delete the repo directory from this project.") + "\n\n" +
				"  " + style.KeyStyle.Render("y") + style.DimStyle.Render(" confirm  ") +
				style.KeyStyle.Render("n") + style.DimStyle.Render(" cancel")
		}
		return header + warning
	}

	var permLabel string
	if m.BypassPerms {
		permLabel = style.ErrorStyle.Render("bypass")
	} else {
		permLabel = style.SuccessStyle.Render("normal")
	}
	keybar := style.KeyStyle.Render("p") + style.DimStyle.Render("ermissions") + "  " + permLabel

	var compactLabel string
	if m.AutoCompactLimit == 0 {
		compactLabel = style.DimStyle.Render("off")
	} else {
		compactLabel = style.AccentStyle.Render(fmt.Sprintf("%d%%", m.AutoCompactLimit))
	}
	keybar += "    " + style.KeyStyle.Render("c") + style.DimStyle.Render("ompact") + "  " + compactLabel
	keybar += "    " + style.KeyStyle.Render(",") + style.DimStyle.Render(" settings")

	return header + "\n" + m.fzfList.View() + "\n\n" + keybar
}

// Render callbacks

func RenderTreeItem(item widget.FzfItem, displayNum int, cursor, selected bool, matched []int, width int) string {
	switch it := item.(type) {
	case TreeProjectItem:
		return renderTreeProject(it, displayNum, cursor, matched)
	case TreeRepoItem:
		return renderTreeRepo(it, displayNum, cursor, matched)
	case TreeAddItem:
		return renderTreeAdd(it, displayNum, cursor)
	case TreeNewProjectItem:
		return renderTreeNewProject(displayNum, cursor)
	}
	return ""
}

func renderTreeProject(it TreeProjectItem, displayNum int, cursor bool, matched []int) string {
	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "
	name := widget.HighlightMatches(it.Name, matched)
	repoCount := fmt.Sprintf("%d repo", it.RepoCount)
	if it.RepoCount != 1 {
		repoCount += "s"
	}

	line := fmt.Sprintf("%s%s%s  %s", prefix, numStr, style.AccentStyle.Bold(true).Render(name), style.DimStyle.Render(repoCount))
	if cursor {
		hints := "  " + style.DimStyle.Render("[") +
			style.KeyStyle.Render("t") + style.DimStyle.Render("oggle ") +
			style.KeyStyle.Render("r") + style.DimStyle.Render("ename ") +
			style.KeyStyle.Render("d") + style.DimStyle.Render("elete") +
			style.DimStyle.Render("]")
		line += hints
	}
	return line
}

func renderTreeRepo(it TreeRepoItem, displayNum int, cursor bool, matched []int) string {
	connector := "├─"
	if it.IsLast {
		connector = "├─"
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "

	var prefix string
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ") + numStr + style.TreeBranchStyle.Render(connector) + " "
	} else {
		prefix = "  " + numStr + style.TreeBranchStyle.Render(connector) + " "
	}

	name := widget.HighlightMatches(it.Name, matched)

	meta := style.DimStyle.Render(" on ") + it.Branch
	if it.DirtyCount > 0 {
		meta += " " + style.ErrorStyle.Render("●")
	} else {
		meta += " " + style.SuccessStyle.Render("✓")
	}

	line := prefix + name + meta
	if cursor {
		hints := "  " + style.DimStyle.Render("[") +
			style.KeyStyle.Render("d") + style.DimStyle.Render("elete") +
			style.DimStyle.Render("]")
		line += hints
	}
	return line
}

func renderTreeAdd(it TreeAddItem, displayNum int, cursor bool) string {
	connector := "└─"

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "

	var prefix string
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ") + numStr + style.TreeBranchStyle.Render(connector) + " "
	} else {
		prefix = "  " + numStr + style.TreeBranchStyle.Render(connector) + " "
	}

	return prefix + style.TreeAddStyle.Render("+ add repo")
}

func renderTreeNewProject(displayNum int, cursor bool) string {
	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}
	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "
	return prefix + numStr + style.TreeAddStyle.Render("+ new project")
}

// Preview callback

func treePreview(item widget.FzfItem, width, height int) string {
	switch it := item.(type) {
	case TreeProjectItem:
		return projectDetailPreview(it.Name, width, height)
	case TreeRepoItem:
		return repoDetailPreview(it.ProjectName, it.Name, width, height)
	case TreeAddItem:
		return addRepoPreview(it.ProjectName)
	case TreeNewProjectItem:
		return style.TitleStyle.Render("New Project") + "\n\n" + style.DimStyle.Render("Create a new project")
	}
	return ""
}

func projectDetailPreview(name string, width, height int) string {
	full, err := project.Get(name)
	if err != nil {
		return style.ErrorStyle.Render("Error loading project")
	}

	var lines []string
	lines = append(lines, style.TitleStyle.Render(full.Name))

	if info, err := os.Stat(full.Path); err == nil {
		lines = append(lines, style.DimStyle.Render("opened "+timeAgo(info.ModTime())))
	}
	lines = append(lines, "")

	if meta, err := project.LoadMetadata(name); err == nil {
		if meta.Title != "" {
			lines = append(lines, style.AccentStyle.Render(meta.Title))
			lines = append(lines, "")
		}
		if meta.Description != "" {
			for _, dl := range strings.Split(meta.Description, "\n") {
				lines = append(lines, style.DimStyle.Render(dl))
			}
			lines = append(lines, "")
		}
	}

	for _, r := range full.Repos {
		line := "  " + style.AccentStyle.Render(r.Name) + style.DimStyle.Render(" on ") + r.Branch
		if len(r.DirtyFiles) > 0 {
			line += " " + style.ErrorStyle.Render("●")
		} else {
			line += " " + style.SuccessStyle.Render("✓")
		}
		lines = append(lines, line)

		info := git.GetRepoInfo(full.Path, r.Name)
		if info.Ahead > 0 || info.Behind > 0 {
			ab := "    "
			if info.Ahead > 0 {
				ab += style.SuccessStyle.Render(fmt.Sprintf("↑%d", info.Ahead))
			}
			if info.Behind > 0 {
				if info.Ahead > 0 {
					ab += " "
				}
				ab += style.ErrorStyle.Render(fmt.Sprintf("↓%d", info.Behind))
			}
			lines = append(lines, ab)
		}

		if len(r.DirtyFiles) > 0 {
			lines = append(lines, "    "+style.ErrorStyle.Render(fmt.Sprintf("%d modified", len(r.DirtyFiles))))
		}

		commits := git.RecentCommits(r.Path, 3)
		for _, commit := range commits {
			lines = append(lines, "    "+style.DimStyle.Render(commit))
		}
		lines = append(lines, "")
	}

	if len(full.Repos) == 0 {
		lines = append(lines, style.DimStyle.Render("  No repos"))
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func repoDetailPreview(projectName, repoName string, width, height int) string {
	full, err := project.Get(projectName)
	if err != nil {
		return style.ErrorStyle.Render("Error loading project")
	}

	info := git.GetRepoInfo(full.Path, repoName)

	var lines []string
	lines = append(lines, style.TitleStyle.Render(repoName))
	lines = append(lines, style.DimStyle.Render("on ")+info.Branch)
	lines = append(lines, "")

	if info.Ahead > 0 || info.Behind > 0 {
		ab := ""
		if info.Ahead > 0 {
			ab += style.SuccessStyle.Render(fmt.Sprintf("↑%d ahead", info.Ahead))
		}
		if info.Behind > 0 {
			if ab != "" {
				ab += "  "
			}
			ab += style.ErrorStyle.Render(fmt.Sprintf("↓%d behind", info.Behind))
		}
		lines = append(lines, ab)
		lines = append(lines, "")
	}

	if info.Clean {
		lines = append(lines, style.SuccessStyle.Render("✓ clean"))
	} else {
		repoPath := filepath.Join(full.Path, repoName)
		dirtyFiles := git.DirtyFiles(repoPath)
		lines = append(lines, style.ErrorStyle.Render(fmt.Sprintf("● %d modified", len(dirtyFiles))))
	}
	lines = append(lines, "")

	if len(info.RecentCommits) > 0 {
		lines = append(lines, style.DimStyle.Render("Recent commits:"))
		for _, commit := range info.RecentCommits {
			lines = append(lines, "  "+style.DimStyle.Render(commit))
		}
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func addRepoPreview(projectName string) string {
	var lines []string
	lines = append(lines, style.TitleStyle.Render("Add Repo"))
	lines = append(lines, "")
	lines = append(lines, style.DimStyle.Render("Add a new repository to"))
	lines = append(lines, style.AccentStyle.Render(projectName))
	lines = append(lines, "")
	lines = append(lines, style.DimStyle.Render("Options:"))
	lines = append(lines, "  "+style.DimStyle.Render("• Clone from GitHub"))
	lines = append(lines, "  "+style.DimStyle.Render("• Clone from Git URL"))
	lines = append(lines, "  "+style.DimStyle.Render("• Link local directory"))
	lines = append(lines, "  "+style.DimStyle.Render("• Create empty repo"))
	return strings.Join(lines, "\n")
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
