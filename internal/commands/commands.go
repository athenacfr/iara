package commands

func init() {
	Register(Command{
		Name:        "compact-and-continue",
		Description: "Compact the current session context and continue where you left off.",
		CLICommand:  "compact-and-continue",
	})

	Register(Command{
		Name:        "new-session",
		Description: "Close the current session and start a fresh one with updated configuration.",
		CLICommand:  "new-session",
	})

	Register(Command{
		Name:        "reload",
		Description: "Reload the cw session to pick up new commands, rules, and config changes.",
		CLICommand:  "reload",
	})

	Register(Command{
		Name:        "open-project",
		Description: "Open the project root directory in the system editor or file explorer.",
		CLICommand:  "open-project",
	})

	Register(Command{
		Name:        "switch-mode",
		Description: "Switch Claude's behavioral mode mid-session. The session will reload with the new mode's system prompt.",
		CLICommand:  "mode-switch",
		Internal:    true,
		Params: map[string]ParamDef{
			"mode": {
				Type:        "string",
				Description: "The mode to switch to.",
				Enum:        []string{"code", "research", "review", "plan", "tdd", "debug", "free"},
				Required:    true,
			},
		},
	})

	Register(Command{
		Name:        "switch-permissions",
		Description: "Switch Claude's permission mode mid-session. The session will reload with the new permission setting.",
		CLICommand:  "permissions-switch",
		Internal:    true,
		Params: map[string]ParamDef{
			"value": {
				Type:        "string",
				Description: "The permissions mode.",
				Enum:        []string{"bypass", "normal"},
				Required:    true,
			},
		},
	})

	Register(Command{
		Name:        "save-metadata",
		Description: "Save project metadata (title, description, instructions) for the current cw project.",
		CLICommand:  "save-metadata",
		Internal:    true,
		Params: map[string]ParamDef{
			"json": {
				Type:        "string",
				Description: `JSON string with fields: title, description, instructions. Example: {"title":"My Project","description":"...","instructions":"..."}`,
				Required:    true,
			},
		},
	})

	// Prompt-only commands — no CLI handler, just generate .md plugin stubs

	Register(Command{
		Name:        "mode",
		Description: "Switch or show the current behavioral mode. Usage: /mode [code|research|review|plan|tdd|debug|free]",
		PluginBody: `# Mode Management

Switch Claude's behavioral mode mid-session.

## Usage

` + "```" + `
/mode                  # show current mode and available modes
/mode <mode-name>      # switch to a new mode
` + "```" + `

## Argument

The argument is: ` + "`$ARGUMENTS`" + `

## Available Modes

### code (default)
Normal development mode. Full tool access. Write features, fix bugs, refactor code. This is the default behavior.

### research
**READ-ONLY mode.** Do NOT edit or write any files. Only use Read, Grep, Glob, and Bash (read-only commands like ` + "`git log`" + `, ` + "`git diff`" + `, ` + "`ls`" + `, etc). Answer questions about the codebase, explain architecture, trace logic flow, search for patterns. If the user asks you to make changes, remind them to switch to ` + "`code`" + ` mode first.

### review
Code review mode. Focus on reviewing recent changes. Run ` + "`/code-review`" + ` to start. Look for bugs, security issues, code smells, and suggest improvements. Do not make changes unless explicitly asked.

### plan
Planning mode. Focus on understanding requirements and creating implementation plans. Run ` + "`/plan`" + ` to start. Do not write code until the user confirms the plan.

### tdd
Test-driven development mode. Run ` + "`/tdd`" + ` to start. Write tests first, then implement. Follow the RED → GREEN → REFACTOR cycle strictly.

### debug
Investigation mode. Focus on diagnosing issues. Read logs, trace errors, check state, run diagnostic commands. Do not fix anything until the root cause is identified and confirmed with the user.

### free
No preset behavior. Raw Claude with no mode constraints.

## Process

### If no argument (show current mode)

Display:
` + "```" + `
Current mode: <mode from CW_MODE env var, or "code" if not set>

Available modes:
  code       — Write features, fix bugs (default)
  research   — Explore codebase, read-only
  review     — Review code changes
  plan       — Plan before building
  tdd        — Test-driven development
  debug      — Investigate issues
  free       — No preset behavior

Switch with: /mode <name>
` + "```" + `

### If argument provided (switch mode)

1. Validate the mode name is one of: code, research, review, plan, tdd, debug, free
2. If invalid, show the available modes list
3. If valid, run this exact command using the Bash tool:
   ` + "```" + `
   cw internal mode-switch <mode-name>
   ` + "```" + `
   Do NOT do anything else after running the command. The session will restart and resume with the correct mode system prompt injected.

## Important

- Mode switches trigger a session reload so the correct system prompt is injected
- The session resumes automatically using the session ID — no context is lost
- If CW_MODE env var is set (by the cw launcher), that was the initial mode`,
	})

	Register(Command{
		Name:        "permissions",
		Description: "Switch permissions mode. Usage: /permissions [bypass|normal]",
		PluginBody: `# Permissions Management

Switch Claude's permission mode mid-session. This change is **temporary** — it only applies to the current session and does not persist to the project config.

## Usage

` + "```" + `
/permissions            # show current permissions mode
/permissions bypass     # switch to bypass (skip all permission prompts)
/permissions normal     # switch to normal (prompt for permissions)
` + "```" + `

## Argument

The argument is: ` + "`$ARGUMENTS`" + `

## Process

### If no argument (show current)

Check if ` + "`--dangerously-skip-permissions`" + ` is active by looking at the environment. Display:

` + "```" + `
Current permissions: bypass (skipping all permission prompts)

Switch with: /permissions <bypass|normal>
  bypass  — skip all permission prompts
  normal  — prompt for each permission
` + "```" + `

Or if normal:

` + "```" + `
Current permissions: normal (prompting for each permission)

Switch with: /permissions <bypass|normal>
  bypass  — skip all permission prompts
  normal  — prompt for each permission
` + "```" + `

### If argument provided (switch permissions)

1. Validate the value is one of: ` + "`bypass`" + `, ` + "`normal`" + `
2. If invalid, show usage
3. If valid, run this exact command using the Bash tool:
   ` + "```" + `
   cw internal permissions-switch <value>
   ` + "```" + `
   Do NOT do anything else after running the command. The session will restart and resume with the correct permission setting applied.

## Important

- Permission switches trigger a session reload
- The session resumes automatically using the session ID — no context is lost
- This is a **temporary** change for the current session only
- To permanently change the default, use the ` + "`p`" + ` key on the project list screen in the cw TUI`,
	})

	Register(Command{
		Name:        "help",
		Description: "Show all cw commands and modes available inside Claude.",
		PluginBody: `# CW Help

Show the user all available cw commands and current session info.

## Process

Display the following:

` + "```" + `
CW Commands
═══════════

/mode                   Show current mode
/mode <name>            Switch mode (code, research, review, plan, tdd, debug, free)

/permissions            Show current permissions mode
/permissions <value>    Switch permissions (bypass, normal)

/cw:compact-and-continue  Compact context and continue where you left off
/cw:new-session         Start a fresh session
/cw:reload              Reload session (picks up new commands, rules, and config)
/cw:open-project        Open project folder in editor
/cw-help                Show this help

Current Session
═══════════════
Project:   $CW_PROJECT (from env var, or "unknown")
Mode:      $CW_MODE (from env var, or "code")
Directory: (run pwd)
Branch:    (run git branch --show-current)
` + "```" + `

Read the ` + "`CW_PROJECT`" + ` and ` + "`CW_MODE`" + ` environment variables using Bash to populate the current session info. If they're not set, show "unknown" and "code" respectively.`,
	})

	Register(Command{
		Name:        "yolo",
		Description: "Plan and execute work autonomously. Usage: /yolo [objective]",
		PluginBody: `# Yolo Mode — Autonomous Planning & Execution

Plan a set of tasks and then execute them autonomously without human intervention.

## Argument

The argument is: ` + "`$ARGUMENTS`" + `

## Process

### Step 1: Determine Objective

**If argument provided** (e.g., ` + "`/yolo implement auth token refresh`" + `):
- Use the argument as the objective

**If no argument** (` + "`/yolo`" + `):
1. Check for an existing plan file by running: ` + "`ls $CW_PROJECT_DIR/.cw/yolo/plan-*.md 2>/dev/null`" + ` using the Bash tool
   - **If plan exists**: Use AskUserQuestion to ask: "Active yolo plan found. Resume execution or re-plan?"
     - Resume → skip to Step 5 (start executing)
     - Re-plan → continue to Step 2
2. If no plan exists, analyze the recent conversation context
   - **If relevant context exists** → propose a plan based on what's been discussed
   - **If no context** → ask the user what they want to accomplish

### Step 2: Explore & Plan

1. Explore the codebase to understand what's needed for the objective
2. Ask clarifying questions if the objective is ambiguous (keep it brief — 1-2 questions max)
3. Build a plan with atomic, ordered tasks using ` + "`[ ]`" + ` checkboxes

### Step 3: Write Plan File

Create the directory and write the plan:

` + "```bash" + `
mkdir -p "$CW_PROJECT_DIR/.cw/yolo"
` + "```" + `

Then write the plan file to ` + "`$CW_PROJECT_DIR/.cw/yolo/plan-$CW_SESSION_ID.md`" + ` using the Write tool.

Plan format:

` + "```markdown" + `
# Yolo Plan

## Objective
Brief description of what we're building.

## Tasks
- [ ] First task to do
- [ ] Second task
  - [ ] Subtask if needed
- [ ] Third task

## Notes
Any context, decisions, or observations.
` + "```" + `

### Step 4: Confirm

Use AskUserQuestion to show:
- The objective
- The task list (summarized if long)
- Estimated scope (e.g., "~8 tasks, touching 5 files")

Ask: **"Ready to start yolo?"**

- **Yes** → run ` + "`cw internal yolo-start`" + ` using the Bash tool, then continue to Step 5
- **No** → refine the plan based on feedback, update the plan file, and ask again

### Step 5: Execute

Work through all pending tasks in the plan file autonomously.

**Never ask questions.** Do not use AskUserQuestion. Make decisions yourself.
**Never stop to wait for input.** Keep working until all tasks are done.

1. Read the plan file
2. Find the first unchecked ` + "`[ ]`" + ` task
3. Implement it
4. Verify it works (run tests, build, lint as appropriate)
5. Check it off ` + "`[x]`" + ` in the plan file
6. Git commit if you've made meaningful progress
7. Move to the next ` + "`[ ]`" + ` task
8. Repeat until all tasks are ` + "`[x]`" + `

You can add, modify, reorder, or remove tasks as you learn things. Add notes to the Notes section.

**When stuck:** Try a different approach. If you've tried 3 times, skip the task with a note and move on.

**Agents:** Use the Agent tool for parallel or focused work — researcher for exploration, tester for tests, implementer for independent subtasks.

**When ALL tasks are done:** Run ` + "`cw internal yolo-stop`" + ` using the Bash tool. Do NOT call yolo-stop until every task is done.

## Important

- Tasks should be atomic — one clear deliverable per task
- Order tasks by dependency — things that must be done first come first
- Include verification tasks (run tests, build, lint) where appropriate
- The plan file is a living document — modify it during execution`,
	})

	Register(Command{
		Name:        "yolo-start",
		Description: "Start yolo autonomous execution. Writes sideband file and triggers reload.",
		CLICommand:  "yolo-start",
		Internal:    true,
	})

	Register(Command{
		Name:        "yolo-stop",
		Description: "Stop yolo autonomous execution. Deletes plan file and triggers reload.",
		CLICommand:  "yolo-stop",
		Internal:    true,
	})

	Register(Command{
		Name:        "new-task",
		Description: "Create a new task: map codebase, understand intent, create worktree branches, and set up task context.",
		PluginBody: `# New Task

Set up a new task by understanding what the user wants to work on, creating git worktrees with branches, and saving task context.

## Environment

- ` + "`CW_PROJECT_DIR`" + ` — the project root directory
- ` + "`CW_TASK_ID`" + ` — the task ID (set after save-task)

## Process

### Step 1: Map the codebase

Explore all subprojects in the project directory autonomously — do NOT ask the user about tech stack or structure:

- List top-level files and directories in each subproject
- Read package.json, go.mod, Cargo.toml, pyproject.toml, requirements.txt, Makefile, docker-compose.yml, or whatever dependency/config files exist
- Scan a few key source files to understand patterns (naming, formatting, test structure)
- Check for existing linter configs (.eslintrc, .prettierrc, .golangci.yml, etc.)
- Check for CI configs (.github/workflows/, .gitlab-ci.yml, etc.)
- Look at git log for commit message style

### Step 2: Ask what the user wants to work on

Ask: **"What are you working on?"**

This is the only question you ask unprompted. Wait for their answer.

### Step 3: Confirm the intention description

Based on their answer and your codebase understanding, write a clear **description** of the work intention. This should describe what will be done, not summarize what was said.

Present the description and use the **AskUserQuestion** tool to confirm it's correct. If the user wants changes, adjust and confirm again.

### Step 4: Decide and confirm branch names

For each subproject, look at existing branches to detect the naming pattern:

` + "```bash" + `
git -C <subproject> branch -a --format='%(refname:short)' | head -30
` + "```" + `

Common patterns: ` + "`feat/...`" + `, ` + "`feature/...`" + `, ` + "`fix/...`" + `, ` + "`chore/...`" + `, flat names like ` + "`add-auth`" + `. Match whatever the repo already uses. If no clear pattern, use ` + "`feat/<slug>`" + `.

Present the branch name and use **AskUserQuestion** to confirm. All repos will use the same branch name.

### Step 5: Save task and create worktrees

Build a JSON object and run this command using the Bash tool:

` + "```bash" + `
cw internal save-task '{"name":"<slug>","description":"<description>","branch":"<branch-name>"}'
` + "```" + `

This creates the task, sets up git worktrees for each repo, creates the task CLAUDE.md, and symlinks project rules.

### Step 6: Save project metadata (first task only)

Check if project metadata already exists:

` + "```bash" + `
cat "$CW_PROJECT_DIR/.cw/metadata.json" 2>/dev/null
` + "```" + `

If the file doesn't exist or is empty, this is the first task. Build a JSON object with technical context and save it:

` + "```bash" + `
cw internal save-metadata '{"title":"<title>","description":"<description>","instructions":"<technical-context>"}'
` + "```" + `

The instructions field should contain: tech stack, build/test commands, conventions, coding patterns — everything you learned from mapping the codebase. Write it as direct instructions to Claude.

If metadata already exists, skip this step.

### Step 7: Finish

Check how this command was invoked:

` + "```bash" + `
echo $CW_AUTO_SETUP
` + "```" + `

- If ` + "`1`" + `: Say "All set! Starting fresh session..." then run ` + "`cw internal new-session`" + `
- Otherwise: Say "All set! Reloading session..." then run ` + "`cw internal reload`" + `

## Important

- Derive everything technical from the subprojects. Only ask the user what they're working on.
- Do NOT mention internal files (task.json, metadata.json) to the user.
- Do NOT create any files directly — use the CLI commands.
- All repos share the same branch name for a task.
- Worktrees are created by the save-task command — do NOT run git worktree commands yourself.`,
	})

	Register(Command{
		Name:        "finish-task",
		Description: "Complete the current task: verify clean state, remove worktrees, and return to task selection.",
		PluginBody: `# Finish Task

Complete the current task by cleaning up worktrees and marking it as done.

## Environment

- ` + "`CW_PROJECT_DIR`" + ` — the project root directory
- ` + "`CW_TASK_ID`" + ` — the current task ID
- ` + "`CW_TASK_NAME`" + ` — the current task name

## Process

### Step 1: Check for uncommitted changes

Check all repos in the worktree for dirty state:

` + "```bash" + `
for dir in "$CW_PROJECT_DIR"/.worktrees/"$CW_TASK_NAME"/*/; do
  if [ -d "$dir/.git" ] || [ -f "$dir/.git" ]; then
    echo "=== $(basename "$dir") ==="
    git -C "$dir" status --porcelain
  fi
done
` + "```" + `

If any repo has uncommitted changes, warn the user and ask if they want to:
1. Commit the changes first
2. Discard changes and proceed
3. Cancel

### Step 2: Confirm

Use **AskUserQuestion**: "Ready to finish task '<task-name>'? The branch will be preserved but the worktree will be removed."

### Step 3: Finish the task

` + "```bash" + `
cw internal finish-task
` + "```" + `

This removes the worktrees and marks the task as completed.

### Step 4: Return to task selection

` + "```bash" + `
cw internal reload
` + "```" + `

## Important

- Always check for uncommitted changes before finishing.
- The git branch is preserved in the original repos — only the worktree working copy is removed.
- The user can push/merge the branch via normal git/gh workflow before or after finishing.`,
	})

	Register(Command{
		Name:        "setup-project",
		Description: "Map the codebase and save project metadata for a new cw project.",
		PluginBody: `# Setup Project

Set up project metadata by mapping the codebase. This runs automatically on the first launch of a new project.

## Process

### Step 1: Map the codebase

Explore all subprojects in the project directory autonomously — do NOT ask the user about tech stack or structure:

- List top-level files and directories in each subproject
- Read package.json, go.mod, Cargo.toml, pyproject.toml, requirements.txt, Makefile, docker-compose.yml, or whatever dependency/config files exist
- Scan a few key source files to understand patterns (naming, formatting, test structure)
- Check for existing linter configs (.eslintrc, .prettierrc, .golangci.yml, etc.)
- Check for CI configs (.github/workflows/, .gitlab-ci.yml, etc.)
- Look at git log for commit message style

### Step 2: Ask what the project is about

Ask: **"What is this project?"**

Wait for their answer.

### Step 3: Save metadata

Build a JSON object and run:

` + "```bash" + `
cw internal save-metadata '{"title":"<title>","description":"<description>","instructions":"<technical-context>"}'
` + "```" + `

Fields:
- **title**: Short project title
- **description**: What the project is about
- **instructions**: Technical context — structure, tech stack, conventions, build/test commands, coding patterns. Write as direct instructions to Claude.

### Step 4: Finish

Check:

` + "```bash" + `
echo $CW_AUTO_SETUP
` + "```" + `

- If ` + "`1`" + `: Say "Project set up! Starting fresh session..." then run ` + "`cw internal new-session`" + `
- Otherwise: Say "Project set up! Reloading..." then run ` + "`cw internal reload`" + `

## Important

- Derive everything technical from the subprojects. Only ask what the project is about.
- Do NOT mention internal files to the user.
- Do NOT create any files directly — use the CLI command.
- Keep instructions concise but complete.`,
	})

	Register(Command{
		Name:        "save-task",
		Description: "Save task metadata and create worktrees for a new task.",
		CLICommand:  "save-task",
		Internal:    true,
		Params: map[string]ParamDef{
			"json": {
				Type:        "string",
				Description: `JSON string with fields: name, description, branch. Example: {"name":"add-auth","description":"...","branch":"feat/add-auth"}`,
				Required:    true,
			},
		},
	})

	Register(Command{
		Name:        "complete-task",
		Description: "Mark task as completed and remove its worktrees.",
		CLICommand:  "finish-task",
		Internal:    true,
	})

	Register(Command{
		Name:        "dev",
		Description: "Run development commands (dev servers, watchers, type generators) in the background. Auto-discovers commands on first run. Usage: /dev [stop|restart|status|update|logs]",
		PluginBody: `# Dev — Background Development Commands

Run development commands (dev servers, build watchers, type generators) in the background for all subprojects.

## Argument

The argument is: ` + "`$ARGUMENTS`" + `

## Config File

Dev commands are persisted at ` + "`$CW_TASK_DIR/dev-config.json`" + `:

` + "```json" + `
{
  "subprojects": [
    {
      "path": "frontend",
      "port": 5173,
      "commands": [
        {
          "cmd": "npm run dev",
          "description": "Vite dev server with HMR",
          "type": "long-running"
        },
        {
          "cmd": "npm run generate:types",
          "description": "Generate GraphQL types from schema",
          "type": "one-shot"
        }
      ]
    },
    {
      "path": "backend",
      "venv": ".venv",
      "port": 8000,
      "commands": [
        {
          "cmd": "uvicorn main:app --reload",
          "description": "FastAPI dev server",
          "type": "long-running"
        },
        {
          "cmd": "alembic upgrade head",
          "description": "Run database migrations",
          "type": "one-shot"
        }
      ]
    }
  ]
}
` + "```" + `

**Command types:**
- ` + "`one-shot`" + ` — runs once and completes (codegen, migrations, builds). Executed first, sequentially.
- ` + "`long-running`" + ` — runs continuously (dev servers, watchers, file watchers). Launched in parallel as background tasks.

**Optional fields per subproject:**
- ` + "`venv`" + ` (string) — path to a Python virtual environment relative to the subproject root (e.g. ` + "`.venv`" + `). When set, all commands for this subproject are prefixed with ` + "`source <venv>/bin/activate &&`" + `.
- ` + "`port`" + ` (number) — the port this subproject's dev server listens on. Used for port conflict detection and env override updates.

**Top-level optional field:**
- ` + "`portBase`" + ` (number) — the base port for this project's port range. Each subproject gets ports offset from this base. This prevents port conflicts between different cw projects.

## Port Allocation

Each cw project gets a deterministic port range to avoid conflicts when multiple projects run simultaneously.

**Scheme:**
- During first discovery, assign a ` + "`portBase`" + ` derived from a hash of the project name, mapped to a range (e.g. 3000-9999). Example: project "myapp" → portBase 4200.
- Each subproject gets the next port in sequence: first subproject uses ` + "`portBase`" + `, second uses ` + "`portBase + 1`" + `, etc.
- The ` + "`port`" + ` field on each subproject stores its assigned port.
- Commands that need a port (dev servers) should be launched with the assigned port via flag (e.g. ` + "`--port 4200`" + `) or env var.

**Config example with portBase:**
` + "```json" + `
{
  "portBase": 4200,
  "subprojects": [
    { "path": "frontend", "port": 4200, ... },
    { "path": "backend",  "port": 4201, "venv": ".venv", ... }
  ]
}
` + "```" + `

**During discovery**, after identifying subproject commands:
1. Compute a ` + "`portBase`" + ` from the project name (hash mod 5000 + 3000, giving range 3000-7999)
2. Assign sequential ports to subprojects that have long-running dev servers
3. Modify the discovered commands to use the assigned port (e.g. append ` + "`--port <N>`" + ` or set ` + "`PORT=<N>`" + ` prefix)
4. Show the port assignments in the discovery confirmation so the user can adjust

**Port flag conventions by stack:**
- Node.js/Vite: ` + "`--port <N>`" + ` or ` + "`PORT=<N>`" + `
- Python/uvicorn: ` + "`--port <N>`" + `
- Python/Django: ` + "`0.0.0.0:<N>`" + ` as positional arg to runserver
- Go/air/custom: ` + "`PORT=<N>`" + ` env var prefix
- Cargo: ` + "`PORT=<N>`" + ` env var prefix

## Process

### /dev (no argument or first run)

#### If config exists — launch

1. Read ` + "`$CW_TASK_DIR/dev-config.json`" + `
2. **Port conflict check**: For each subproject with a ` + "`port`" + ` field, check if that port is already in use:
   ` + "```bash" + `
   lsof -i :<port> -sTCP:LISTEN -t 2>/dev/null
   ` + "```" + `
   If a port is occupied, warn the user and suggest an alternative port. If the user picks a different port, update the command accordingly (e.g. append ` + "`--port <new-port>`" + `) and update the config file.
3. **Env override sync**: If any port was changed from the config default, check the env override files at ` + "`$CW_PROJECT_DIR/.env.<repo>.override`" + ` for variables that reference the old port (e.g. ` + "`API_URL`" + `, ` + "`BACKEND_URL`" + `, ` + "`VITE_API_URL`" + `, ` + "`PORT`" + `, ` + "`NEXT_PUBLIC_API_URL`" + `). Update them to the new port so other subprojects connect to the right address.
5. For each subproject, run one-shot commands first (sequentially, wait for each to complete). **Redirect output to log files**:
   - If the subproject has a ` + "`venv`" + ` field:
     ` + "```bash" + `
     cd <project-dir>/<subproject-path> && source <venv>/bin/activate && <one-shot-cmd> >> "$CW_TASK_DIR/logs/<subproject>.log" 2>&1
     ` + "```" + `
   - Otherwise:
     ` + "```bash" + `
     cd <project-dir>/<subproject-path> && <one-shot-cmd> >> "$CW_TASK_DIR/logs/<subproject>.log" 2>&1
     ` + "```" + `
   If a one-shot command fails (non-zero exit), read the last 20 lines of its log file to show the error, then ask if the user wants to continue or abort.
6. Then launch all long-running commands in parallel using ` + "`run_in_background: true`" + `. **Redirect all output to log files** so it doesn't accumulate in Claude's memory:
   - If the subproject has a ` + "`venv`" + ` field:
     ` + "```bash" + `
     cd <project-dir>/<subproject-path> && source <venv>/bin/activate && <long-running-cmd> >> "$CW_TASK_DIR/logs/<subproject>.log" 2>&1
     ` + "```" + `
   - Otherwise:
     ` + "```bash" + `
     cd <project-dir>/<subproject-path> && <long-running-cmd> >> "$CW_TASK_DIR/logs/<subproject>.log" 2>&1
     ` + "```" + `
7. Display a summary table:
   ` + "```" + `
   Dev commands running:

   Subproject   Command              Type          Port   Status
   ─────────────────────────────────────────────────────────────
   frontend     npm run dev          long-running  :5173  ✓ background
   backend      uvicorn main:app     long-running  :8000  ✓ background (venv)
   backend      alembic upgrade head one-shot       —     ✓ completed (venv)

   URLs:
     frontend  → http://localhost:5173
     backend   → http://localhost:8000

   Logs: .cw/logs/frontend.log, .cw/logs/backend.log
   Use /dev logs to view output, /dev status to check health.
   ` + "```" + `

8. After displaying the table, show a **URLs section** listing each subproject that has a port with its URL as ` + "`http://localhost:<port>`" + `. Only include subprojects with long-running commands that have a port assigned.

#### If NO config exists — discover and confirm

1. List all subdirectories in the project root (these are subprojects)
2. For each subproject, look for:
   - ` + "`package.json`" + ` → check ` + "`scripts`" + ` for dev, start, watch, generate, build:watch, codegen, typecheck entries
   - ` + "`Makefile`" + ` → check for dev, watch, serve, run, generate targets
   - ` + "`Cargo.toml`" + ` → cargo watch, cargo run
   - ` + "`go.mod`" + ` → check Makefile or common go run/air/templ patterns
   - ` + "`pyproject.toml`" + ` / ` + "`manage.py`" + ` → check for runserver, celery, uvicorn patterns. Also check if ` + "`.venv/`" + ` or ` + "`venv/`" + ` exists — if so, set ` + "`venv`" + ` field in the config so commands are activated properly.
   - ` + "`docker-compose.yml`" + ` → check for dev services
   - ` + "`Procfile`" + ` / ` + "`Procfile.dev`" + ` → dev process definitions
3. For each discovered command, classify as ` + "`one-shot`" + ` or ` + "`long-running`" + `:
   - **long-running**: dev, start, watch, serve, runserver (anything that keeps running)
   - **one-shot**: generate, codegen, build, typecheck, migrate (anything that completes)
4. Present the discovered config to the user using **AskUserQuestion**:
   ` + "```" + `
   Discovered dev commands:

   frontend/ (Node.js) — port 5173
     - npm run dev          → Vite dev server [long-running]
     - npm run generate     → Generate GraphQL types [one-shot]

   backend/ (Python, venv: .venv) — port 8000
     - uvicorn main:app --reload  → FastAPI dev server [long-running]
     - alembic upgrade head       → Run migrations [one-shot]

   Does this look right? You can:
   - Confirm to save and start
   - Add/remove/modify commands
   - Change ports or venv paths
   - Skip a subproject
   ` + "```" + `
5. If user confirms, write the config to ` + "`$CW_TASK_DIR/dev-config.json`" + ` and launch (go to "If config exists" flow)
6. If user wants changes, adjust and confirm again

### /dev stop

1. Stop all running background dev tasks using the TaskStop tool
2. Confirm: "All dev commands stopped." (Log files persist at ` + "`.cw/logs/`" + ` — cw cleans them up automatically when the session ends.)

### /dev restart

1. Stop all running background dev tasks (same as /dev stop)
2. Re-launch everything from config (same as /dev launch flow)
3. Confirm: "Dev commands restarted."

### /dev update

Re-discover and merge changes into the existing config without losing manual edits.

1. Read the existing config from ` + "`$CW_TASK_DIR/dev-config.json`" + `
2. Run the full discovery process (same as "If NO config exists" above)
3. Diff the discovered config against the existing config and present changes using **AskUserQuestion**:
   ` + "```" + `
   Config update — changes detected:

   NEW subprojects:
     + worker/ (Python, venv: .venv) — port 4202
       - celery -A app worker  → Celery worker [long-running]

   CHANGED subprojects:
     ~ frontend/ — 1 new command found
       + npm run typecheck      → TypeScript check [one-shot]

   REMOVED subprojects (no longer detected):
     - legacy-api/  (still in config — keep or remove?)

   UNCHANGED:
     = backend/ — no changes

   Accept changes? You can:
   - Accept all
   - Accept selectively
   - Edit before saving
   ` + "```" + `
4. Merge accepted changes into the existing config, preserving:
   - Manual edits to commands, descriptions, and types
   - Custom ` + "`venv`" + ` paths and ` + "`port`" + ` assignments
   - The existing ` + "`portBase`" + ` (assign new subprojects the next available port in sequence)
5. Write updated config to ` + "`$CW_TASK_DIR/dev-config.json`" + `
6. If dev commands are currently running, ask: "Restart with updated config?"

### /dev status

1. Check if background tasks are still running using TaskOutput with ` + "`block: false`" + ` (non-blocking check — just get status, don't wait)
2. Display status for each:
   ` + "```" + `
   Dev command status:

   Subproject   Command          Port   Status
   ──────────────────────────────────────────────
   frontend     npm run dev      :5173  running
   backend      uvicorn main:app :8000  exited (error)
   ` + "```" + `
3. For any failed or errored tasks, show the **last 10 lines** of the log file (not full output):
   ` + "```bash" + `
   tail -n 10 "$CW_TASK_DIR/logs/<subproject>.log"
   ` + "```" + `
4. Ask if the user wants to restart failed commands
5. Mention: "Use ` + "`/dev logs <subproject>`" + ` for more output."

### /dev logs [subproject] [lines]

Read dev server logs in controlled chunks to minimize token usage.

1. If no subproject specified, show the **last 50 lines** of each subproject's log file
2. If subproject specified, show the **last 50 lines** of that subproject's log
3. If lines specified, override the default (e.g. ` + "`/dev logs backend 200`" + `)
4. Read logs using:
   ` + "```bash" + `
   tail -n <lines> "$CW_TASK_DIR/logs/<subproject>.log"
   ` + "```" + `
Note: Log file size is managed automatically by cw (truncated to last 5000 lines if exceeding 10MB). No need to handle this in the LLM.

## Error Handling

- When a background task completes unexpectedly (crash), you'll get an automatic notification. Read the **last 20 lines** of the log file to surface the error — do NOT use TaskOutput for the full output. Ask if the user wants to restart it.
- If a one-shot command fails during startup, show the error and ask whether to continue with remaining commands or abort.
- If the config file is malformed, show the error and offer to re-discover.

## Important

- Always ` + "`cd`" + ` to the subproject directory before running commands — never run from project root
- Use absolute paths when constructing the ` + "`cd`" + ` path: ` + "`$CW_PROJECT_DIR/<subproject-path>`" + `
- One-shot commands run sequentially and must complete before long-running commands start
- Long-running commands all run in parallel as background tasks
- The config file is the source of truth — always read it before launching
- If the user modifies the config manually, respect those changes
- When discovering commands, read actual file contents (package.json scripts, Makefile targets) — don't guess
- **Python venv**: During discovery, check for ` + "`.venv/`" + ` or ` + "`venv/`" + ` directories. If found, set the ` + "`venv`" + ` field in config. Always activate the venv before running any Python subproject command — without it, commands will use the system Python and fail to find project dependencies.
- **Port awareness**: During discovery, infer default ports from config files and command flags (e.g. ` + "`--port 8000`" + `, Vite's default 5173, Django's default 8000). Store in the ` + "`port`" + ` field. Before launching, check for port conflicts — if multiple cw sessions or external processes occupy a port, choose an alternative and update env override files so cross-service references stay correct.
- **Env override sync**: When a port changes, scan ` + "`$CW_PROJECT_DIR/.env.<repo>.override`" + ` files for URL or port variables referencing the old port and update them. This ensures that e.g. a frontend's ` + "`VITE_API_URL`" + ` points to the backend's actual running port.
- **Log management**: All dev process output goes to ` + "`$CW_TASK_DIR/logs/<subproject>.log`" + `. NEVER use TaskOutput to read full process output — always read log files with ` + "`tail`" + ` to control token usage. Log cleanup (deletion on session end, truncation of oversized files) is handled automatically by cw — do NOT run cleanup commands yourself.`,
	})
}
