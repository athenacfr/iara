package tui

import (
	"os"
	"path/filepath"

	"github.com/ahtwr/iara/internal/claude"
	"github.com/ahtwr/iara/internal/git"
	"github.com/ahtwr/iara/internal/project"
	"github.com/ahtwr/iara/internal/tui/shared"
	"github.com/ahtwr/iara/internal/tui/screen"
	"github.com/ahtwr/iara/internal/tui/style"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type activeScreen int

const (
	screenProjectExplorer activeScreen = iota
	screenProjectWizard
	screenAddRepo
	screenLauncher
	screenSettings
	screenTaskSelect
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
	settings        screen.SettingsModel
	taskSelect      screen.TaskSelectModel
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
		screen:          screenTaskSelect,
		selectedProject: proj,
		taskSelect:      screen.NewTaskSelectModel(proj.Path, proj.Name),
		launchConfig: &claude.LaunchConfig{
			PluginDir:   pluginDir,
			WorkDir:     proj.Path,
			ProjectName: proj.Name,
		},
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.screen == screenTaskSelect && m.selectedProject != nil {
		return m.taskSelect.Init()
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
		m.settings.SetSize(msg.Width, msg.Height)
		m.taskSelect.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, style.GlobalKeys.Quit) {
			if m.screen == screenProjectExplorer {
				m.launchConfig.WorkDir = ""
				return m, tea.Quit
			}
			// Treat ctrl+c as escape on non-main screens
			msg = tea.KeyMsg(tea.Key{Type: tea.KeyEscape})
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
		m.launchConfig.AutoCompactLimit = m.projectExplorer.AutoCompactLimit

		var repoNames []string
		for _, r := range m.selectedProject.Repos {
			repoNames = append(repoNames, r.Name)
		}
		go func() {
			ch := git.PullAll(m.selectedProject.Path, repoNames)
			for range ch {
			}
		}()

		// No metadata → auto-setup project
		if !project.HasMetadata(msg.Project.Name) {
			m.launchConfig.Prompt = "/iara:setup-project"
			m.launchConfig.SkipPermissions = true
			m.launchConfig.AutoSetup = true
			return m, tea.Quit
		}

		// Navigate to task selection
		m.screen = screenTaskSelect
		m.taskSelect = screen.NewTaskSelectModel(m.selectedProject.Path, m.selectedProject.Name)
		m.taskSelect.SetSize(m.width, m.height)
		return m, m.taskSelect.Init()

	case shared.TaskSelectedMsg:
		if msg.IsNew {
			// Launch Claude with /iara:new-task in the project directory
			m.launchConfig.WorkDir = msg.WorkDir
			m.launchConfig.Prompt = "/iara:new-task"
			m.launchConfig.SkipPermissions = true
			m.launchConfig.AutoSetup = true
			return m, tea.Quit
		}

		// Set WorkDir and SessionsDir based on task
		m.launchConfig.WorkDir = msg.WorkDir
		m.launchConfig.SessionsDir = msg.SessionsDir

		if msg.Task != nil {
			m.launchConfig.TaskID = msg.Task.ID
			m.launchConfig.TaskName = msg.Task.Name

			// Pull worktree repos in background
			worktreeBase := msg.WorkDir
			var repoNames []string
			entries, _ := os.ReadDir(worktreeBase)
			for _, e := range entries {
				if e.IsDir() && git.IsRepo(filepath.Join(worktreeBase, e.Name())) {
					repoNames = append(repoNames, e.Name())
				}
			}
			if len(repoNames) > 0 {
				go func() {
					ch := git.PullAll(worktreeBase, repoNames)
					for range ch {
					}
				}()
			}
		}

		bypass := m.projectExplorer.BypassPerms
		s := project.LoadGlobalSettings()
		taskName := ""
		if msg.Task != nil {
			taskName = msg.Task.Name
		}

		m.screen = screenLauncher
		m.launcher = screen.NewLauncherModelForTask(bypass, msg.SessionsDir, s.DefaultMode, taskName)
		m.launcher.SetSize(m.width, m.height)
		return m, m.launcher.LoadSessions()

	case shared.ModeSelectedMsg:
		m.launchConfig.Mode = msg.Mode
		m.launchConfig.SkipPermissions = msg.SkipPermissions
		switch msg.SessionKind {
		case 1:
			m.launchConfig.Prompt = "/resume"
		case 2:
			m.launchConfig.ResumeSessionID = msg.ResumeSessionID
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
		case shared.ScreenSettings:
			m.screen = screenSettings
			m.settings = screen.NewSettingsModel()
			m.settings.SetSize(m.width, m.height)
			return m, m.settings.Init()
		case shared.ScreenTaskSelect:
			if m.selectedProject != nil {
				m.screen = screenTaskSelect
				m.taskSelect = screen.NewTaskSelectModel(m.selectedProject.Path, m.selectedProject.Name)
				m.taskSelect.SetSize(m.width, m.height)
				return m, m.taskSelect.Init()
			}
			// Fallback to project explorer if no project selected
			m.screen = screenProjectExplorer
			m.projectExplorer = screen.NewProjectExplorerModel()
			m.projectExplorer.SetSize(m.width, m.height)
			return m, m.projectExplorer.Init()
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
	case screenSettings:
		m.settings, cmd = m.settings.Update(msg)
	case screenTaskSelect:
		m.taskSelect, cmd = m.taskSelect.Update(msg)
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
	case screenSettings:
		return m.settings.View()
	case screenTaskSelect:
		return m.taskSelect.View()
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
