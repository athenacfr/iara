package screen

import (
	"fmt"
	"os"
	"strings"

	"github.com/ahtwr/cw/internal/git"
	"github.com/ahtwr/cw/internal/session"
	"github.com/ahtwr/cw/internal/task"
	"github.com/ahtwr/cw/internal/tui/shared"
	"github.com/ahtwr/cw/internal/tui/style"
	"github.com/ahtwr/cw/internal/tui/widget"
	tea "github.com/charmbracelet/bubbletea"
)

type taskItem struct {
	entry taskEntry
}

func (t taskItem) FilterValue() string { return t.entry.label }

type taskEntry struct {
	label   string
	relTime string
	task    *task.Task // nil for "new task" and "default branch" entries
	kind    int        // 0=new task, 1=default branch, 2=existing task
}

type tasksLoadedMsg struct {
	tasks []task.Task
	err   error
}

type defaultBranchMsg struct {
	branch string
}

// TaskSelectModel shows the task selection screen.
type TaskSelectModel struct {
	fzfList       widget.FzfListModel
	projectDir    string
	projectName   string
	defaultBranch string
	tasks         []task.Task
	width, height int
}

// NewTaskSelectModel creates a new task selection screen.
func NewTaskSelectModel(projectDir, projectName string) TaskSelectModel {
	entries := []taskEntry{
		{label: "+ New Task", kind: 0},
		{label: "● default branch", kind: 1},
	}

	items := make([]widget.FzfItem, len(entries))
	for i, e := range entries {
		items[i] = taskItem{entry: e}
	}

	fzf := widget.NewFzfList(items, widget.FzfListConfig{
		RenderItem:   renderTaskItem,
		PreviewFunc:  taskPreview,
		Placeholder:  "No tasks",
		ListWidthPct: 0.5,
	})

	return TaskSelectModel{
		fzfList:       fzf,
		projectDir:    projectDir,
		projectName:   projectName,
		defaultBranch: "default branch",
	}
}

// LoadTasks returns a command that loads tasks from the project directory.
func (m TaskSelectModel) LoadTasks(projectDir string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := task.List(projectDir)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

// DetectDefaultBranch returns a command that detects the default branch.
func (m TaskSelectModel) DetectDefaultBranch() tea.Cmd {
	return func() tea.Msg {
		// Find the first repo in projectDir (a subdirectory with .git)
		entries, err := os.ReadDir(m.projectDir)
		if err != nil {
			return defaultBranchMsg{branch: "main"}
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			repoPath := m.projectDir + "/" + e.Name()
			if git.IsRepo(repoPath) {
				branch := git.DefaultBranch(repoPath)
				return defaultBranchMsg{branch: branch}
			}
		}
		return defaultBranchMsg{branch: "main"}
	}
}

// Init returns the initial commands to load tasks and detect the default branch.
func (m TaskSelectModel) Init() tea.Cmd {
	// One-time migration of legacy sessions
	task.MigrateSessionsIfNeeded(m.projectDir)
	return tea.Batch(m.LoadTasks(m.projectDir), m.DetectDefaultBranch())
}

// SetSize sets the available size for the screen.
func (m *TaskSelectModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.fzfList.SetSize(w, h-5)
}

func (m TaskSelectModel) buildItems() []widget.FzfItem {
	entries := []taskEntry{
		{label: "+ New Task", kind: 0},
		{label: "● " + m.defaultBranch, kind: 1},
	}
	for i := range m.tasks {
		t := m.tasks[i]
		relTime := session.RelativeTime(session.ParseTime(t.LastActive))
		entries = append(entries, taskEntry{
			label:   t.Name,
			relTime: relTime,
			task:    &t,
			kind:    2,
		})
	}
	items := make([]widget.FzfItem, len(entries))
	for i, e := range entries {
		items[i] = taskItem{entry: e}
	}
	return items
}

// Update handles messages for the task selection screen.
func (m TaskSelectModel) Update(msg tea.Msg) (TaskSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.tasks = msg.tasks
		m.fzfList.SetItems(m.buildItems())
		return m, nil

	case defaultBranchMsg:
		m.defaultBranch = msg.branch
		m.fzfList.SetItems(m.buildItems())
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		newList, consumed, result := m.fzfList.HandleKey(key)
		m.fzfList = newList

		if result != nil {
			switch r := result.(type) {
			case widget.FzfConfirmMsg:
				ti := r.Item.(taskItem)
				switch ti.entry.kind {
				case 0: // new task
					return m, func() tea.Msg {
						return shared.TaskSelectedMsg{
							IsNew:   true,
							WorkDir: m.projectDir,
						}
					}
				case 1: // default branch
					return m, func() tea.Msg {
						return shared.TaskSelectedMsg{
							IsDefault:   true,
							SessionsDir: task.DefaultSessionsDir(m.projectDir),
							WorkDir:     m.projectDir,
						}
					}
				case 2: // existing task
					t := ti.entry.task
					return m, func() tea.Msg {
						return shared.TaskSelectedMsg{
							Task:        t,
							SessionsDir: task.SessionsDir(m.projectDir, t.ID),
							WorkDir:     task.WorktreeBase(m.projectDir, t.Name),
						}
					}
				}
			case widget.FzfCancelMsg:
				_ = r
				return m, func() tea.Msg {
					return shared.NavigateMsg{Screen: shared.ScreenProjectExplorer}
				}
			}
		}

		if consumed {
			return m, nil
		}
	}
	return m, nil
}

// View renders the task selection screen.
func (m TaskSelectModel) View() string {
	var b strings.Builder

	b.WriteString(style.TitleStyle.Render("TASKS"))
	b.WriteString("\n")
	b.WriteString(m.fzfList.View())
	b.WriteString("\n")

	keybar := style.RenderKeybar(
		style.KeyBind{Key: "↑↓", Desc: "select"},
		style.KeyBind{Key: "enter", Desc: "confirm"},
		style.KeyBind{Key: "esc", Desc: "back"},
	)
	b.WriteString(keybar)

	return b.String()
}

func renderTaskItem(item widget.FzfItem, displayNum int, cursor, selected bool, matched []int, width int) string {
	ti := item.(taskItem)

	prefix := "  "
	if cursor {
		prefix = style.FzfCursorPrefix.Render("▶ ")
	}

	numStr := style.KeyStyle.Render(fmt.Sprintf("%d.", displayNum)) + " "

	switch ti.entry.kind {
	case 0:
		return prefix + numStr + style.TreeAddStyle.Render(ti.entry.label)
	case 1:
		return prefix + numStr + style.AccentStyle.Render(ti.entry.label)
	}

	// kind=2: existing task
	plain := ti.entry.label
	timeSuffix := "  " + style.DimStyle.Render(ti.entry.relTime)
	timePlainLen := 2 + len(ti.entry.relTime)

	numWidth := len(fmt.Sprintf("%d.", displayNum)) + 1
	prefixWidth := 2 + numWidth
	contentWidth := width - prefixWidth

	labelWidth := contentWidth - timePlainLen
	if labelWidth <= 0 || contentWidth <= 0 {
		return prefix + numStr + widget.HighlightMatches(plain, matched) + timeSuffix
	}

	if len(plain) <= labelWidth {
		return prefix + numStr + widget.HighlightMatches(plain, matched) + timeSuffix
	}

	truncated := widget.Truncate(widget.HighlightMatches(plain, matched), labelWidth)
	return prefix + numStr + truncated + timeSuffix
}

func taskPreview(item widget.FzfItem, width, height int) string {
	ti := item.(taskItem)

	var lines []string

	switch ti.entry.kind {
	case 0:
		lines = append(lines, style.TitleStyle.Render("New Task"))
		lines = append(lines, "")
		lines = append(lines, style.DimStyle.Render("Create a new task with its own"))
		lines = append(lines, style.DimStyle.Render("branch and worktree."))
	case 1:
		lines = append(lines, style.TitleStyle.Render("Default Branch"))
		lines = append(lines, "")
		lines = append(lines, style.DimStyle.Render("Work directly on the original"))
		lines = append(lines, style.DimStyle.Render("repos without a worktree."))
	case 2:
		if ti.entry.task != nil {
			t := ti.entry.task
			lines = append(lines, style.TitleStyle.Render("Task"))
			lines = append(lines, "")
			if t.Description != "" {
				lines = append(lines, t.Description)
				lines = append(lines, "")
			}
			lines = append(lines, style.AccentStyle.Render("branch: ")+t.Branch)
			lines = append(lines, style.AccentStyle.Render("status: ")+t.Status)
			lines = append(lines, "")
			lines = append(lines, style.DimStyle.Render(session.RelativeTime(session.ParseTime(t.LastActive))))
		}
	}

	return strings.Join(lines, "\n")
}
