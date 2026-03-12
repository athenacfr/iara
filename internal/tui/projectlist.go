package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/project"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Tree item types

type treeProjectItem struct {
	name      string
	repoCount int
}

func (t treeProjectItem) FilterValue() string { return t.name }

type treeRepoItem struct {
	projectName string
	name        string
	branch      string
	dirtyCount  int
	isLast      bool // last repo before the + add row
}

func (t treeRepoItem) FilterValue() string { return t.name }

type treeAddItem struct {
	projectName string
}

func (t treeAddItem) FilterValue() string { return "+ add repo to " + t.projectName }

type treeNewProjectItem struct{}

func (t treeNewProjectItem) FilterValue() string { return "+ new project" }

// Messages
type projectsLoadedMsg struct {
	projects []project.Project
	err      error
}
type projectDeletedMsg struct{ name string }
type projectRenamedMsg struct{ oldName, newName string }

type projectListModel struct {
	fzfList       FzfListModel
	width, height int
	loading       bool
	err           error

	// Inline rename
	renaming    bool
	renameInput textinput.Model
	renameErr   string
	renameProj  string // project being renamed

	// Delete confirmation
	confirmDelete bool
	deleteTarget  string // repo name
	deleteProject string // which project the repo belongs to

	// Tree display — per-project expansion
	expandedProjects map[string]bool

	// All loaded projects (kept for preview)
	projects []project.Project
}

func newProjectListModel() projectListModel {
	return projectListModel{loading: true, expandedProjects: make(map[string]bool)}
}

func (m projectListModel) Init() tea.Cmd {
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

func (m *projectListModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.fzfList.SetSize(w, h-3)
}

func (m projectListModel) selectedProjectName() string {
	item := m.fzfList.SelectedItem()
	if item == nil {
		return ""
	}
	switch it := item.(type) {
	case treeProjectItem:
		return it.name
	case treeRepoItem:
		return it.projectName
	case treeAddItem:
		return it.projectName
	}
	return ""
}

func (m *projectListModel) buildTreeItems(projects []project.Project) []FzfItem {
	var items []FzfItem
	for _, p := range projects {
		full, err := project.Get(p.Name)
		if err != nil {
			full = &p
		}
		items = append(items, treeProjectItem{
			name:      full.Name,
			repoCount: len(full.Repos),
		})
		if m.expandedProjects[full.Name] {
			for i, r := range full.Repos {
				items = append(items, treeRepoItem{
					projectName: full.Name,
					name:        r.Name,
					branch:      r.Branch,
					dirtyCount:  len(r.DirtyFiles),
					isLast:      i == len(full.Repos)-1,
				})
			}
			items = append(items, treeAddItem{projectName: full.Name})
		}
	}
	items = append(items, treeNewProjectItem{})
	return items
}

func (m projectListModel) update(msg tea.Msg) (projectListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.projects = msg.projects
		items := m.buildTreeItems(msg.projects)
		m.fzfList = NewFzfList(items, FzfListConfig{
			PreviewFunc:  treePreview,
			RenderItem:   renderTreeItem,
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

	case projectRenamedMsg:
		m.renaming = false
		m.renameProj = ""
		m.renameErr = ""
		return m, loadProjects

	case tea.KeyMsg:
		// Inline rename mode
		if m.renaming {
			return m.updateRename(msg)
		}

		// Delete confirmation
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

		// Let fzf handle first
		newList, consumed, result := m.fzfList.HandleKey(msg.String())
		m.fzfList = newList

		if result != nil {
			switch result.(type) {
			case FzfConfirmMsg:
				return m.handleConfirm()
			case FzfCancelMsg:
				return m, tea.Quit
			}
		}

		if consumed {
			return m, nil
		}

		// Action keys (only when not searching)
		if !m.fzfList.IsSearching() {
			switch msg.String() {
			case "n":
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenCreateProject}
				}
			case "t", " ":
				// Toggle repos for selected project
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = !m.expandedProjects[projName]
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "right", "l":
				// Expand repos for selected project
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = true
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "left", "h":
				// Collapse repos for selected project
				if projName := m.selectedProjectName(); projName != "" {
					m.expandedProjects[projName] = false
					items := m.buildTreeItems(m.projects)
					m.fzfList.SetItems(items)
				}
				return m, nil
			case "r":
				// Rename: only works when cursor is on a project item
				if item := m.fzfList.SelectedItem(); item != nil {
					if pi, ok := item.(treeProjectItem); ok {
						m.renaming = true
						m.renameProj = pi.name
						m.renameErr = ""
						ti := textinput.New()
						ti.SetValue(pi.name)
						ti.Focus()
						ti.CharLimit = 50
						m.renameInput = ti
						return m, textinput.Blink
					}
				}
				return m, nil
			case "d":
				// Delete: project if on project row, repo if on repo row
				if item := m.fzfList.SelectedItem(); item != nil {
					switch it := item.(type) {
					case treeProjectItem:
						m.confirmDelete = true
						m.deleteTarget = it.name
						m.deleteProject = it.name
						return m, nil
					case treeRepoItem:
						m.confirmDelete = true
						m.deleteTarget = it.name
						m.deleteProject = it.projectName
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

func (m projectListModel) handleConfirm() (projectListModel, tea.Cmd) {
	item := m.fzfList.SelectedItem()
	if item == nil {
		return m, nil
	}

	switch it := item.(type) {
	case treeProjectItem:
		// Enter on project → open project (select it)
		full, err := project.Get(it.name)
		if err == nil {
			return m, func() tea.Msg {
				return projectSelectedMsg{project: full}
			}
		}
		return m, nil

	case treeRepoItem:
		// Enter on repo → open Claude at the repo directly
		return m, func() tea.Msg {
			return repoSelectedMsg{projectName: it.projectName, repoName: it.name}
		}

	case treeAddItem:
		// Enter on + add → go to edit screen add flow
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenEditProject, projectName: it.projectName}
		}

	case treeNewProjectItem:
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenCreateProject}
		}
	}
	return m, nil
}

func (m projectListModel) updateRename(msg tea.KeyMsg) (projectListModel, tea.Cmd) {
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
		newPath := filepath.Join(config.ProjectsDir(), newName)
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
			return projectRenamedMsg{oldName: oldName, newName: newName}
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

func (m projectListModel) view() string {
	if m.loading {
		return "\n  Loading projects..."
	}
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("\n  Error: %v", m.err))
	}

	header := titleStyle.Render("PROJECTS") + "\n"

	// Inline rename
	if m.renaming {
		kb := renderKeybar(keyBind{"enter", "confirm"}, keyBind{"esc", "cancel"})
		body := "\n  Rename project:\n\n  " + m.renameInput.View()
		if m.renameErr != "" {
			body += "\n\n  " + errorStyle.Render(m.renameErr)
		}
		return header + kb + "\n" + body
	}

	// Delete confirmation
	if m.confirmDelete {
		var warning string
		if m.deleteTarget == m.deleteProject {
			// Deleting a project
			warning = "\n" + errorStyle.Render("  Delete project '"+m.deleteTarget+"'?") + "\n" +
				dimStyle.Render("  This will remove the project and all cloned repos.") + "\n\n" +
				"  " + keyStyle.Render("y") + dimStyle.Render(" confirm  ") +
				keyStyle.Render("n") + dimStyle.Render(" cancel")
		} else {
			// Deleting a repo
			warning = "\n" + errorStyle.Render("  Remove '"+m.deleteTarget+"' from "+m.deleteProject+"?") + "\n" +
				dimStyle.Render("  This will delete the repo directory from this project.") + "\n\n" +
				"  " + keyStyle.Render("y") + dimStyle.Render(" confirm  ") +
				keyStyle.Render("n") + dimStyle.Render(" cancel")
		}
		return header + warning
	}

	return header + "\n" + m.fzfList.View()
}

// Render callbacks

func renderTreeItem(item FzfItem, index int, cursor, selected bool, matched []int) string {
	switch it := item.(type) {
	case treeProjectItem:
		return renderTreeProject(it, cursor, matched)
	case treeRepoItem:
		return renderTreeRepo(it, cursor, matched)
	case treeAddItem:
		return renderTreeAdd(it, cursor)
	case treeNewProjectItem:
		return renderTreeNewProject(cursor)
	}
	return ""
}

func renderTreeProject(it treeProjectItem, cursor bool, matched []int) string {
	prefix := "  "
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}

	name := highlightMatches(it.name, matched)
	repoCount := fmt.Sprintf("%d repo", it.repoCount)
	if it.repoCount != 1 {
		repoCount += "s"
	}

	line := fmt.Sprintf("%s%s  %s", prefix, accentStyle.Bold(true).Render(name), dimStyle.Render(repoCount))
	if cursor {
		hints := "  " + dimStyle.Render("[") +
			keyStyle.Render("t") + dimStyle.Render("oggle ") +
			keyStyle.Render("r") + dimStyle.Render("ename ") +
			keyStyle.Render("d") + dimStyle.Render("elete") +
			dimStyle.Render("]")
		line += hints
	}
	return line
}

func renderTreeRepo(it treeRepoItem, cursor bool, matched []int) string {
	connector := "├─"
	if it.isLast {
		connector = "├─"
	}

	var prefix string
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ") + treeBranchStyle.Render(connector) + " "
	} else {
		prefix = "  " + treeBranchStyle.Render(connector) + " "
	}

	name := highlightMatches(it.name, matched)

	// Branch + status
	meta := dimStyle.Render(" on ") + it.branch
	if it.dirtyCount > 0 {
		meta += " " + errorStyle.Render("●")
	} else {
		meta += " " + successStyle.Render("✓")
	}

	line := prefix + name + meta
	if cursor {
		hints := "  " + dimStyle.Render("[") +
			keyStyle.Render("d") + dimStyle.Render("elete") +
			dimStyle.Render("]")
		line += hints
	}
	return line
}

func renderTreeAdd(it treeAddItem, cursor bool) string {
	connector := "└─"

	var prefix string
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ") + treeBranchStyle.Render(connector) + " "
	} else {
		prefix = "  " + treeBranchStyle.Render(connector) + " "
	}

	line := prefix + treeAddStyle.Render("+ add repo")
	return line
}

func renderTreeNewProject(cursor bool) string {
	prefix := "  "
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}
	line := prefix + treeAddStyle.Render("+ new project")
	return line
}

// Preview callback — adapts based on selected item type

func treePreview(item FzfItem, width, height int) string {
	switch it := item.(type) {
	case treeProjectItem:
		return projectDetailPreview(it.name, width, height)
	case treeRepoItem:
		return repoDetailPreview(it.projectName, it.name, width, height)
	case treeAddItem:
		return addRepoPreview(it.projectName)
	case treeNewProjectItem:
		return titleStyle.Render("New Project") + "\n\n" + dimStyle.Render("Create a new project")
	}
	return ""
}

func projectDetailPreview(name string, width, height int) string {
	full, err := project.Get(name)
	if err != nil {
		return errorStyle.Render("Error loading project")
	}

	var lines []string
	lines = append(lines, titleStyle.Render(full.Name))

	if info, err := os.Stat(full.Path); err == nil {
		lines = append(lines, dimStyle.Render("opened "+timeAgo(info.ModTime())))
	}
	lines = append(lines, "")

	for _, r := range full.Repos {
		line := "  " + accentStyle.Render(r.Name) + dimStyle.Render(" on ") + r.Branch
		if len(r.DirtyFiles) > 0 {
			line += " " + errorStyle.Render("●")
		} else {
			line += " " + successStyle.Render("✓")
		}
		lines = append(lines, line)

		info := git.GetRepoInfo(full.Path, r.Name)
		if info.Ahead > 0 || info.Behind > 0 {
			ab := "    "
			if info.Ahead > 0 {
				ab += successStyle.Render(fmt.Sprintf("↑%d", info.Ahead))
			}
			if info.Behind > 0 {
				if info.Ahead > 0 {
					ab += " "
				}
				ab += errorStyle.Render(fmt.Sprintf("↓%d", info.Behind))
			}
			lines = append(lines, ab)
		}

		if len(r.DirtyFiles) > 0 {
			lines = append(lines, "    "+errorStyle.Render(fmt.Sprintf("%d modified", len(r.DirtyFiles))))
		}

		commits := git.RecentCommits(r.Path, 3)
		for _, c := range commits {
			lines = append(lines, "    "+dimStyle.Render(c))
		}
		lines = append(lines, "")
	}

	if len(full.Repos) == 0 {
		lines = append(lines, dimStyle.Render("  No repos"))
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func repoDetailPreview(projectName, repoName string, width, height int) string {
	full, err := project.Get(projectName)
	if err != nil {
		return errorStyle.Render("Error loading project")
	}

	info := git.GetRepoInfo(full.Path, repoName)

	var lines []string
	lines = append(lines, titleStyle.Render(repoName))
	lines = append(lines, dimStyle.Render("on ")+info.Branch)
	lines = append(lines, "")

	if info.Ahead > 0 || info.Behind > 0 {
		ab := ""
		if info.Ahead > 0 {
			ab += successStyle.Render(fmt.Sprintf("↑%d ahead", info.Ahead))
		}
		if info.Behind > 0 {
			if ab != "" {
				ab += "  "
			}
			ab += errorStyle.Render(fmt.Sprintf("↓%d behind", info.Behind))
		}
		lines = append(lines, ab)
		lines = append(lines, "")
	}

	if info.Clean {
		lines = append(lines, successStyle.Render("✓ clean"))
	} else {
		repoPath := filepath.Join(full.Path, repoName)
		dirtyFiles := git.DirtyFiles(repoPath)
		lines = append(lines, errorStyle.Render(fmt.Sprintf("● %d modified", len(dirtyFiles))))
	}
	lines = append(lines, "")

	if len(info.RecentCommits) > 0 {
		lines = append(lines, dimStyle.Render("Recent commits:"))
		for _, c := range info.RecentCommits {
			lines = append(lines, "  "+dimStyle.Render(c))
		}
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func addRepoPreview(projectName string) string {
	var lines []string
	lines = append(lines, titleStyle.Render("Add Repo"))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Add a new repository to"))
	lines = append(lines, accentStyle.Render(projectName))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Options:"))
	lines = append(lines, "  "+dimStyle.Render("• Clone from GitHub"))
	lines = append(lines, "  "+dimStyle.Render("• Clone from Git URL"))
	lines = append(lines, "  "+dimStyle.Render("• Link local directory"))
	lines = append(lines, "  "+dimStyle.Render("• Create empty repo"))
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
