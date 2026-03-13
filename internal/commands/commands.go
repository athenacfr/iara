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
   Do NOT do anything else after running the command. The session will restart with ` + "`--continue`" + ` and the correct mode system prompt injected.

## Important

- Mode switches trigger a session reload so the correct system prompt is injected
- The session resumes automatically with ` + "`--continue`" + ` — no context is lost
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
   Do NOT do anything else after running the command. The session will restart with ` + "`--continue`" + ` and the correct permission setting applied.

## Important

- Permission switches trigger a session reload
- The session resumes automatically with ` + "`--continue`" + ` — no context is lost
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

/cw                     List worktrees for current project
/cw create <branch>     Create new worktree
/cw remove <branch>     Remove a worktree

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
		Name:        "worktree",
		Description: "Manage git worktrees for the current project. Usage: /cw [list|create|remove] [branch]",
		PluginBody: `# Worktree Management

Manage git worktrees for the current project without leaving Claude.

## Usage

` + "```" + `
/cw                    # list all worktrees
/cw list               # same as above
/cw create <branch>    # create new worktree for branch
/cw remove <branch>    # remove a worktree
` + "```" + `

## Process

### Parse the argument

The argument is: ` + "`$ARGUMENTS`" + `

- If empty or "list": run the **list** action
- If starts with "create": run the **create** action with the branch name
- If starts with "remove": run the **remove** action with the branch name

### List Action

Run ` + "`git worktree list`" + ` and display results in a formatted table:

` + "```" + `
| Path | Branch | Status |
` + "```" + `

For each worktree, show whether it's the main worktree or a linked one. Also show if there are uncommitted changes (` + "`git -C <path> status --porcelain`" + `).

### Create Action

Given a branch name:

1. Determine the worktree path: ` + "`../<current-project-name>-<branch>`" + ` (sibling directory)
2. Check if that path already exists — if so, report and stop
3. Check if the branch exists locally: ` + "`git show-ref --verify refs/heads/<branch>`" + `
4. Check if the branch exists remotely: ` + "`git show-ref --verify refs/remotes/origin/<branch>`" + `
5. Create the worktree:
   - Branch exists locally or remotely: ` + "`git worktree add <path> <branch>`" + `
   - Branch doesn't exist: ` + "`git worktree add -b <branch> <path>`" + `
6. Report success with the full path
7. Tell the user: "To work in this worktree, exit and run: ` + "`cw <project> -w <branch>`" + `"

### Remove Action

Given a branch name:

1. Find the worktree path from ` + "`git worktree list`" + ` matching the branch
2. If not found, report and stop
3. Check for uncommitted changes in the worktree — warn if dirty
4. Ask for confirmation before removing
5. Run ` + "`git worktree remove <path>`" + `
6. Report success

## Important Rules

- NEVER run ` + "`git checkout`" + ` to switch branches — worktrees exist to avoid that
- NEVER remove the main worktree
- Always show the full path when creating/listing worktrees
- If the user asks to "switch" to a worktree, tell them to exit and use ` + "`cw <project> -w <branch>`" + ``,
	})

	Register(Command{
		Name:        "new-intention",
		Description: "Map the codebase, understand intent, create branches, and set up project context.",
		PluginBody: `# New Intention

Set up a cw project by understanding what the user wants to work on, creating branches, and saving project context.

## Process

### Step 1: Map the codebase

Explore all repos in the project directory autonomously — do NOT ask the user about tech stack or structure:

- List top-level files and directories in each repo
- Read package.json, go.mod, Cargo.toml, pyproject.toml, requirements.txt, Makefile, docker-compose.yml, or whatever dependency/config files exist
- Scan a few key source files to understand patterns (naming, formatting, test structure)
- Check for existing linter configs (.eslintrc, .prettierrc, .golangci.yml, etc.)
- Check for CI configs (.github/workflows/, .gitlab-ci.yml, etc.)
- Look at git log for commit message style

### Step 2: Ask what the user wants to work on

Ask: **"What are you working on?"**

This is the only question you ask unprompted. Wait for their answer.

### Step 3: Confirm the intention description

Based on their answer and your codebase understanding, write a clear **description** (not a summary) of the work intention. This should describe what will be done, not summarize what was said.

Present the description and use the **AskUserQuestion** tool to confirm it's correct. If the user wants changes, adjust and confirm again.

### Step 4: Decide and confirm branch names

For each repo, look at existing branches to detect the naming pattern:

` + "```bash" + `
git -C <repo> branch -a --format='%(refname:short)' | head -30
` + "```" + `

Common patterns: ` + "`feat/...`" + `, ` + "`feature/...`" + `, ` + "`fix/...`" + `, ` + "`chore/...`" + `, flat names like ` + "`add-auth`" + `. Match whatever the repo already uses. If no clear pattern, use ` + "`feat/<slug>`" + `.

Present the branch plan as a numbered list with the last option always being to commit directly to the current branch (usually main):
- **Single repo**: "1. ` + "`<branch-name>`" + ` 2. Commit directly to ` + "`<current-branch>`" + `"
- **Multi repo**: "1. ` + "`<branch-1>`" + ` / ` + "`<branch-2>`" + ` 2. Commit directly to current branches"

Use **AskUserQuestion** to confirm. If the user wants changes, adjust and confirm again.

### Step 5: Create branches

If the user chose to commit directly to the current branch, skip branch creation entirely.

Otherwise, for each repo, create and checkout the branch:

` + "```bash" + `
git -C <repo> checkout -b <branch-name>
` + "```" + `

### Step 6: Save metadata

Build a JSON object with three fields and run this command using the Bash tool:

` + "```bash" + `
cw internal save-metadata '<json-string>'
` + "```" + `

The JSON string should have these fields:
- **title**: A short project title (derived from the repo or user's description)
- **description**: The confirmed intention description from Step 3
- **instructions**: Technical context and conventions for this project — structure, tech stack, conventions, build/test commands, coding patterns. This will be injected as a system prompt in future sessions, so write it as direct instructions to Claude (e.g., "This project uses Go 1.24 with Bubble Tea for TUI..." not "The project uses..."). Include everything you learned from mapping the codebase.

Make sure to properly escape any quotes, newlines, or special characters in the JSON string. Use ` + "`\\n`" + ` for newlines within field values.

### Step 7: Finish

Check how this command was invoked by running:

` + "```bash" + `
echo $CW_AUTO_SETUP
` + "```" + `

- If the output is ` + "`1`" + ` (cw auto-invoked this during project setup): Say "All set! Starting fresh session..." then run ` + "`cw internal new-session`" + ` using the Bash tool.
- Otherwise (user manually invoked this): Say "All set! Reloading session..." then run ` + "`cw internal reload`" + ` using the Bash tool.

## Important

- Derive everything technical from the repos. Only ask the user what they're working on.
- Do NOT mention internal files (metadata.json) to the user.
- Do NOT write .claude/CLAUDE.md — everything goes into metadata.
- Do NOT create any files — metadata is saved via the CLI command.
- Keep instructions concise but complete — they replace CLAUDE.md as the project context.`,
	})
}
