package tui

import (
	"github.com/ahtwr/cw/internal/claude"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/project"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/screen"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type activeScreen int

const (
	screenProjectExplorer activeScreen = iota
	screenProjectWizard
	screenAddRepo
	screenLauncher
)

type Model struct {
	screen activeScreen
	width  int
	height int

	selectedProject *project.Project
	launchConfig    *claude.LaunchConfig

	projectExplorer screen.ProjectExplorerModel
	projectWizard   screen.ProjectWizardModel
	addRepo         screen.AddRepoModel
	launcher        screen.LauncherModel
}

func NewModel(pluginDir string) Model {
	return Model{
		screen:      screenProjectExplorer,
		projectExplorer: screen.NewProjectExplorerModel(),
		launchConfig: &claude.LaunchConfig{
			PluginDir: pluginDir,
		},
	}
}

func NewModelWithProject(pluginDir string, proj *project.Project, bypass bool) Model {
	m := Model{
		screen:          screenLauncher,
		selectedProject: proj,
		launcher:        screen.NewLauncherModel(bypass, proj.Path),
		launchConfig: &claude.LaunchConfig{
			PluginDir:   pluginDir,
			WorkDir:     proj.Path,
			ProjectName: proj.Name,
		},
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.screen == screenLauncher && m.selectedProject != nil {
		return m.launcher.LoadSessions(m.selectedProject.Path)
	}
	return m.projectExplorer.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectExplorer.SetSize(msg.Width, msg.Height)
		m.projectWizard.SetSize(msg.Width, msg.Height)
		m.addRepo.SetSize(msg.Width, msg.Height)
		m.launcher.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, style.GlobalKeys.Quit) {
			if m.screen == screenProjectWizard {
				m.projectWizard.CleanupIfNeeded()
			}
			m.launchConfig.WorkDir = ""
			return m, tea.Quit
		}

	case shared.RepoSelectedMsg:
		full, err := project.Get(msg.ProjectName)
		if err != nil {
			return m, nil
		}
		for _, r := range full.Repos {
			if r.Name == msg.RepoName {
				m.launchConfig.WorkDir = r.Path
				m.launchConfig.ProjectName = msg.RepoName
				m.launchConfig.EditorMode = true

				go func() {
					ch := git.PullAll(full.Path, []string{msg.RepoName})
					for range ch {
					}
				}()

				return m, tea.Quit
			}
		}
		return m, nil

	case shared.ProjectSelectedMsg:
		m.selectedProject = msg.Project
		m.launchConfig.WorkDir = m.selectedProject.Path
		m.launchConfig.ProjectName = m.selectedProject.Name

		var repoNames []string
		for _, r := range m.selectedProject.Repos {
			repoNames = append(repoNames, r.Name)
		}
		go func() {
			ch := git.PullAll(m.selectedProject.Path, repoNames)
			for range ch {
			}
		}()

		if !project.HasMetadata(msg.Project.Name) {
			m.launchConfig.Prompt = "/cw:new-intention"
			m.launchConfig.SkipPermissions = true
			m.launchConfig.AutoSetup = true
			m.launchConfig.AutoCompactLimit = m.projectExplorer.AutoCompactLimit
			return m, tea.Quit
		}

		m.launchConfig.AutoCompactLimit = m.projectExplorer.AutoCompactLimit

		bypass := m.projectExplorer.BypassPerms
		m.screen = screenLauncher
		m.launcher = screen.NewLauncherModel(bypass, m.selectedProject.Path)
		m.launcher.SetSize(m.width, m.height)
		return m, m.launcher.LoadSessions(m.selectedProject.Path)

	case shared.ModeSelectedMsg:
		m.launchConfig.Mode = msg.Mode
		m.launchConfig.SkipPermissions = msg.SkipPermissions
		switch msg.SessionKind {
		case 1:
			m.launchConfig.Prompt = "/resume"
		case 2:
			m.launchConfig.SessionID = msg.SessionID
		}
		return m, tea.Quit

	case shared.NavigateMsg:
		switch msg.Screen {
		case shared.ScreenProjectExplorer:
			m.screen = screenProjectExplorer
			m.projectExplorer = screen.NewProjectExplorerModel()
			m.projectExplorer.SetSize(m.width, m.height)
			return m, m.projectExplorer.Init()
		case shared.ScreenProjectWizard:
			m.screen = screenProjectWizard
			m.projectWizard = screen.NewProjectWizardModel()
			m.projectWizard.SetSize(m.width, m.height)
			return m, m.projectWizard.Init()
		case shared.ScreenAddRepo:
			m.screen = screenAddRepo
			m.addRepo = screen.NewAddRepoModel(msg.ProjectName)
			m.addRepo.SetSize(m.width, m.height)
			return m, m.addRepo.Init()
		}
		return m, nil

	case shared.LaunchMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenProjectExplorer:
		m.projectExplorer, cmd = m.projectExplorer.Update(msg)
	case screenProjectWizard:
		m.projectWizard, cmd = m.projectWizard.Update(msg)
	case screenAddRepo:
		m.addRepo, cmd = m.addRepo.Update(msg)
	case screenLauncher:
		m.launcher, cmd = m.launcher.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.screen {
	case screenProjectExplorer:
		return m.projectExplorer.View()
	case screenProjectWizard:
		return m.projectWizard.View()
	case screenAddRepo:
		return m.addRepo.View()
	case screenLauncher:
		return m.launcher.View()
	}
	return ""
}

func (m Model) ShouldLaunch() bool {
	return m.launchConfig != nil && m.launchConfig.WorkDir != ""
}

func (m Model) LaunchConfig() claude.LaunchConfig {
	if m.launchConfig != nil {
		return *m.launchConfig
	}
	return claude.LaunchConfig{}
}

func (m Model) SelectedProject() *project.Project {
	return m.selectedProject
}
