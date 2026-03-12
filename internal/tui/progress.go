package tui

import (
	"fmt"
	"strings"

	"github.com/ahtwr/cw/internal/git"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type cloneTickMsg git.CloneProgress
type cloneAllDoneMsg struct{}

type repoStatus struct {
	repo string
	done bool
	err  error
}

type progressModel struct {
	title   string
	repos   []repoStatus
	spinner spinner.Model
	done    bool
	width   int
	height  int
}

func newProgressModel(title string, repoNames []string) progressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	repos := make([]repoStatus, len(repoNames))
	for i, r := range repoNames {
		repos[i] = repoStatus{repo: r}
	}

	return progressModel{
		title:   title,
		repos:   repos,
		spinner: s,
	}
}

func (m *progressModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressModel) update(msg tea.Msg) (progressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case cloneTickMsg:
		for i := range m.repos {
			if m.repos[i].repo == msg.Repo && msg.Done {
				m.repos[i].done = true
				m.repos[i].err = msg.Err
				break
			}
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m progressModel) view() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", titleStyle.Render(m.title)))

	doneCount := 0
	for _, r := range m.repos {
		prefix := m.spinner.View()
		status := dimStyle.Render("cloning...")

		if r.done {
			doneCount++
			if r.err != nil {
				prefix = errorStyle.Render("✗")
				status = errorStyle.Render(r.err.Error())
			} else {
				prefix = successStyle.Render("✓")
				status = successStyle.Render("done")
			}
		}

		sb.WriteString(fmt.Sprintf("  %s %s  %s\n", prefix, r.repo, status))
	}

	sb.WriteString(fmt.Sprintf("\n  %s\n",
		dimStyle.Render(fmt.Sprintf("%d/%d complete", doneCount, len(m.repos))),
	))

	return sb.String()
}
