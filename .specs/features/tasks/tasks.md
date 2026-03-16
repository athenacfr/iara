# Tasks Feature — Task Breakdown

## Dependency Graph

```
T1 (git ops) ──┐
               ├── T3 (task package) ── T5 (task screen) ── T7 (app.go wiring)
T2 (session)  ─┘                    └── T6 (launcher changes)─┘
                                                                └── T8 (main.go launch)
T4 (commands) ─── depends on T3 ─── T9 (new-task skill)
                                 └── T10 (finish-task skill)
T11 (migration) ── depends on T3
T12 (setup-project) ── depends on T4
T13 (cleanup) ── depends on all
```

## Tasks

### T1: Git worktree operations
**Files:** `internal/git/git.go`, `internal/git/git_test.go`
**What:**
- Add `WorktreeAdd(repoPath, worktreePath, branchName string) error`
- Add `WorktreeRemove(repoPath, worktreePath string) error`
- Add `DefaultBranch(repoPath string) string` — check `refs/remotes/origin/HEAD`, fall back to detecting main/master
**Verify:** `go test ./internal/git/...` — new tests for worktree add/remove/default branch

---

### T2: Session package — parametric sessionsDir
**Files:** `internal/session/session.go`, `internal/session/session_test.go`
**What:**
- Change `Save(projectDir, s)` → `Save(sessionsDir, s)`
- Change `List(projectDir)` → `List(sessionsDir)`
- Change `Load(projectDir, id)` → `Load(sessionsDir, id)`
- Change `Touch(projectDir)` → `Touch(sessionsDir)`
- Remove internal `sessionsDir()` path computation — caller provides full path
- Update `GenerateSummary` to take sessionsDir
**Verify:** `go test ./internal/session/...` — update existing tests to pass sessionsDir

---

### T3: Task package (new)
**Files:** `internal/task/task.go`, `internal/task/worktree.go`, `internal/task/task_test.go`
**What:**
- Task struct: ID, Name, Description, Branch, Status, CreatedAt, LastActive
- `New(name, description, branch) Task`
- `Save(projectDir, t) error` — writes `.cw/tasks/<id>/task.json`
- `Load(projectDir, taskID) (Task, error)`
- `List(projectDir) ([]Task, error)` — reads all task.json files
- `Touch(projectDir, taskID) error` — updates LastActive
- `SetStatus(projectDir, taskID, status) error`
- `SessionsDir(projectDir, taskID) string` — returns `.cw/tasks/<id>/sessions/`
- `DefaultSessionsDir(projectDir) string` — returns `.cw/tasks/default/sessions/`
- `WorktreeBase(projectDir, taskSlug) string` — returns `.worktrees/<slug>/`
- `SetupWorktree(projectDir string, t Task, repoNames []string) error`:
  - `git worktree add` for each repo (uses git.WorktreeAdd)
  - Create `.worktrees/<slug>/.claude/rules/`
  - Symlink `PROJECT.md` → project CLAUDE.md
  - Write initial `CLAUDE.md` with task description
- `RemoveWorktree(projectDir string, t Task, repoNames []string) error`:
  - `git worktree remove` for each repo
  - Remove `.worktrees/<slug>/` directory
**Depends on:** T1
**Verify:** `go test ./internal/task/...`

---

### T4: Command registry — new-task, finish-task, setup-project
**Files:** `internal/commands/commands.go`, `internal/commands/commands_test.go`
**What:**
- Add `new-task` command (plugin body: full skill prompt)
- Add `finish-task` command (plugin body: full skill prompt)
- Add `setup-project` command (simplified new-intention: metadata only, no branching)
- Remove `new-intention` command
**Depends on:** T3 (needs internal CLI commands defined)
**Verify:** `go test ./internal/commands/...`

---

### T5: Task selection screen (new)
**Files:** `internal/tui/screen/task_select.go`
**What:**
- New `TaskSelectModel` using `widget.FzfList`
- Entries: "+ New Task", "● <default-branch>", active tasks
- Load tasks from `task.List(projectDir)`
- Detect default branch via `git.DefaultBranch()`
- Preview panel: task description, branch, session count, last active
- On confirm: emit `TaskSelectedMsg`
- On cancel: emit `NavigateMsg{ScreenProjectExplorer}`
- Keybar: `↑↓ select  enter confirm  esc back`
**Depends on:** T3
**Verify:** Build compiles, manual TUI testing

---

### T6: Launcher screen changes
**Files:** `internal/tui/screen/launcher.go`
**What:**
- Constructor takes `sessionsDir` instead of `projectDir`
- `LoadSessions` uses `sessionsDir` directly
- Add task name to header display
- `esc` emits `NavigateMsg{ScreenTaskSelect}` instead of `ScreenProjectExplorer`
**Depends on:** T2
**Verify:** `go test ./...`, build compiles

---

### T7: App.go wiring
**Files:** `internal/tui/app.go`, `internal/tui/shared/messages.go`
**What:**
- Add `ScreenTaskSelect` to screen enum
- Add `TaskSelectedMsg` message type (Task, SessionsDir, WorkDir, IsDefault)
- Add `taskSelect screen.TaskSelectModel` to Model
- `ProjectSelectedMsg` handler: navigate to `ScreenTaskSelect` (not launcher)
- `TaskSelectedMsg` handler:
  - If new task: set prompt `/cw:new-task`, quit
  - If default + no metadata: set prompt `/cw:setup-project`, quit
  - If default: navigate to launcher with default sessionsDir
  - If existing task: pull worktree repos, navigate to launcher with task sessionsDir
- `NavigateMsg` handler: add `ScreenTaskSelect` case
- Store selected task in Model for LaunchConfig building
**Depends on:** T5, T6
**Verify:** Build compiles, manual TUI flow testing

---

### T8: main.go launch loop changes
**Files:** `main.go`
**What:**
- Read `TaskID` and `SessionsDir` from LaunchConfig
- Set `CW_TASK_ID` and `CW_TASK_NAME` env vars
- `env.Sync` uses worktree base path when task is active, project path for default
- Session save/load/touch uses `SessionsDir` from LaunchConfig
- `save-task` internal command: parse JSON, call `task.Save` + `task.SetupWorktree`
- `finish-task` internal command: call `task.RemoveWorktree` + `task.SetStatus`
- Git pull on worktree repos for task selection
**Depends on:** T3, T7
**Verify:** `go test ./...`, `make build`, manual e2e

---

### T9: `/cw:new-task` skill
**Files:** `internal/commands/commands.go` (plugin body)
**What:**
- Adapt from `/cw:new-intention` plugin body
- Map codebase (autonomous)
- Ask intent, confirm description
- Propose branch name, confirm
- Call `cw internal save-task '<json>'` (creates task + worktrees + CLAUDE.md)
- Save project metadata if first task
- Call `cw internal new-session` to reload into worktree
**Depends on:** T4, T8
**Verify:** Manual e2e — create a new task, verify worktree, CLAUDE.md, sessions

---

### T10: `/cw:finish-task` skill
**Files:** `internal/commands/commands.go` (plugin body)
**What:**
- Check dirty files across worktree repos
- Confirm with user
- Call `cw internal finish-task`
- Signal reload to return to task selection
**Depends on:** T4, T8
**Verify:** Manual e2e — finish a task, verify worktree removed, branch persists

---

### T11: Session migration
**Files:** `internal/task/migrate.go`
**What:**
- `MigrateSessionsIfNeeded(projectDir string) error`
- If `.cw/sessions/` exists and `.cw/tasks/default/sessions/` does not:
  - Move `.cw/sessions/` → `.cw/tasks/default/sessions/`
- Called once on task list load
**Depends on:** T3
**Verify:** Unit test — create legacy sessions, verify migration

---

### T12: Setup-project skill (replaces new-intention for first-time)
**Files:** `internal/commands/commands.go` (plugin body)
**What:**
- Simplified version of new-intention
- Maps codebase, asks intent
- Saves project metadata only (no branching, no worktree)
- Reloads to TUI (user then creates tasks from task list)
**Depends on:** T4
**Verify:** Manual e2e — new project, verify metadata saved, returns to task list

---

### T13: Dev command scoping
**Files:** `internal/devlog/devlog.go`, `internal/devlog/devlog_test.go`, `internal/commands/commands.go` (dev command body)
**What:**
- `devlog.EnsureDir`, `devlog.Cleanup`, `devlog.TruncateOversized` take a base dir instead of projectDir
- Dev config path (`dev-config.json`) resolved from task-scoped base dir
- `/dev` skill reads config from task-scoped path (via env var or path convention)
- main.go passes task-scoped base dir to devlog functions
**Depends on:** T3, T8
**Verify:** `go test ./internal/devlog/...`, manual test — different dev configs per task

---

### T14: Cleanup — remove new-intention
**Files:** `internal/commands/commands.go`, `internal/tui/app.go`
**What:**
- Remove `new-intention` command from registry
- Remove auto-launch `/cw:new-intention` from app.go
- Replace with `/cw:setup-project` auto-launch for projects without metadata
- Verify embed cleanup removes stale `new-intention.md` plugin
**Depends on:** T12
**Verify:** `go test ./...`, `make build`, verify old plugin file removed

---

## Suggested Implementation Order

**Phase 1 — Foundation (can be parallel):**
- T1: Git worktree operations
- T2: Session parametric sessionsDir

**Phase 2 — Core:**
- T3: Task package (depends on T1)
- T4: Command registry (depends on T3)

**Phase 3 — TUI:**
- T5: Task selection screen (depends on T3)
- T6: Launcher changes (depends on T2)
- T7: App.go wiring (depends on T5, T6)

**Phase 4 — Launch loop:**
- T8: main.go changes (depends on T3, T7)

**Phase 5 — Skills:**
- T9: new-task skill (depends on T4, T8)
- T10: finish-task skill (depends on T4, T8)
- T12: setup-project skill (depends on T4)

**Phase 6 — Polish:**
- T11: Session migration (depends on T3)
- T13: Dev command scoping (depends on T3, T8)
- T14: Cleanup new-intention (depends on T12)

**Phase 7 — Testing:**
- Update e2e TUI tests for new task screen flow
- Update snapshots
