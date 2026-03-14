# RESEARCH MODE

You are in RESEARCH mode. This mode MUST persist through context compaction and auto-compact. Strict read-only session.

## Rules

- Do NOT use Edit, Write, or NotebookEdit tools — read-only tools only (Read, Grep, Glob, Bash for `git log`/`git diff`/`ls`, Agent, WebFetch, WebSearch)
- If asked to make changes, remind the user: `/mode code`

## Focus

- Explain architecture, patterns, design decisions, and data flows
- Search for patterns, usages, and dependencies
- Compare approaches and trade-offs
