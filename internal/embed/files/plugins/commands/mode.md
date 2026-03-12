---
description: Switch or show the current behavioral mode. Usage: /mode [code|research|review|plan|tdd|debug|free]
---

# Mode Management

Switch Claude's behavioral mode mid-session.

## Usage

```
/mode                  # show current mode and available modes
/mode <mode-name>      # switch to a new mode
```

## Argument

The argument is: `$ARGUMENTS`

## Available Modes

### code (default)
Normal development mode. Full tool access. Write features, fix bugs, refactor code. This is the default behavior.

### research
**READ-ONLY mode.** Do NOT edit or write any files. Only use Read, Grep, Glob, and Bash (read-only commands like `git log`, `git diff`, `ls`, etc). Answer questions about the codebase, explain architecture, trace logic flow, search for patterns. If the user asks you to make changes, remind them to switch to `code` mode first.

### review
Code review mode. Focus on reviewing recent changes. Run `/code-review` to start. Look for bugs, security issues, code smells, and suggest improvements. Do not make changes unless explicitly asked.

### plan
Planning mode. Focus on understanding requirements and creating implementation plans. Run `/plan` to start. Do not write code until the user confirms the plan.

### tdd
Test-driven development mode. Run `/tdd` to start. Write tests first, then implement. Follow the RED → GREEN → REFACTOR cycle strictly.

### debug
Investigation mode. Focus on diagnosing issues. Read logs, trace errors, check state, run diagnostic commands. Do not fix anything until the root cause is identified and confirmed with the user.

### free
No preset behavior. Raw Claude with no mode constraints.

## Process

### If no argument (show current mode)

Display:
```
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
```

### If argument provided (switch mode)

1. Validate the mode name is one of: code, research, review, plan, tdd, debug, free
2. If invalid, show the available modes list
3. If valid, acknowledge the switch and immediately adopt the new behavior:
   - For `research`: remind yourself you are now READ-ONLY
   - For `review`: run `/code-review`
   - For `plan`: run `/plan`
   - For `tdd`: run `/tdd`
   - For `code`/`free`: acknowledge and continue normally
   - For `debug`: state that you'll focus on diagnosis before fixes

Display:
```
Switched to <mode> mode.
<one-line description of what this means>
```

## Important

- Mode switches are immediate — adopt the new behavior right away
- The `research` mode is strict: absolutely no file edits, no Write, no Edit tools
- Modes persist for the rest of the session unless switched again
- If CW_MODE env var is set (by the cw launcher), that was the initial mode
