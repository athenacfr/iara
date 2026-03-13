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
3. If valid, trigger a reload with the new mode's system prompt:
   - Display: `Switching to <mode> mode... reloading session.`
   - Run: `cw internal mode-switch <mode-name>`
   - Do NOT do anything else after running the command. The session will restart with `--continue` and the correct mode system prompt injected.

## Important

- Mode switches trigger a session reload so the correct system prompt is injected
- The session resumes automatically with `--continue` — no context is lost
- If CW_MODE env var is set (by the cw launcher), that was the initial mode
