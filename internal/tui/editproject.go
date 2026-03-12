package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/gh"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/paths"
	"github.com/ahtwr/cw/internal/project"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type editStep int

const (
	editStepMain editStep = iota
	editStepRename
	editStepMethod
	editStepRepos
	editStepGitURL
	editStepLocalPath
	editStepCopyMove
	editStepCloning
	editStepConfirmRemove
)

// editRepoItem represents a repo in the edit list.
type editRepoItem struct {
	name       string
	branch     string
	dirtyCount int
}

func (e editRepoItem) FilterValue() string { return e.name }

// repoRemovedMsg is sent after a repo is removed.
type repoRemovedMsg struct{}

type editProjectModel struct {
	step   editStep
	width  int
	height int

	projectName string
	projectPath string
	repoList    FzfListModel

	// Rename
	renameInput textinput.Model
	renameErr   string

	// Remove confirmation
	removeTarget string

	// Add repo (reuses create flow)
	methodList   FzfListModel
	ghRepoList   FzfListModel
	urlInput     textinput.Model
	pathInput    textinput.Model
	copyMoveList FzfListModel
	localPath    string
	spinner      spinner.Model
	statusText   string
	progress     progressModel
	cloneChan    <-chan git.CloneProgress
	loading      bool
	ghAvail      bool
	reposErr     error

	// Inline error
	errMsg string
}

func newEditProjectModel(name string) editProjectModel {
	urlTi := textinput.New()
	urlTi.Placeholder = "https://github.com/user/repo.git"
	urlTi.CharLimit = 200

	pathTi := textinput.New()
	pathTi.Placeholder = "/path/to/directory"
	pathTi.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	projPath := filepath.Join(paths.ProjectsDir(), name)

	m := editProjectModel{
		step:        editStepMain,
		projectName: name,
		projectPath: projPath,
		urlInput:    urlTi,
		pathInput:   pathTi,
		spinner:     sp,
		ghAvail:     gh.IsAvailable() && gh.IsAuthenticated(),
	}
	m.refreshRepoList()
	return m
}

func (m *editProjectModel) refreshRepoList() {
	p, err := project.Get(m.projectName)
	if err != nil {
		return
	}
	m.projectPath = p.Path
	items := make([]FzfItem, len(p.Repos))
	for i, r := range p.Repos {
		items[i] = editRepoItem{
			name:       r.Name,
			branch:     r.Branch,
			dirtyCount: len(r.DirtyFiles),
		}
	}
	m.repoList = NewFzfList(items, FzfListConfig{
		RenderItem:   renderEditRepoItem,
		PreviewFunc:  editRepoPreview(m.projectPath),
		Placeholder:  "No repos. Press 'a' to add one.",
		ListWidthPct: 0.4,
	})
	m.repoList.SetSize(m.width, m.height-5)
}

func (m editProjectModel) Init() tea.Cmd {
	return nil
}

func (m *editProjectModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.repoList.SetSize(w, h-5)
	m.methodList.SetSize(w, h-5)
	m.ghRepoList.SetSize(w, h-5)
	m.copyMoveList.SetSize(w, h-5)
}

func (m editProjectModel) buildMethodList() FzfListModel {
	var items []FzfItem
	if m.ghAvail {
		items = append(items, methodItem{name: "GitHub"})
	}
	items = append(items, methodItem{name: "Git URL"})
	items = append(items, methodItem{name: "Local directory"})
	items = append(items, methodItem{name: "Empty (git init)"})
	list := NewFzfList(items, FzfListConfig{
		Placeholder: "No methods",
	})
	list.SetSize(m.width, m.height-5)
	return list
}

func (m editProjectModel) update(msg tea.Msg) (editProjectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case allReposLoadedMsg:
		m.loading = false
		m.reposErr = msg.err
		if msg.err == nil {
			m.ghRepoList = NewFzfList(msg.repos, FzfListConfig{
				MultiSelect:  true,
				PreviewFunc:  repoPreview,
				RenderItem:   renderRepoItem,
				Placeholder:  "No repos found",
				ListWidthPct: 0.45,
			})
			m.ghRepoList.SetSize(m.width, m.height-5)
		}
		return m, nil

	case cloneTickMsg:
		m.progress, _ = m.progress.update(msg)
		return m, listenForCloneProgress(m.cloneChan)

	case cloneAllDoneMsg:
		m.refreshRepoList()
		m.step = editStepMain
		return m, nil

	case addCompleteMsg:
		m.loading = false
		m.refreshRepoList()
		m.step = editStepMain
		return m, nil

	case repoRemovedMsg:
		m.refreshRepoList()
		m.step = editStepMain
		return m, nil

	case projectRenamedMsg:
		m.projectName = msg.newName
		m.projectPath = filepath.Join(paths.ProjectsDir(), msg.newName)
		m.step = editStepMain
		return m, nil

	case spinner.TickMsg:
		if m.step == editStepCloning {
			if m.cloneChan != nil {
				var cmd tea.Cmd
				m.progress, cmd = m.progress.update(msg)
				return m, cmd
			}
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		// Clear error on any keypress
		m.errMsg = ""

		switch m.step {
		case editStepMain:
			return m.updateMain(msg)
		case editStepRename:
			return m.updateRename(msg)
		case editStepConfirmRemove:
			return m.updateConfirmRemove(msg)
		case editStepMethod:
			return m.updateMethod(msg)
		case editStepRepos:
			return m.updateGHRepos(msg)
		case editStepGitURL:
			return m.updateGitURL(msg)
		case editStepLocalPath:
			return m.updateLocalPath(msg)
		case editStepCopyMove:
			return m.updateCopyMove(msg)
		}
	}
	return m, nil
}

func (m editProjectModel) updateMain(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	// Let fzf handle navigation first
	newList, consumed, result := m.repoList.HandleKey(msg.String())
	m.repoList = newList

	if result != nil {
		switch result.(type) {
		case FzfCancelMsg:
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenProjectList}
			}
		}
	}

	if consumed {
		return m, nil
	}

	if !m.repoList.IsSearching() {
		switch msg.String() {
		case "a":
			m.step = editStepMethod
			m.methodList = m.buildMethodList()
			return m, nil
		case "x", "d":
			item := m.repoList.SelectedItem()
			if item != nil {
				ri := item.(editRepoItem)
				m.removeTarget = ri.name
				m.step = editStepConfirmRemove
			}
			return m, nil
		case "r":
			m.step = editStepRename
			m.renameErr = ""
			ti := textinput.New()
			ti.SetValue(m.projectName)
			ti.Focus()
			ti.CharLimit = 50
			m.renameInput = ti
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m editProjectModel) updateRename(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := strings.TrimSpace(m.renameInput.Value())
		if newName == "" {
			m.renameErr = "Name cannot be empty"
			return m, nil
		}
		if newName == m.projectName {
			m.step = editStepMain
			return m, nil
		}
		// Check if name already exists
		newPath := filepath.Join(paths.ProjectsDir(), newName)
		if _, err := os.Stat(newPath); err == nil {
			m.renameErr = "A project with that name already exists"
			return m, nil
		}
		if err := project.Rename(m.projectName, newName); err != nil {
			m.renameErr = "Rename failed: " + err.Error()
			return m, nil
		}
		m.projectName = newName
		m.projectPath = newPath
		m.step = editStepMain
		return m, nil
	case "esc":
		m.step = editStepMain
		return m, nil
	default:
		m.renameErr = ""
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

func (m editProjectModel) updateConfirmRemove(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		name := m.removeTarget
		m.removeTarget = ""
		return m, func() tea.Msg {
			project.RemoveRepo(m.projectName, name)
			return repoRemovedMsg{}
		}
	default:
		m.removeTarget = ""
		m.step = editStepMain
		return m, nil
	}
}

func (m editProjectModel) updateMethod(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	newList, consumed, result := m.methodList.HandleKey(msg.String())
	m.methodList = newList

	if result != nil {
		switch r := result.(type) {
		case FzfConfirmMsg:
			mi := r.Item.(methodItem)
			switch mi.name {
			case "GitHub":
				m.step = editStepRepos
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, loadAllRepos)
			case "Git URL":
				m.step = editStepGitURL
				m.urlInput.SetValue("")
				m.urlInput.Focus()
				return m, textinput.Blink
			case "Local directory":
				m.step = editStepLocalPath
				m.pathInput.SetValue("")
				m.pathInput.Focus()
				return m, textinput.Blink
			case "Empty (git init)":
				m.step = editStepCloning
				m.statusText = "Initializing repo..."
				return m, tea.Batch(
					m.spinner.Tick,
					initEmptyRepoInProject(m.projectName),
				)
			}
		case FzfCancelMsg:
			m.step = editStepMain
			return m, nil
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m editProjectModel) updateGHRepos(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	newList, consumed, result := m.ghRepoList.HandleKey(msg.String())
	m.ghRepoList = newList

	if result != nil {
		switch r := result.(type) {
		case FzfConfirmMsg:
			var selected []repoItem
			for _, item := range r.Items {
				ri := item.(repoItem)
				selected = append(selected, ri)
			}
			if len(selected) == 0 {
				return m, nil
			}

			projDir := filepath.Join(paths.ProjectsDir(), m.projectName)
			var repoNames []string
			for _, r := range selected {
				name := r.NameWithOwner
				if name == "" {
					name = r.org + "/" + r.Name
				}
				repoNames = append(repoNames, name)
			}

			ch := git.ParallelClone(projDir, repoNames)
			m.cloneChan = ch
			m.step = editStepCloning
			m.progress = newProgressModel("Cloning repos", repoNames)
			m.progress.setSize(m.width, m.height)

			return m, tea.Batch(
				m.progress.Init(),
				listenForCloneProgress(ch),
			)
		case FzfCancelMsg:
			m.step = editStepMethod
			m.methodList = m.buildMethodList()
			return m, nil
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m editProjectModel) updateGitURL(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil
		}
		m.step = editStepCloning
		m.statusText = "Cloning..."
		return m, tea.Batch(
			m.spinner.Tick,
			cloneURLToProject(m.projectName, url),
		)
	case "esc":
		m.step = editStepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func (m editProjectModel) updateLocalPath(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		p := strings.TrimSpace(m.pathInput.Value())
		if p == "" {
			return m, nil
		}
		if strings.HasPrefix(p, "~/") {
			home, _ := os.UserHomeDir()
			p = filepath.Join(home, p[2:])
		}
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			m.errMsg = "Directory not found"
			return m, nil
		}
		m.localPath = p
		m.step = editStepCopyMove
		items := []FzfItem{
			copyMoveItem{name: "Copy"},
			copyMoveItem{name: "Move"},
		}
		m.copyMoveList = NewFzfList(items, FzfListConfig{
			Placeholder: "Choose action",
		})
		m.copyMoveList.SetSize(m.width, m.height-5)
		return m, nil
	case "esc":
		m.step = editStepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	default:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
}

func (m editProjectModel) updateCopyMove(msg tea.KeyMsg) (editProjectModel, tea.Cmd) {
	newList, consumed, result := m.copyMoveList.HandleKey(msg.String())
	m.copyMoveList = newList

	if result != nil {
		switch r := result.(type) {
		case FzfConfirmMsg:
			ci := r.Item.(copyMoveItem)
			move := ci.name == "Move"
			action := "Copying..."
			if move {
				action = "Moving..."
			}
			m.step = editStepCloning
			m.statusText = action
			return m, tea.Batch(
				m.spinner.Tick,
				addLocalDirToProject(m.projectName, m.localPath, move),
			)
		case FzfCancelMsg:
			m.step = editStepLocalPath
			m.pathInput.Focus()
			return m, textinput.Blink
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

// Commands for adding repos to existing projects

func cloneURLToProject(projectName, url string) tea.Cmd {
	return func() tea.Msg {
		projDir := filepath.Join(paths.ProjectsDir(), projectName)
		name := filepath.Base(strings.TrimSuffix(url, ".git"))
		if name == "" || name == "." {
			name = "repo"
		}
		git.Clone(url, filepath.Join(projDir, name))
		return addCompleteMsg{}
	}
}

func addLocalDirToProject(projectName, srcPath string, move bool) tea.Cmd {
	return func() tea.Msg {
		projDir := filepath.Join(paths.ProjectsDir(), projectName)
		name := filepath.Base(srcPath)
		dest := filepath.Join(projDir, name)
		if move {
			project.MoveDir(srcPath, dest)
		} else {
			project.CopyDir(srcPath, dest)
		}
		return addCompleteMsg{}
	}
}

func initEmptyRepoInProject(projectName string) tea.Cmd {
	return func() tea.Msg {
		projDir := filepath.Join(paths.ProjectsDir(), projectName)
		git.Init(projDir)
		return addCompleteMsg{}
	}
}

// Views

func (m editProjectModel) view() string {
	switch m.step {
	case editStepMain:
		return m.viewMain()
	case editStepRename:
		return m.viewRename()
	case editStepConfirmRemove:
		return m.viewConfirmRemove()
	case editStepMethod:
		return m.viewMethod()
	case editStepRepos:
		return m.viewGHRepos()
	case editStepGitURL:
		return m.viewGitURL()
	case editStepLocalPath:
		return m.viewLocalPath()
	case editStepCopyMove:
		return m.viewCopyMove()
	case editStepCloning:
		return m.viewCloning()
	}
	return ""
}

func (m editProjectModel) viewMain() string {
	repoCount := m.repoList.TotalCount()
	countStr := fmt.Sprintf("%d repo", repoCount)
	if repoCount != 1 {
		countStr += "s"
	}

	header := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName) +
		"  " + dimStyle.Render("("+countStr+")") + "\n"
	kb := renderKeybar(
		keyBind{"a", "add repo"},
		keyBind{"x", "remove"},
		keyBind{"r", "rename"},
		keyBind{"s", "search"},
		keyBind{"esc", "back"},
	)
	return header + kb + "\n\n" + m.repoList.View()
}

func (m editProjectModel) viewRename() string {
	header := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › rename") + "\n"
	kb := renderKeybar(keyBind{"enter", "confirm"}, keyBind{"esc", "cancel"})
	body := "\n  New name:\n\n  " + m.renameInput.View()
	if m.renameErr != "" {
		body += "\n\n  " + errorStyle.Render(m.renameErr)
	}
	return header + kb + "\n" + body
}

func (m editProjectModel) viewConfirmRemove() string {
	header := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName) + "\n"
	warning := "\n" + errorStyle.Render("  Remove '"+m.removeTarget+"'?") + "\n" +
		dimStyle.Render("  This will delete the repo directory from this project.") + "\n\n" +
		"  " + keyStyle.Render("y") + dimStyle.Render(" confirm  ") +
		keyStyle.Render("n") + dimStyle.Render(" cancel")
	return header + warning
}

func (m editProjectModel) viewMethod() string {
	breadcrumb := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › add repo") + "\n"
	kb := renderKeybar(keyBind{"enter", "select"}, keyBind{"esc", "back"})
	return breadcrumb + kb + "\n\n" + m.methodList.View()
}

func (m editProjectModel) viewGHRepos() string {
	breadcrumb := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › GitHub") + "\n"

	if m.loading {
		return breadcrumb + fmt.Sprintf("\n  %s Loading repos...", m.spinner.View())
	}
	if m.reposErr != nil {
		kb := renderKeybar(keyBind{"esc", "back"})
		return breadcrumb + "\n" + errorStyle.Render("  "+m.reposErr.Error()) + "\n\n" + kb
	}

	kb := renderKeybar(
		keyBind{"space", "toggle"},
		keyBind{"enter", "clone"},
		keyBind{"s", "search"},
		keyBind{"esc", "back"},
	)
	return breadcrumb + kb + "\n\n" + m.ghRepoList.View()
}

func (m editProjectModel) viewGitURL() string {
	breadcrumb := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › Git URL") + "\n"
	kb := renderKeybar(keyBind{"enter", "clone"}, keyBind{"esc", "back"})
	body := "\n  Repository URL:\n\n  " + m.urlInput.View()
	return breadcrumb + kb + "\n" + body
}

func (m editProjectModel) viewLocalPath() string {
	breadcrumb := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › local") + "\n"
	kb := renderKeybar(keyBind{"enter", "next"}, keyBind{"esc", "back"})
	body := "\n  Directory path:\n\n  " + m.pathInput.View()
	if m.errMsg != "" {
		body += "\n\n  " + errorStyle.Render(m.errMsg)
	}
	return breadcrumb + kb + "\n" + body
}

func (m editProjectModel) viewCopyMove() string {
	breadcrumb := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › "+filepath.Base(m.localPath)) + "\n"
	kb := renderKeybar(keyBind{"enter", "select"}, keyBind{"esc", "back"})
	return breadcrumb + kb + "\n\n" + m.copyMoveList.View()
}

func (m editProjectModel) viewCloning() string {
	if m.cloneChan != nil {
		return m.progress.view()
	}
	header := titleStyle.Render("EDIT PROJECT") +
		dimStyle.Render(" › "+m.projectName) + "\n"
	body := fmt.Sprintf("\n  %s %s", m.spinner.View(), m.statusText)
	return header + body
}

func renderEditRepoItem(item FzfItem, index int, cursor, selected bool, matched []int) string {
	ri := item.(editRepoItem)

	prefix := "  "
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}

	name := highlightMatches(ri.name, matched)

	// Show branch and dirty status inline
	meta := dimStyle.Render(" on ") + ri.branch
	if ri.dirtyCount > 0 {
		meta += " " + errorStyle.Render("●")
	} else {
		meta += " " + successStyle.Render("✓")
	}

	line := prefix + name + meta

	if cursor {
		line = fzfSelectedLine.Render(line)
	}
	return line
}

// editRepoPreview returns a preview function that shows detailed repo info.
func editRepoPreview(projectPath string) func(FzfItem, int, int) string {
	return func(item FzfItem, width, height int) string {
		ri := item.(editRepoItem)
		info := git.GetRepoInfo(projectPath, ri.name)

		var lines []string
		lines = append(lines, titleStyle.Render(ri.name))
		lines = append(lines, dimStyle.Render("on ")+info.Branch)
		lines = append(lines, "")

		// Ahead/behind
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

		// Status
		if info.Clean {
			lines = append(lines, successStyle.Render("✓ clean"))
		} else {
			lines = append(lines, errorStyle.Render(fmt.Sprintf("● %d modified", ri.dirtyCount)))
		}
		lines = append(lines, "")

		// Recent commits
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
}
