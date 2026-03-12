---
description: Show all cw commands and modes available inside Claude.
---

# CW Help

Show the user all available cw commands and current session info.

## Process

Display the following:

```
CW Commands
═══════════

/cw                     List worktrees for current project
/cw create <branch>     Create new worktree
/cw remove <branch>     Remove a worktree

/mode                   Show current mode
/mode <name>            Switch mode (code, research, review, plan, tdd, debug, free)

/cw-help                Show this help

Current Session
═══════════════
Project:   $CW_PROJECT (from env var, or "unknown")
Mode:      $CW_MODE (from env var, or "code")
Directory: (run pwd)
Branch:    (run git branch --show-current)
```

Read the `CW_PROJECT` and `CW_MODE` environment variables using Bash to populate the current session info. If they're not set, show "unknown" and "code" respectively.
