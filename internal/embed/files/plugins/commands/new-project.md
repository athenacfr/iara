---
description: Set up a new cw project by creating a .claude/CLAUDE.md with project goals and conventions.
---

# New Project Setup

This is a new cw project that doesn't have a .claude/CLAUDE.md yet. Explore the repo autonomously and create a .claude/CLAUDE.md.

## Process

1. **Explore the repo yourself** — do NOT ask the user about tech stack, structure, or infra:
   - List top-level files and directories
   - Read package.json, go.mod, Cargo.toml, pyproject.toml, requirements.txt, Makefile, docker-compose.yml, or whatever dependency/config files exist
   - Scan a few key source files to understand patterns (naming, formatting, test structure)
   - Check for existing linter configs (.eslintrc, .prettierrc, .golangci.yml, etc.)
   - Check for CI configs (.github/workflows/, .gitlab-ci.yml, etc.)
   - Look at git log for commit message style
2. **Ask the user one question**: "What are you working on?" — get a brief description of what the project is and what it aims to do. This is the only user input needed.
3. Present a concise summary of what you found and plan to save (structure, stack, conventions, rules). Ask the user to confirm before writing.
4. Once confirmed, write .claude/CLAUDE.md

## CLAUDE.md Format

Write the file with this structure (adapt based on what you find):

```markdown
# <Project Name>

## Overview
<what the project does and its goals — from user>

## Structure
<key directories and what they contain — from your exploration>

## Tech Stack
<languages, frameworks, databases — from your exploration>

## Conventions
<coding conventions, formatting, patterns, commit style — from your exploration>

## Rules
<any rules from user, plus anything you inferred (e.g., "tests use pytest", "commits follow conventional commits")>
```

## Important

- Write .claude/CLAUDE.md using Bash (e.g., `mkdir -p .claude && cat > .claude/CLAUDE.md << 'EOF' ... EOF`) — do NOT use the Write tool, as it is blocked by the write guard
- Do NOT create any other files
- Keep it concise — this is a reference doc, not a novel
- Derive everything technical from the repo itself. Only ask the user what they're working on.
- Do NOT mention CLAUDE.md to the user. Just present the summary and ask for confirmation.
- After saving, say: "All set! Reloading session to apply changes..."
- Then immediately run `/cw:reload` to reload the session with the new configuration. Do NOT ask — just do it.
