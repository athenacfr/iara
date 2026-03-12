# cw

A multi-repo project manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Organize multiple git repositories into projects, pick a mode, and launch Claude with the right context.

## Features

- **Multi-repo projects** — group related repos into a single workspace
- **Modes** — launch Claude in different behavioral modes (code, research, review, plan, tdd, debug, free)
- **Fuzzy search** — fzf-style navigation for projects, repos, and modes
- **GitHub integration** — clone repos directly from GitHub via `gh` CLI
- **Auto-context** — injects project structure and rules into Claude's system prompt
- **Permissions toggle** — choose whether to bypass Claude Code's permission prompts
- **Git worktrees** — manage worktrees from inside Claude sessions
- **Background pulls** — repos are pulled in parallel on project launch

## Install

Requires Go 1.24+ and [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed.

```sh
git clone https://github.com/ahtwr/cw.git
cd cw
make install
```

This installs the `cw` binary to `~/.local/bin/cw`.

## Usage

```sh
cw
```

This opens the TUI. From there:

1. **Create a project** — press `n`, give it a name, and add repos (from GitHub, git URL, local path, or init empty)
2. **Select a project** — navigate with `j`/`k`, press `enter`
3. **Pick a mode** — choose how Claude should behave, toggle permissions with `tab`, press `enter`
4. Claude launches in your project directory with full context

### Modes

| Mode | Description |
|------|-------------|
| `code` | Write features, fix bugs (default) |
| `research` | Explore codebase, read-only |
| `review` | Review code changes |
| `plan` | Plan before building |
| `tdd` | Test-driven development |
| `debug` | Investigate issues |
| `free` | No preset behavior |

### In-session commands

Once inside Claude, these slash commands are available:

```
/mode [name]              Show or switch mode
/cw [list|create|remove]  Manage git worktrees
/cw:help                  Show all commands
```

## Project structure

Each project lives under `~/.local/share/cw/projects/<name>/` and contains:

- One or more git repos as subdirectories
- A `CLAUDE.md` with project-wide instructions (auto-generated on first launch)

## Configuration

| Env var | Description |
|---------|-------------|
| `CW_PROJECTS_DIR` | Override the default projects directory |

## Uninstall

```sh
cw uninstall
```

Removes the binary and all data from `~/.local/share/cw/`.
