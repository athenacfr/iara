# Tasks Feature — Design

## Component Architecture

```
┌─────────────────────────────────────────────────────┐
│ internal/task/                   NEW PACKAGE         │
│   task.go        — Task struct, CRUD, List           │
│   worktree.go    — worktree setup (CLAUDE.md, dirs)  │
├─────────────────────────────────────────────────────┤
│ internal/git/                    MODIFIED            │
│   git.go         — +WorktreeAdd, WorktreeRemove      │
│                    +DefaultBranch                     │
├─────────────────────────────────────────────────────┤
│ internal/session/                MODIFIED            │
│   session.go     — change projectDir → sessionsDir   │
├─────────────────────────────────────────────────────┤
│ internal/tui/screen/             MODIFIED + NEW      │
│   task_select.go — NEW task selection screen          │
│   launcher.go    — receives sessionsDir, esc → tasks  │
├─────────────────────────────────────────────────────┤
│ internal/tui/                    MODIFIED            │
│   app.go         — new screen enum, task routing      │
│   shared/messages.go — TaskSelectedMsg                │
├─────────────────────────────────────────────────────┤
│ internal/commands/               MODIFIED            │
│   commands.go    — new-task, finish-task, remove      │
│                    new-intention, add setup-project    │
├─────────────────────────────────────────────────────┤
│ internal/claude/                 MODIFIED            │
│   claude.go      — TaskID, TaskName, SessionsDir      │
│                    in LaunchConfig                    │
├─────────────────────────────────────────────────────┤
│ main.go                          MODIFIED            │
│   — save-task, finish-task internal commands          │
│   — session save/load uses SessionsDir               │
│   — env.Sync uses worktree base when in task          │
│   — git pull on task select                          │
└─────────────────────────────────────────────────────┘
```

## Data Flow

### New Task Creation

```
TUI: "+ New Task" selected
  → LaunchConfig.Prompt = "/cw:new-task"
  → LaunchConfig.WorkDir = <project-root> (temporary, skill creates worktree)
  → Claude runs /cw:new-task skill:
      1. Maps codebase (reads from main repos)
      2. Asks intent, confirms description
      3. Proposes branch name, confirms
      4. Calls: cw internal save-task '{"name":"add-auth","description":"...","branch":"feat/add-auth"}'
         → internal/task/ creates task.json in .cw/tasks/<id>/
         → internal/task/ creates worktrees:
            - git worktree add .worktrees/<slug>/<repo> -b <branch> (for each repo)
            - mkdir .worktrees/<slug>/.claude/rules/
            - symlink .worktrees/<slug>/.claude/rules/PROJECT.md → project CLAUDE.md
            - writes .worktrees/<slug>/CLAUDE.md (task instructions)
      5. Calls: cw internal new-session (reloads into worktree context)
  → cw re-launches Claude with:
      WorkDir = .worktrees/<slug>/
      SessionsDir = .cw/tasks/<id>/sessions/
      CW_TASK_ID, CW_TASK_NAME set
```

### Existing Task Selection

```
TUI: task selected → TaskSelectedMsg
  → git pull on worktree repos (background)
  → navigate to launcher screen with sessionsDir
  → user picks mode + session → launch Claude
  → WorkDir = .worktrees/<slug>/
  → env.Sync(worktreeBase, repoNames)
```

### Default Branch Selection

```
TUI: "● main" selected → TaskSelectedMsg (default=true)
  → navigate to launcher screen with sessionsDir = .cw/tasks/default/sessions/
  → user picks mode + session → launch Claude
  → WorkDir = <project-root> (same as today)
  → env.Sync(projectDir, repoNames) (same as today)
```

### Finish Task

```
Claude: user runs /cw:finish-task
  → skill checks dirty files
  → confirms with user
  → calls: cw internal finish-task
    → git worktree remove for each repo
    → removes .worktrees/<slug>/ directory
    → marks task status = "completed" in task.json
  → signals sideband reload → cw returns to TUI task select
```

## New Package: internal/task/

```go
package task

type Task struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Branch      string `json:"branch"`
    Status      string `json:"status"`
    CreatedAt   string `json:"created_at"`
    LastActive  string `json:"last_active"`
}

// CRUD
func New(name, description, branch string) Task
func Save(projectDir string, t Task) error
func Load(projectDir, taskID string) (Task, error)
func List(projectDir string) ([]Task, error)
func Touch(projectDir, taskID string) error
func SetStatus(projectDir, taskID, status string) error

// Worktree setup
func SetupWorktree(projectDir string, t Task, repoNames []string) error
func RemoveWorktree(projectDir string, t Task, repoNames []string) error
func WorktreeBase(projectDir, taskSlug string) string
func SessionsDir(projectDir, taskID string) string
func DefaultSessionsDir(projectDir string) string
```

`SetupWorktree` handles:
1. `git worktree add` for each repo
2. Create `.claude/rules/` directory
3. Symlink `PROJECT.md` → project CLAUDE.md
4. Write initial task `CLAUDE.md`

## New Git Operations

```go
// internal/git/

func WorktreeAdd(repoPath, worktreePath, branchName string) error
// runs: git -C <repoPath> worktree add <worktreePath> -b <branchName>

func WorktreeRemove(repoPath, worktreePath string) error
// runs: git -C <repoPath> worktree remove <worktreePath>

func DefaultBranch(repoPath string) string
// checks: refs/remotes/origin/HEAD, falls back to main/master detection
```

## TUI Changes

### New Screen: screenTaskSelect

```go
// shared/messages.go
type TaskSelectedMsg struct {
    Task        *task.Task  // nil for default branch
    SessionsDir string      // computed sessions path
    WorkDir     string      // worktree base or project root
    IsDefault   bool        // true for default branch entry
}

// Screen enum addition
const (
    // ...existing...
    ScreenTaskSelect
)
```

### Navigation Changes

```
ProjectSelectedMsg:
  Before: → screenLauncher (or auto-launch new-intention)
  After:  → screenTaskSelect (always)

TaskSelectedMsg (default=false, new task):
  → quit, launch Claude with /cw:new-task

TaskSelectedMsg (default=false, existing task):
  → screenLauncher with sessionsDir from task

TaskSelectedMsg (default=true):
  → if no metadata: quit, launch setup skill
  → if has metadata: screenLauncher with default sessionsDir
```

### Launcher Changes

- Constructor takes `sessionsDir` parameter
- `esc` emits `NavigateMsg{Screen: ScreenTaskSelect}` (not ScreenProjectExplorer)
- Header shows task name

## Session Package Changes

Minimal — just change path computation:

```go
// Before: func sessionsDir(projectDir string) string {
//     return filepath.Join(projectDir, ".cw", "sessions")
// }

// After: sessionsDir is passed by caller. Remove internal path computation.
// Save(dir, s), List(dir), Load(dir, id) all take dir directly.
```

## LaunchConfig Changes

```go
type LaunchConfig struct {
    // ...existing...
    TaskID      string // CW_TASK_ID env var
    TaskName    string // CW_TASK_NAME env var
    SessionsDir string // path to task's sessions directory
}
```

## Migration Strategy

On first task list load for a project:
1. Check if `.cw/sessions/` exists and `.cw/tasks/default/sessions/` does not
2. If so, move `.cw/sessions/` → `.cw/tasks/default/sessions/`
3. One-time, transparent to user

## Git Pull Timing

- **Project selection**: pull main repos (background) — same as today
- **Task selection (worktree)**: pull worktree repos (background)
- **Task selection (default)**: no extra pull needed (already pulled at project select)
