---
description: Manage git worktrees for the current project. Usage: /cw [list|create|remove] [branch]
---

# Worktree Management

Manage git worktrees for the current project without leaving Claude.

## Usage

```
/cw                    # list all worktrees
/cw list               # same as above
/cw create <branch>    # create new worktree for branch
/cw remove <branch>    # remove a worktree
```

## Process

### Parse the argument

The argument is: `$ARGUMENTS`

- If empty or "list": run the **list** action
- If starts with "create": run the **create** action with the branch name
- If starts with "remove": run the **remove** action with the branch name

### List Action

Run `git worktree list` and display results in a formatted table:

```
| Path | Branch | Status |
```

For each worktree, show whether it's the main worktree or a linked one. Also show if there are uncommitted changes (`git -C <path> status --porcelain`).

### Create Action

Given a branch name:

1. Determine the worktree path: `../<current-project-name>-<branch>` (sibling directory)
2. Check if that path already exists — if so, report and stop
3. Check if the branch exists locally: `git show-ref --verify refs/heads/<branch>`
4. Check if the branch exists remotely: `git show-ref --verify refs/remotes/origin/<branch>`
5. Create the worktree:
   - Branch exists locally or remotely: `git worktree add <path> <branch>`
   - Branch doesn't exist: `git worktree add -b <branch> <path>`
6. Report success with the full path
7. Tell the user: "To work in this worktree, exit and run: `cw <project> -w <branch>`"

### Remove Action

Given a branch name:

1. Find the worktree path from `git worktree list` matching the branch
2. If not found, report and stop
3. Check for uncommitted changes in the worktree — warn if dirty
4. Ask for confirmation before removing
5. Run `git worktree remove <path>`
6. Report success

## Important Rules

- NEVER run `git checkout` to switch branches — worktrees exist to avoid that
- NEVER remove the main worktree
- Always show the full path when creating/listing worktrees
- If the user asks to "switch" to a worktree, tell them to exit and use `cw <project> -w <branch>`
