package screen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ahtwr/cw/internal/gh"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/paths"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/ahtwr/cw/internal/tui/widget"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type addRepoStep int

const (
	addRepoStepMethod addRepoStep = iota
	addRepoStepGHRepos
	addRepoStepGitURL
	addRepoStepCloning
)

type AddRepoModel struct {
	step        addRepoStep
	width       int
	height      int
	projectName string

	methodList widget.FzfListModel
	ghRepoList widget.FzfListModel
	urlInput   textinput.Model
	spinner    spinner.Model
	statusText string
	progress   widget.ProgressModel
	cloneChan  <-chan git.CloneProgress
	loading    bool
	ghAvail    bool
	reposErr   error
}

func NewAddRepoModel(projectName string) AddRepoModel {
	urlTi := textinput.New()
	urlTi.Placeholder = "https://github.com/user/repo.git"
	urlTi.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := AddRepoModel{
		step:        addRepoStepMethod,
		projectName: projectName,
		urlInput:    urlTi,
		spinner:     sp,
		ghAvail:     gh.IsAvailable() && gh.IsAuthenticated(),
	}
	m.methodList = m.buildMethodList()
	return m
}

func (m AddRepoModel) buildMethodList() widget.FzfListModel {
	var items []widget.FzfItem
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

func (m AddRepoModel) Init() tea.Cmd {
	return nil
}

func (m *AddRepoModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.methodList.SetSize(w, h-5)
	m.ghRepoList.SetSize(w, h-5)
}

func (m AddRepoModel) Update(msg tea.Msg) (AddRepoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.AllReposLoadedMsg:
		m.loading = false
		m.reposErr = msg.Err
		if msg.Err == nil {
			m.ghRepoList = widget.NewFzfList(msg.Repos, widget.FzfListConfig{
				MultiSelect:  true,
				PreviewFunc:  shared.RepoPreview,
				RenderItem:   shared.RenderRepoItem,
				Placeholder:  "No repos found",
				ListWidthPct: 0.45,
			})
			m.ghRepoList.SetSize(m.width, m.height-5)
		}
		return m, nil

	case widget.CloneTickMsg:
		m.progress, _ = m.progress.Update(msg)
		return m, shared.ListenForCloneProgress(m.cloneChan)

	case widget.CloneAllDoneMsg:
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
		}

	case shared.AddCompleteMsg:
		m.loading = false
		return m, func() tea.Msg {
			return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
		}

	case spinner.TickMsg:
		if m.step == addRepoStepCloning {
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
		case addRepoStepMethod:
			return m.updateMethod(msg)
		case addRepoStepGHRepos:
			return m.updateGHRepos(msg)
		case addRepoStepGitURL:
			return m.updateGitURL(msg)
		}
	}
	return m, nil
}

func (m AddRepoModel) updateMethod(msg tea.KeyMsg) (AddRepoModel, tea.Cmd) {
	newList, consumed, result := m.methodList.HandleKey(msg.String())
	m.methodList = newList

	if result != nil {
		switch r := result.(type) {
		case widget.FzfConfirmMsg:
			mi := r.Item.(shared.MethodItem)
			switch mi.Name {
			case "GitHub":
				m.step = addRepoStepGHRepos
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, shared.LoadAllRepos)
			case "Git URL":
				m.step = addRepoStepGitURL
				m.urlInput.SetValue("")
				m.urlInput.Focus()
				return m, textinput.Blink
			case "Empty (git init)":
				m.step = addRepoStepCloning
				m.statusText = "Initializing repo..."
				return m, tea.Batch(
					m.spinner.Tick,
					initEmptyRepoInProject(m.projectName),
				)
			}
		case widget.FzfCancelMsg:
			return m, func() tea.Msg {
				return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
			}
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m AddRepoModel) updateGHRepos(msg tea.KeyMsg) (AddRepoModel, tea.Cmd) {
	newList, consumed, result := m.ghRepoList.HandleKey(msg.String())
	m.ghRepoList = newList

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

			projDir := filepath.Join(paths.ProjectsDir(), m.projectName)
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
			m.step = addRepoStepCloning
			m.progress = widget.NewProgressModel("Cloning repos", repoNames)
			m.progress.SetSize(m.width, m.height)

			return m, tea.Batch(
				m.progress.Init(),
				shared.ListenForCloneProgress(ch),
			)
		case widget.FzfCancelMsg:
			m.step = addRepoStepMethod
			m.methodList = m.buildMethodList()
			return m, nil
		}
	}

	if consumed {
		return m, nil
	}
	return m, nil
}

func (m AddRepoModel) updateGitURL(msg tea.KeyMsg) (AddRepoModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil
		}
		m.step = addRepoStepCloning
		m.statusText = "Cloning..."
		return m, tea.Batch(
			m.spinner.Tick,
			cloneURLToProject(m.projectName, url),
		)
	case "esc":
		m.step = addRepoStepMethod
		m.methodList = m.buildMethodList()
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func cloneURLToProject(projectName, url string) tea.Cmd {
	return func() tea.Msg {
		projDir := filepath.Join(paths.ProjectsDir(), projectName)
		name := filepath.Base(strings.TrimSuffix(url, ".git"))
		if name == "" || name == "." {
			name = "repo"
		}
		git.Clone(url, filepath.Join(projDir, name))
		return shared.AddCompleteMsg{}
	}
}

func initEmptyRepoInProject(projectName string) tea.Cmd {
	return func() tea.Msg {
		projDir := filepath.Join(paths.ProjectsDir(), projectName)
		git.Init(projDir)
		return shared.AddCompleteMsg{}
	}
}

// Views

func (m AddRepoModel) View() string {
	switch m.step {
	case addRepoStepMethod:
		return m.viewMethod()
	case addRepoStepGHRepos:
		return m.viewGHRepos()
	case addRepoStepGitURL:
		return m.viewGitURL()
	case addRepoStepCloning:
		return m.viewCloning()
	}
	return ""
}

func (m AddRepoModel) viewMethod() string {
	breadcrumb := style.TitleStyle.Render("ADD REPO") +
		style.DimStyle.Render(" › "+m.projectName) + "\n"
	kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "select"}, style.KeyBind{Key: "esc", Desc: "back"})
	return breadcrumb + kb + "\n\n" + m.methodList.View()
}

func (m AddRepoModel) viewGHRepos() string {
	breadcrumb := style.TitleStyle.Render("ADD REPO") +
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
	return breadcrumb + kb + "\n\n" + m.ghRepoList.View()
}

func (m AddRepoModel) viewGitURL() string {
	breadcrumb := style.TitleStyle.Render("ADD REPO") +
		style.DimStyle.Render(" › "+m.projectName+" › Git URL") + "\n"
	kb := style.RenderKeybar(style.KeyBind{Key: "enter", Desc: "clone"}, style.KeyBind{Key: "esc", Desc: "back"})
	body := "\n  Repository URL:\n\n  " + m.urlInput.View()
	return breadcrumb + kb + "\n" + body
}

func (m AddRepoModel) viewCloning() string {
	if m.cloneChan != nil {
		return m.progress.View()
	}
	header := style.TitleStyle.Render("ADD REPO") +
		style.DimStyle.Render(" › "+m.projectName) + "\n"
	body := fmt.Sprintf("\n  %s %s", m.spinner.View(), m.statusText)
	return header + body
}
