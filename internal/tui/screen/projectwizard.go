package screen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/gh"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/ahtwr/cw/internal/tui/widget"
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
	stepCloning
)

type ProjectWizardModel struct {
	step   createStep
	width  int
	height int

	nameInput    textinput.Model
	methodList  widget.FzfListModel
	repoList    widget.FzfListModel
	reposErr    error
	urlInput    textinput.Model
	spinner     spinner.Model
	clonesTotal  int
	projectName  string
	statusText   string
	progress     widget.ProgressModel
	cloneChan    <-chan git.CloneProgress
	addedCount   int
	loading      bool
	ghAvail      bool
}

func NewProjectWizardModel() ProjectWizardModel {
	ti := textinput.New()
	ti.Placeholder = "my-project"
	ti.Focus()
	ti.CharLimit = 50

	urlTi := textinput.New()
	urlTi.Placeholder = "https://github.com/user/repo.git"
	urlTi.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return ProjectWizardModel{
		step:      stepName,
		nameInput: ti,
		urlInput:  urlTi,
		spinner:   sp,
		ghAvail:   gh.IsAvailable() && gh.IsAuthenticated(),
	}
}

func (m ProjectWizardModel) CleanupIfNeeded() {
	if m.projectName != "" && m.addedCount == 0 {
		project.Delete(m.projectName)
	}
}

func (m ProjectWizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ProjectWizardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.repoList.SetSize(w, h-5)
	m.methodList.SetSize(w, h-5)
}

func (m ProjectWizardModel) buildMethodList() widget.FzfListModel {
	var items []widget.FzfItem
	items = append(items, shared.MethodItem{Name: "Done"})
	if m.ghAvail {
		items = append(items, shared.MethodItem{Name: "GitHub"})
	}
	items = append(items, shared.MethodItem{Name: "Git URL"})
	items = append(items, shared.MethodItem{Name: "Empty (git init)"})
	list := widget.NewFzfList(items, widget.FzfListConfig{
		Placeholder: "No methods",
	})
	list.SetSize(m.width, m.height-5)
	return list
}

func cloneURL(projectName, url string) tea.Cmd {
	return func() tea.Msg {
		projDir, err := project.Create(projectName)
		if err != nil {
			return shared.AddCompleteMsg{}
		}
		name := filepath.Base(strings.TrimSuffix(url, ".git"))
		if name == "" || name == "." {
			name = "repo"
		}
		git.Clone(url, filepath.Join(projDir, name))
		return shared.AddCompleteMsg{}
	}
}

func initEmptyRepo(projectName string) tea.Cmd {
	return func() tea.Msg {
		projDir, err := project.Create(projectName)
		if err != nil {
			return shared.AddCompleteMsg{}
		}
		git.Init(projDir)
		return shared.AddCompleteMsg{}
	}
}

func (m ProjectWizardModel) Update(msg tea.Msg) (ProjectWizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.AllReposLoadedMsg:
		m.loading = false
		m.reposErr = msg.Err
		if msg.Err == nil {
			m.repoList = widget.NewFzfList(msg.Repos, widget.FzfListConfig{
				MultiSelect:  true,
				PreviewFunc:  shared.RepoPreview,
				RenderItem:   shared.RenderRepoItem,
				Placeholder:  "No repos found",
				ListWidthPct: 0.45,
			})
			m.repoList.SetSize(m.width, m.height-5)
		}
		return m, nil

	case widget.CloneTickMsg:
		m.progress, _ = m.progress.Update(msg)
		return m, shared.ListenForCloneProgress(m.cloneChan)

	case widget.CloneAllDoneMsg:
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
		}

	case shared.AllClonesCompleteMsg:
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
		}

	case shared.AddCompleteMsg:
		m.addedCount++
		m.loading = false
		m.step = stepMethod
		m.methodList = m.buildMethodList()
		return m, nil

	case spinner.TickMsg:
		if m.step == stepCloning {
			if m.cloneChan != nil {
				var cmd tea.Cmd
				m.progress, cmd = m.progress.Update(msg)
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
		switch m.step {
		case stepName:
			return m.updateName(msg)
		case stepMethod:
			return m.updateMethod(msg)
		case stepRepos:
			return m.updateRepos(msg)
		case stepGitURL:
			return m.updateGitURL(msg)
		}
	}
	return m, nil
}

func (m ProjectWizardModel) updateName(msg tea.KeyMsg) (ProjectWizardModel, tea.Cmd) {
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
		return m, func() tea.Msg { return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer} }
	default:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
}

func (m ProjectWizardModel) updateMethod(msg tea.KeyMsg) (ProjectWizardModel, tea.Cmd) {
	newList, consumed, result := m.methodList.HandleKey(msg.String())
	m.methodList = newList

	if result != nil {
		switch r := result.(type) {
		case widget.FzfConfirmMsg:
			mi := r.Item.(shared.MethodItem)
			switch mi.Name {
			case "Done":
				if m.addedCount == 0 {
					project.Create(m.projectName)
				}
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
				}
			case "GitHub":
				m.step = stepRepos
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, shared.LoadAllRepos)
			case "Git URL":
				m.step = stepGitURL
				m.urlInput.SetValue("")
				m.urlInput.Focus()
				return m, textinput.Blink
			case "Empty (git init)":
				m.step = stepCloning
				m.statusText = "Initializing repo..."
				return m, tea.Batch(
					m.spinner.Tick,
					initEmptyRepo(m.projectName),
				)
			}
		case widget.FzfCancelMsg:
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

func (m ProjectWizardModel) updateRepos(msg tea.KeyMsg) (ProjectWizardModel, tea.Cmd) {
	newList, consumed, result := m.repoList.HandleKey(msg.String())
	m.repoList = newList

	if result != nil {
		switch r := result.(type) {
		case widget.FzfConfirmMsg:
			var selected []shared.RepoItem
			for _, item := range r.Items {
				ri := item.(shared.RepoItem)
				selected = append(selected, ri)
			}
			if len(selected) == 0 {
				return m, nil
			}

			projDir, err := project.Create(m.projectName)
			if err != nil {
				return m, func() tea.Msg { return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer} }
			}

			var repoNames []string
			for _, r := range selected {
				name := r.NameWithOwner
				if name == "" {
					name = r.Org + "/" + r.Name
				}
				repoNames = append(repoNames, name)
			}

			ch := git.ParallelClone(projDir, repoNames)
			m.cloneChan = ch
			m.step = stepCloning
			m.progress = widget.NewProgressModel("Cloning repos", repoNames)
			m.progress.SetSize(m.width, m.height)

			return m, tea.Batch(
				m.progress.Init(),
				shared.ListenForCloneProgress(ch),
			)
		case widget.FzfCancelMsg:
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

func (m ProjectWizardModel) updateGitURL(msg tea.KeyMsg) (ProjectWizardModel, tea.Cmd) {
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

func (m ProjectWizardModel) View() string {
	switch m.step {
	case stepName:
		return m.viewName()
	case stepMethod:
		return m.viewMethod()
	case stepRepos:
		return m.viewRepos()
	case stepGitURL:
		return m.viewGitURL()
	case stepCloning:
		return m.viewCloning()
	}
	return ""
}

func (m ProjectWizardModel) viewName() string {
	header := style.TitleStyle.Render("NEW PROJECT") + "\n"
	kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "next"}, style.KeyBind{Key: "esc", Desc: "cancel"})
	body := "\n  Project name:\n\n  " + m.nameInput.View()
	return header + kb + "\n" + body
}

func (m ProjectWizardModel) viewMethod() string {
	breadcrumb := style.TitleStyle.Render("NEW PROJECT") +
		style.DimStyle.Render(" › "+m.projectName+" › add repos") + "\n"

	added := ""
	if m.addedCount > 0 {
		added = style.DimStyle.Render(fmt.Sprintf("  %d added", m.addedCount)) + "\n"
	}

	kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "select"}, style.KeyBind{Key: "esc", Desc: "back"})
	return breadcrumb + kb + "\n" + added + "\n" + m.methodList.View()
}

func (m ProjectWizardModel) viewRepos() string {
	breadcrumb := style.TitleStyle.Render("NEW PROJECT") +
		style.DimStyle.Render(" › "+m.projectName+" › GitHub") + "\n"

	if m.loading {
		return breadcrumb + fmt.Sprintf("\n  %s Loading repos...", m.spinner.View())
	}
	if m.reposErr != nil {
		kb := style.RenderKeybar(style.KeyBind{Key: "esc", Desc: "back"})
		return breadcrumb + "\n" + style.ErrorStyle.Render("  "+m.reposErr.Error()) + "\n\n" + kb
	}

	kb := style.RenderKeybar(
		style.KeyBind{Key: "space", Desc: "toggle"},
		style.KeyBind{Key: "enter", Desc: "clone"},
		style.KeyBind{Key: "s", Desc: "search"},
		style.KeyBind{Key: "esc", Desc: "back"},
	)
	return breadcrumb + kb + "\n\n" + m.repoList.View()
}

func (m ProjectWizardModel) viewGitURL() string {
	breadcrumb := style.TitleStyle.Render("NEW PROJECT") +
		style.DimStyle.Render(" › "+m.projectName+" › Git URL") + "\n"
	kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "clone"}, style.KeyBind{Key: "esc", Desc: "back"})
	body := "\n  Repository URL:\n\n  " + m.urlInput.View()
	return breadcrumb + kb + "\n" + body
}

func (m ProjectWizardModel) viewCloning() string {
	if m.cloneChan != nil {
		return m.progress.View()
	}
	header := style.TitleStyle.Render("NEW PROJECT") +
		style.DimStyle.Render(" › "+m.projectName) + "\n"
	body := fmt.Sprintf("\n  %s %s", m.spinner.View(), m.statusText)
	return header + body
}
