package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/gh"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/project"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createStep int

const (
	stepName createStep = iota
	stepMethod
	stepRepos
	stepGitURL
	stepLocalPath
	stepCopyMove
	stepCloning
)

// Method items
type methodItem struct{ name string }

func (m methodItem) FilterValue() string { return m.name }

// Copy/Move items
type copyMoveItem struct{ name string }

func (c copyMoveItem) FilterValue() string { return c.name }

// Messages
type allReposLoadedMsg struct {
	repos []FzfItem // mix of orgDividerItem and repoItem
	err   error
}
type allClonesCompleteMsg struct{ projectName string }
type addCompleteMsg struct{}

// Divider for org groups
type orgDividerItem struct{ name string }

func (o orgDividerItem) FilterValue() string { return o.name }
func (o orgDividerItem) IsDivider() bool     { return true }

type repoItem struct {
	gh.Repo
	org string
}

func (r repoItem) FilterValue() string {
	return r.NameWithOwner + " " + r.Description + " " + r.PrimaryLanguage.Name
}

type createProjectModel struct {
	step   createStep
	width  int
	height int

	// Step: Name
	nameInput textinput.Model

	// Step: Method
	methodList FzfListModel

	// Step: Repos (all orgs combined)
	repoList FzfListModel
	reposErr error

	// Step: Git URL
	urlInput textinput.Model

	// Step: Local Path
	pathInput textinput.Model

	// Step: Copy/Move
	copyMoveList FzfListModel
	localPath    string

	// Step: Cloning
	spinner     spinner.Model
	clonesTotal int
	projectName string
	statusText  string

	// Step: Cloning (progress)
	progress  progressModel
	cloneChan <-chan git.CloneProgress

	// Tracks added repos count
	addedCount int

	loading bool
	ghAvail bool
}

func newCreateProjectModel() createProjectModel {
	ti := textinput.New()
	ti.Placeholder = "my-project"
	ti.Focus()
	ti.CharLimit = 50

	urlTi := textinput.New()
	urlTi.Placeholder = "https://github.com/user/repo.git"
	urlTi.CharLimit = 200

	pathTi := textinput.New()
	pathTi.Placeholder = "/path/to/directory"
	pathTi.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return createProjectModel{
		step:      stepName,
		nameInput: ti,
		urlInput:  urlTi,
		pathInput: pathTi,
		spinner:   sp,
		ghAvail:   gh.IsAvailable() && gh.IsAuthenticated(),
	}
}

func (m createProjectModel) cleanupIfNeeded() {
	if m.projectName != "" && m.addedCount == 0 {
		project.Delete(m.projectName)
	}
}

func (m createProjectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *createProjectModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.repoList.SetSize(w, h-5)
	m.methodList.SetSize(w, h-5)
	m.copyMoveList.SetSize(w, h-5)
}

func (m createProjectModel) buildMethodList() FzfListModel {
	var items []FzfItem
	items = append(items, methodItem{name: "Done"})
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

func loadAllRepos() tea.Msg {
	groups, err := gh.FetchRepos()
	if err != nil {
		return allReposLoadedMsg{err: err}
	}

	var items []FzfItem
	for _, g := range groups {
		items = append(items, orgDividerItem{name: g.Owner})
		for _, r := range g.Repos {
			items = append(items, repoItem{Repo: r, org: g.Owner})
		}
	}
	return allReposLoadedMsg{repos: items}
}

func listenForCloneProgress(ch <-chan git.CloneProgress) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-ch
		if !ok {
			return cloneAllDoneMsg{}
		}
		return cloneTickMsg(progress)
	}
}

func cloneURL(projectName, url string) tea.Cmd {
	return func() tea.Msg {
		projDir, err := project.Create(projectName)
		if err != nil {
			return addCompleteMsg{}
		}
		// Derive repo name from URL
		name := filepath.Base(strings.TrimSuffix(url, ".git"))
		if name == "" || name == "." {
			name = "repo"
		}
		git.Clone(url, filepath.Join(projDir, name))
		return addCompleteMsg{}
	}
}

func addLocalDir(projectName, srcPath string, move bool) tea.Cmd {
	return func() tea.Msg {
		projDir, err := project.Create(projectName)
		if err != nil {
			return addCompleteMsg{}
		}
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

func initEmptyRepo(projectName string) tea.Cmd {
	return func() tea.Msg {
		projDir, err := project.Create(projectName)
		if err != nil {
			return addCompleteMsg{}
		}
		git.Init(projDir)
		return addCompleteMsg{}
	}
}

func (m createProjectModel) update(msg tea.Msg) (createProjectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case allReposLoadedMsg:
		m.loading = false
		m.reposErr = msg.err
		if msg.err == nil {
			m.repoList = NewFzfList(msg.repos, FzfListConfig{
				MultiSelect:  true,
				PreviewFunc:  repoPreview,
				RenderItem:   renderRepoItem,
				Placeholder:  "No repos found",
				ListWidthPct: 0.45,
			})
			m.repoList.SetSize(m.width, m.height-5)
		}
		return m, nil

	case cloneTickMsg:
		m.progress, _ = m.progress.update(msg)
		return m, listenForCloneProgress(m.cloneChan)

	case cloneAllDoneMsg:
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenProjectList}
		}

	case allClonesCompleteMsg:
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenProjectList}
		}

	case addCompleteMsg:
		m.addedCount++
		m.loading = false
		m.step = stepMethod
		m.methodList = m.buildMethodList()
		return m, nil

	case spinner.TickMsg:
		if m.step == stepCloning {
			if m.cloneChan != nil {
				// Route to progress model for parallel clone
				var cmd tea.Cmd
				m.progress, cmd = m.progress.update(msg)
				return m, cmd
			}
			// Fallback to simple spinner for non-clone operations
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
		switch m.step {
		case stepName:
			return m.updateName(msg)
		case stepMethod:
			return m.updateMethod(msg)
		case stepRepos:
			return m.updateRepos(msg)
		case stepGitURL:
			return m.updateGitURL(msg)
		case stepLocalPath:
			return m.updateLocalPath(msg)
		case stepCopyMove:
			return m.updateCopyMove(msg)
		}
	}
	return m, nil
}

func (m createProjectModel) updateName(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, nil
		}
		m.projectName = name
		m.step = stepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	case "esc":
		return m, func() tea.Msg { return switchScreenMsg{screen: screenProjectList} }
	default:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
}

func (m createProjectModel) updateMethod(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
	newList, consumed, result := m.methodList.HandleKey(msg.String())
	m.methodList = newList

	if result != nil {
		switch r := result.(type) {
		case FzfConfirmMsg:
			mi := r.Item.(methodItem)
			switch mi.name {
			case "Done":
				if m.addedCount == 0 {
					// Create empty project dir
					project.Create(m.projectName)
				}
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenProjectList}
				}
			case "GitHub":
				m.step = stepRepos
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, loadAllRepos)
			case "Git URL":
				m.step = stepGitURL
				m.urlInput.SetValue("")
				m.urlInput.Focus()
				return m, textinput.Blink
			case "Local directory":
				m.step = stepLocalPath
				m.pathInput.SetValue("")
				m.pathInput.Focus()
				return m, textinput.Blink
			case "Empty (git init)":
				m.step = stepCloning
				m.statusText = "Initializing repo..."
				return m, tea.Batch(
					m.spinner.Tick,
					initEmptyRepo(m.projectName),
				)
			}
		case FzfCancelMsg:
			m.step = stepName
			m.nameInput.Focus()
			return m, textinput.Blink
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m createProjectModel) updateRepos(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
	newList, consumed, result := m.repoList.HandleKey(msg.String())
	m.repoList = newList

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

			// Create project directory
			projDir, err := project.Create(m.projectName)
			if err != nil {
				return m, func() tea.Msg { return switchScreenMsg{screen: screenProjectList} }
			}

			// Build repo name list for parallel clone
			var repoNames []string
			for _, r := range selected {
				name := r.NameWithOwner
				if name == "" {
					name = r.org + "/" + r.Name
				}
				repoNames = append(repoNames, name)
			}

			// Start parallel clone
			ch := git.ParallelClone(projDir, repoNames)
			m.cloneChan = ch
			m.step = stepCloning
			m.progress = newProgressModel("Cloning repos", repoNames)
			m.progress.setSize(m.width, m.height)

			return m, tea.Batch(
				m.progress.Init(),
				listenForCloneProgress(ch),
			)
		case FzfCancelMsg:
			m.step = stepMethod
			m.methodList = m.buildMethodList()
			return m, nil
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m createProjectModel) updateGitURL(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil
		}
		m.step = stepCloning
		m.statusText = "Cloning..."
		return m, tea.Batch(
			m.spinner.Tick,
			cloneURL(m.projectName, url),
		)
	case "esc":
		m.step = stepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func (m createProjectModel) updateLocalPath(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		p := strings.TrimSpace(m.pathInput.Value())
		if p == "" {
			return m, nil
		}
		// Expand ~
		if strings.HasPrefix(p, "~/") {
			home, _ := os.UserHomeDir()
			p = filepath.Join(home, p[2:])
		}
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			// Show error inline by keeping the step
			return m, nil
		}
		m.localPath = p
		m.step = stepCopyMove
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
		m.step = stepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	default:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
}

func (m createProjectModel) updateCopyMove(msg tea.KeyMsg) (createProjectModel, tea.Cmd) {
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
			m.step = stepCloning
			m.statusText = action
			return m, tea.Batch(
				m.spinner.Tick,
				addLocalDir(m.projectName, m.localPath, move),
			)
		case FzfCancelMsg:
			m.step = stepLocalPath
			m.pathInput.Focus()
			return m, textinput.Blink
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m createProjectModel) view() string {
	switch m.step {
	case stepName:
		return m.viewName()
	case stepMethod:
		return m.viewMethod()
	case stepRepos:
		return m.viewRepos()
	case stepGitURL:
		return m.viewGitURL()
	case stepLocalPath:
		return m.viewLocalPath()
	case stepCopyMove:
		return m.viewCopyMove()
	case stepCloning:
		return m.viewCloning()
	}
	return ""
}

func (m createProjectModel) viewName() string {
	header := titleStyle.Render("NEW PROJECT") + "\n"
	kb := renderKeybar(keyBind{"enter", "next"}, keyBind{"esc", "cancel"})
	body := "\n  Project name:\n\n  " + m.nameInput.View()
	return header + kb + "\n" + body
}

func (m createProjectModel) viewMethod() string {
	breadcrumb := titleStyle.Render("NEW PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › add repos") + "\n"

	added := ""
	if m.addedCount > 0 {
		added = dimStyle.Render(fmt.Sprintf("  %d added", m.addedCount)) + "\n"
	}

	kb := renderKeybar(keyBind{"enter", "select"}, keyBind{"esc", "back"})
	return breadcrumb + kb + "\n" + added + "\n" + m.methodList.View()
}

func (m createProjectModel) viewRepos() string {
	breadcrumb := titleStyle.Render("NEW PROJECT") +
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
	return breadcrumb + kb + "\n\n" + m.repoList.View()
}

func (m createProjectModel) viewGitURL() string {
	breadcrumb := titleStyle.Render("NEW PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › Git URL") + "\n"
	kb := renderKeybar(keyBind{"enter", "clone"}, keyBind{"esc", "back"})
	body := "\n  Repository URL:\n\n  " + m.urlInput.View()
	return breadcrumb + kb + "\n" + body
}

func (m createProjectModel) viewLocalPath() string {
	breadcrumb := titleStyle.Render("NEW PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › local") + "\n"
	kb := renderKeybar(keyBind{"enter", "next"}, keyBind{"esc", "back"})
	body := "\n  Directory path:\n\n  " + m.pathInput.View()
	return breadcrumb + kb + "\n" + body
}

func (m createProjectModel) viewCopyMove() string {
	breadcrumb := titleStyle.Render("NEW PROJECT") +
		dimStyle.Render(" › "+m.projectName+" › "+filepath.Base(m.localPath)) + "\n"
	kb := renderKeybar(keyBind{"enter", "select"}, keyBind{"esc", "back"})
	return breadcrumb + kb + "\n\n" + m.copyMoveList.View()
}

func (m createProjectModel) viewCloning() string {
	if m.cloneChan != nil {
		return m.progress.view()
	}
	header := titleStyle.Render("NEW PROJECT") +
		dimStyle.Render(" › "+m.projectName) + "\n"
	body := fmt.Sprintf("\n  %s %s", m.spinner.View(), m.statusText)
	return header + body
}

// Render callbacks

func renderRepoItem(item FzfItem, index int, cursor, selected bool, matched []int) string {
	if isDivider(item) {
		d := item.(orgDividerItem)
		return fzfDividerStyle.Render("  ── " + d.name + " ──")
	}

	ri := item.(repoItem)

	prefix := "  "
	if cursor {
		prefix = fzfCursorPrefix.Render("▸ ")
	}

	marker := fzfMarkerNormal.Render("○ ")
	if selected {
		marker = fzfMarkerSelected.Render("● ")
	}

	name := highlightMatches(ri.Name, matched)
	lang := ""
	if ri.PrimaryLanguage.Name != "" {
		lang = " " + dimStyle.Render("["+ri.PrimaryLanguage.Name+"]")
	}

	line := prefix + marker + name + lang
	if cursor {
		line = fzfSelectedLine.Render(line)
	}
	return line
}

func repoPreview(item FzfItem, width, height int) string {
	ri := item.(repoItem)
	var lines []string
	title := ri.NameWithOwner
	if title == "" {
		title = ri.Name
	}
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")
	if ri.Description != "" {
		lines = append(lines, ri.Description)
		lines = append(lines, "")
	}
	if ri.PrimaryLanguage.Name != "" {
		lines = append(lines, dimStyle.Render("Language: ")+ri.PrimaryLanguage.Name)
	}
	if ri.StargazerCount > 0 {
		lines = append(lines, dimStyle.Render("Stars: ")+fmt.Sprintf("%d", ri.StargazerCount))
	}
	if ri.UpdatedAt != "" && len(ri.UpdatedAt) >= 10 {
		lines = append(lines, dimStyle.Render("Updated: ")+ri.UpdatedAt[:10])
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
