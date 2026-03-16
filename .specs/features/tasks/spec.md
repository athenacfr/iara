# Tasks Feature

## Summary

Introduce a "task" concept between project selection and session selection. A task represents a unit of work (feature, bugfix, etc.) that operates in isolated git worktrees. Sessions live under tasks.

## Current Flow

```
Select Project → (auto /new-intention if no metadata) → Select Mode + Session → Launch Claude
```

## New Flow

```
Select Project → Select/Create Task → Select Mode + Session → Launch Claude
```

## Requirements

### R1: Task Data Model

A task represents a unit of work within a project.

```go
type Task struct {
    ID          string    // UUID
    Name        string    // short name / slug (used for branch naming)
    Description string    // what the user is working on
    Branch      string    // git branch name (e.g., feat/add-auth)
    Status      string    // "active", "completed"
    CreatedAt   string    // RFC3339
    LastActive  string    // RFC3339
}
```

### R2: Storage Layout

Tasks and their sessions stored under `.cw/tasks/`:

```
<project>/.cw/
  metadata.json              ← project-level metadata (unchanged)
  tasks/
    <task-id>/
      task.json              ← task metadata
      sessions/
        <session-id>.json    ← sessions scoped to this task
```

- Sessions move from `.cw/sessions/` to `.cw/tasks/<task-id>/sessions/`
- A "default" task exists for backward compatibility (sessions with no task)

### R3: Task Selection Screen

New TUI screen between project selection and launcher (mode+session):

```
TASKS
  ▶ 1. + New Task
    2. ● default branch               ← original repos, no worktree
    3. add-user-auth                   2h ago
    4. fix-payment-bug                 1d ago

  ──────────────────────────
  ↑↓ select   enter confirm   esc back
```

- **"default branch"** entry always present — shows actual default branch name detected from the repos (e.g. `● main`, `● master`, `● develop`). Launches Claude in original repos (no worktree). Sessions stored in `.cw/tasks/default/sessions/`. Also serves as backward compat with existing `.cw/sessions/`.
- "+ New Task" at top — launches Claude with `/cw:new-task`
- Active tasks sorted by `LastActive` desc
- Preview panel shows task description, branch, session count, last active
- `esc` goes back to project explorer
- `enter` on existing task → proceed to launcher (mode+session screen)
- `enter` on "+ New Task" → launch Claude with `/cw:new-task`

### R4: Git Worktree Integration

Each task operates in isolated git worktrees under `.worktrees/` at the project root.

**Layout:**
```
<project>/
  frontend/                          ← main checkout (untouched)
  backend/                           ← main checkout (untouched)
  .worktrees/
    <task-slug>/                     ← one folder per task
      frontend/                      ← worktree on feat/add-auth
      backend/                       ← worktree on feat/add-auth
      .env.frontend.global           ← symlink (created by env.Sync)
      .env.frontend.override         ← override file (created by env.Sync)
      .env.backend.global            ← symlink
      .env.backend.override          ← override file
  .cw/
    tasks/<task-id>/
      task.json
      sessions/
```

**How it works:**
- **Create**: `git worktree add <worktree-path> -b <branch-name>` for each repo
- **Worktree location**: `<project>/.worktrees/<task-slug>/<repo-name>/`
- **Claude WorkDir**: `<project>/.worktrees/<task-slug>/`
- **--add-dir**: Points to worktree copies of each repo
- **Finish**: `git worktree remove` cleans up; branch persists in the repo

**CLAUDE.md layering:**
```
.worktrees/<task-slug>/
  CLAUDE.md              ← task-specific (created by /new-task)
  .claude/
    rules/
      PROJECT.md         → ../../../CLAUDE.md (symlink to project rules)
  frontend/              ← worktree (has its own .claude/CLAUDE.md)
  backend/               ← worktree
```
- Task instructions: `CLAUDE.md` at worktree root — intent, approach, decisions
- Project rules: `.claude/rules/PROJECT.md` — symlink, always up to date
- Repo-level: each repo's own `.claude/CLAUDE.md` (already exists in worktree)
- Claude reads all three layers natively

**Env sync**: `env.Sync(worktreeBasePath, repoNames)` works unchanged — it writes `.env` to `<worktreeBase>/<repoName>/.env`, creates symlinks to globals, and creates override files at the worktree base. Each task gets its own env overrides naturally.

**Why `.worktrees/` at project root (not inside `.cw/`):**
- Env sync expects `projectDir/<repoName>/` structure — worktree base mirrors project root
- Override files (`.env.<repo>.override`) live alongside repo dirs — works naturally
- Symlinks to globals land in the right place
- Claude's `--add-dir` paths look like normal repo paths
- Clean separation: `.cw/` = metadata, `.worktrees/` = working copies

New git operations needed in `internal/git/`:
```go
func WorktreeAdd(repoPath, worktreePath, branchName string) error
func WorktreeRemove(worktreePath string) error
func WorktreeList(repoPath string) ([]Worktree, error)
```

### R5: `/cw:new-task` Skill (replaces `/cw:new-intention`)

**Only available when launched in a worktree context** (not from default branch).

Adaptation of `/cw:new-intention` for task creation:

1. **Map Codebase** (autonomous) — same as new-intention
2. **Ask Intent** — "What are you working on?"
3. **Confirm Description** — summarize, confirm
4. **Branch Name** — detect pattern, propose name, confirm
5. **Create Worktrees** — `git worktree add` for each repo (NOT `checkout -b`)
6. **Save Task** — `cw internal save-task '<json>'` (new internal command)
7. **Save/Update Metadata** — update project instructions if first task
8. **Finish** — start new session within the task

Key differences from `/cw:new-intention`:
- Creates worktrees instead of branches
- Saves task metadata instead of just project metadata
- Project metadata (instructions) is saved on first task only, or updated if needed

### R6: `/cw:finish-task` Skill

**Only available when launched in a worktree context** (not from default branch).

Completes a task:

1. **Check Status** — ensure all repos are clean (no uncommitted changes)
2. **Confirm** — "Ready to finish task '<name>'?"
3. **Remove Worktrees** — `git worktree remove` for each repo
4. **Update Task** — set status to "completed"
5. **Signal** — trigger sideband reload to return to task selection

The branch stays in the repo — the user pushes/merges via normal git/gh workflow.

### R7: First-Time Project Flow

When a project has no metadata (new project):
- **Current**: auto-launch `/cw:new-intention`
- **New**: still auto-setup on first launch, but via default branch (not a task). The setup skill maps the codebase and saves project metadata — same as today but without branch creation. User then creates tasks from the task list when ready.
- `/cw:new-intention` is deleted; project setup becomes a simpler skill that only handles metadata (no branching)

### R8: Session Path Routing

Session functions change from `projectDir` to `sessionsDir`:

```go
// Before:
session.Save(projectDir, s)     // hardcoded .cw/sessions/
session.List(projectDir)
session.Load(projectDir, id)

// After:
session.Save(sessionsDir, s)    // caller provides full path
session.List(sessionsDir)
session.Load(sessionsDir, id)
```

The caller computes the sessions directory based on task context:
- Default branch: `<project>/.cw/tasks/default/sessions/`
- Worktree task: `<project>/.cw/tasks/<task-id>/sessions/`

This keeps the session package dumb — no knowledge of tasks. The TUI and main.go know which task is active and pass the right path.

### R9: Launcher Screen Changes

The launcher (mode+session) screen now scopes sessions to the selected task:
- Receives `sessionsDir` from the task selection screen
- Task name shown in the header
- `esc` goes back to task selection (not project explorer)

### R10: Dev Command Scoping

Dev config and logs scope to the active task:

```
.cw/tasks/<task-id>/
  task.json
  sessions/
  dev-config.json        ← task-scoped dev config
  logs/                  ← task-scoped dev logs
```

- Each task discovers and manages its own dev servers independently
- Different tasks can run on different ports without conflicts
- Default branch gets its own dev config at `.cw/tasks/default/dev-config.json`
- `devlog` package functions take a base dir (same pattern as sessions)

### R11: LaunchConfig Changes

```go
type LaunchConfig struct {
    // ... existing fields ...
    TaskID      string   // active task ID
    TaskName    string   // for display / env var
    WorkDir     string   // now points to worktree path (not main checkout)
}
```

New env vars passed to Claude:
- `CW_TASK_ID` — task UUID
- `CW_TASK_NAME` — task name/slug

### R11: Internal CLI Commands

New commands in `main.go`:
- `cw internal save-task '<json>'` — save task metadata to disk
- `cw internal finish-task` — mark task completed, remove worktrees

### R12: Migration / Backward Compatibility

- The **"default branch"** entry in the task list serves as backward compat
- Existing sessions in `.cw/sessions/` are shown under the "default branch" entry
- On first use, existing `.cw/sessions/` are migrated to `.cw/tasks/default/sessions/` (or symlinked)
- No data loss — old sessions just appear under the default entry

## Open Questions

1. **Single-repo projects**: Should we still use worktrees for projects with one repo? (Proposal: yes, for consistency)
2. **Task deletion**: Should tasks be deletable from the TUI? (Proposal: not in v1, just complete them)
3. **Re-opening completed tasks**: Allow resuming a completed task? (Proposal: yes, change status back to active)

## Non-Goals

- Task dependencies or ordering
- Multi-user task assignment
- Integration with external task trackers (Jira, Linear)
- Automatic PR creation on finish (user does this manually)
