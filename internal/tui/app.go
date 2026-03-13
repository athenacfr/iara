package tui

import (
	"github.com/ahtwr/cw/internal/claude"
	"github.com/ahtwr/cw/internal/config"
	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/project"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenProjectList screen = iota
	screenCreateProject
	screenEditProject
	screenModeSelect
)

// Messages for screen transitions
type projectSelectedMsg struct{ project *project.Project }
type repoSelectedMsg struct {
	projectName string
	repoName    string
}
type modeSelectedMsg struct {
	mode            config.Mode
	skipPermissions bool
}
type switchScreenMsg struct {
	screen      screen
	projectName string // used for edit screen
	addRepo     bool   // skip to add-repo flow
}
type launchMsg struct{}

type Model struct {
	screen screen
	width  int
	height int

	// Shared state
	selectedProject *project.Project
	selectedMode    config.Mode
	launchConfig    *claude.LaunchConfig

	// Sub-models
	projectList   projectListModel
	createProject createProjectModel
	editProject   editProjectModel
	modeSelect    modeSelectModel
}

func NewModel(pluginDir string) Model {
	return Model{
		screen:      screenProjectList,
		projectList: newProjectListModel(),
		modeSelect:  newModeSelectModel(),
		launchConfig: &claude.LaunchConfig{
			PluginDir: pluginDir,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return m.projectList.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectList.setSize(msg.Width, msg.Height)
		m.createProject.setSize(msg.Width, msg.Height)
		m.editProject.setSize(msg.Width, msg.Height)
		m.modeSelect.setSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, globalKeys.Quit) {
			if m.screen == screenCreateProject {
				m.createProject.cleanupIfNeeded()
			}
			m.launchConfig.WorkDir = ""
			return m, tea.Quit
		}

	case repoSelectedMsg:
		full, err := project.Get(msg.projectName)
		if err != nil {
			return m, nil
		}
		for _, r := range full.Repos {
			if r.Name == msg.repoName {
				m.launchConfig.WorkDir = r.Path
				m.launchConfig.ProjectName = msg.repoName
				m.launchConfig.EditorMode = true

				go func() {
					ch := git.PullAll(full.Path, []string{msg.repoName})
					for range ch {
					}
				}()

				return m, tea.Quit
			}
		}
		return m, nil

	case projectSelectedMsg:
		m.selectedProject = msg.project
		m.launchConfig.WorkDir = m.selectedProject.Path
		m.launchConfig.ProjectName = m.selectedProject.Name

		// Background pull all repos (fire-and-forget)
		var repoNames []string
		for _, r := range m.selectedProject.Repos {
			repoNames = append(repoNames, r.Name)
		}
		go func() {
			ch := git.PullAll(m.selectedProject.Path, repoNames)
			for range ch {
			} // drain the channel
		}()

		if !project.HasMetadata(msg.project.Name) {
			// No metadata — launch intention setup, skip mode
			m.launchConfig.Prompt = "/cw:new-intention"
			m.launchConfig.SkipPermissions = true
			m.launchConfig.AutoSetup = true
			return m, tea.Quit
		}

		// Use global bypass setting
		bypass := m.projectList.bypassPerms
		m.screen = screenModeSelect
		m.modeSelect = newModeSelectModelWithBypass(bypass)
		m.modeSelect.setSize(m.width, m.height)
		return m, m.modeSelect.Init()

	case modeSelectedMsg:
		m.selectedMode = msg.mode
		m.launchConfig.Mode = m.selectedMode
		m.launchConfig.SkipPermissions = msg.skipPermissions
		return m, tea.Quit

	case switchScreenMsg:
		m.screen = msg.screen
		switch msg.screen {
		case screenProjectList:
			m.projectList = newProjectListModel()
			m.projectList.setSize(m.width, m.height)
			return m, m.projectList.Init()
		case screenCreateProject:
			m.createProject = newCreateProjectModel()
			m.createProject.setSize(m.width, m.height)
			return m, m.createProject.Init()
		case screenEditProject:
			m.editProject = newEditProjectModel(msg.projectName)
			m.editProject.setSize(m.width, m.height)
			if msg.addRepo {
				m.editProject.step = editStepMethod
				m.editProject.methodList = m.editProject.buildMethodList()
			}
			return m, m.editProject.Init()
		}
		return m, nil

	case launchMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenProjectList:
		m.projectList, cmd = m.projectList.update(msg)
	case screenCreateProject:
		m.createProject, cmd = m.createProject.update(msg)
	case screenEditProject:
		m.editProject, cmd = m.editProject.update(msg)
	case screenModeSelect:
		m.modeSelect, cmd = m.modeSelect.update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.screen {
	case screenProjectList:
		return m.projectList.view()
	case screenCreateProject:
		return m.createProject.view()
	case screenEditProject:
		return m.editProject.view()
	case screenModeSelect:
		return m.modeSelect.view()
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
