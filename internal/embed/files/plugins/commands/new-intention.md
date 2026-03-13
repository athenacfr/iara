---
description: Map the codebase, understand intent, create branches, and set up project context.
---

# New Intention

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

```bash
git -C <repo> branch -a --format='%(refname:short)' | head -30
```

Common patterns: `feat/...`, `feature/...`, `fix/...`, `chore/...`, flat names like `add-auth`. Match whatever the repo already uses. If no clear pattern, use `feat/<slug>`.

Present the branch plan as a numbered list with the last option always being to commit directly to the current branch (usually main):
- **Single repo**: "1. `<branch-name>` 2. Commit directly to `<current-branch>`"
- **Multi repo**: "1. `<branch-1>` / `<branch-2>` 2. Commit directly to current branches"

Use **AskUserQuestion** to confirm. If the user wants changes, adjust and confirm again.

### Step 5: Create branches

If the user chose to commit directly to the current branch, skip branch creation entirely.

Otherwise, for each repo, create and checkout the branch:

```bash
git -C <repo> checkout -b <branch-name>
```

### Step 6: Save metadata

Build a JSON object with three fields and save it via the cw command:

- **title**: A short project title (derived from the repo or user's description)
- **description**: The confirmed intention description from Step 3
- **instructions**: Technical context and conventions for this project — structure, tech stack, conventions, build/test commands, coding patterns. This will be injected as a system prompt in future sessions, so write it as direct instructions to Claude (e.g., "This project uses Go 1.24 with Bubble Tea for TUI..." not "The project uses..."). Include everything you learned from mapping the codebase.

```bash
cw internal save-metadata '{"title": "...", "description": "...", "instructions": "..."}'
```

Make sure to properly escape any quotes, newlines, or special characters in the JSON string. Use `\n` for newlines within field values.

### Step 7: Finish

Check how this command was invoked by running:

```bash
echo $CW_AUTO_SETUP
```

- If the output is `1` (cw auto-invoked this during project setup): Say "All set! Starting fresh session..." then run `/cw:new-session`.
- Otherwise (user manually invoked this): Say "All set! Reloading session..." then run `/cw:reload`.

## Important

- Derive everything technical from the repos. Only ask the user what they're working on.
- Do NOT mention internal files (metadata.json) to the user.
- Do NOT write .claude/CLAUDE.md — everything goes into metadata.
- Do NOT create any files — metadata is saved via the cw command.
- Keep instructions concise but complete — they replace CLAUDE.md as the project context.
