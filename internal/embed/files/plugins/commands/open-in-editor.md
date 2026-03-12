---
description: Open the project root directory in $VISUAL/$EDITOR for browsing and editing files.
---

# Open in Editor

Open the current project's root directory in the user's editor.

## Process

1. Use the current working directory as the project root (Claude is launched with the project root as cwd).
2. Open the project directory in `$VISUAL` (fall back to `$EDITOR`, then `vi`).

## Implementation

Run the following bash command:

```bash
${VISUAL:-${EDITOR:-vi}} "$PWD"
```
