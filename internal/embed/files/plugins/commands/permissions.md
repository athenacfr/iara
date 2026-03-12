---
description: Switch permissions mode. Usage: /permissions [bypass|normal]
---

# Permissions Management

Switch Claude's permission mode mid-session. This change is **temporary** — it only applies to the current session and does not persist to the project config.

## Usage

```
/permissions            # show current permissions mode
/permissions bypass     # switch to bypass (skip all permission prompts)
/permissions normal     # switch to normal (prompt for permissions)
```

## Argument

The argument is: `$ARGUMENTS`

## Process

### If no argument (show current)

Check if `--dangerously-skip-permissions` is active by looking at the environment. Display:

```
Current permissions: bypass (skipping all permission prompts)

Switch with: /permissions <bypass|normal>
  bypass  — skip all permission prompts
  normal  — prompt for each permission
```

Or if normal:

```
Current permissions: normal (prompting for each permission)

Switch with: /permissions <bypass|normal>
  bypass  — skip all permission prompts
  normal  — prompt for each permission
```

### If argument provided (switch permissions)

1. Validate the value is one of: `bypass`, `normal`
2. If invalid, show usage
3. If valid, trigger a reload:
   - Display: `Switching to <value> permissions... reloading session.`
   - Run: `cw internal permissions-switch <value>`
   - Do NOT do anything else after running the command. The session will restart with `--continue` and the correct permission setting applied.

## Important

- Permission switches trigger a session reload
- The session resumes automatically with `--continue` — no context is lost
- This is a **temporary** change for the current session only
- To permanently change the default, use the `p` key on the project list screen in the cw TUI
