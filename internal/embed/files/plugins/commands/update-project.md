---
description: Re-scan the project and update CLAUDE.md with current repos, structure, and conventions.
---

# Update Project

Re-scan the current cw project and update the CLAUDE.md to reflect the current state of all repos.

## Process

1. **Scan the project directory** — find all git repos (subdirectories with `.git/`):
   - For each repo, note: name, primary language (from files or config), key directories, frameworks/libraries (from go.mod, package.json, Cargo.toml, pyproject.toml, etc.)
   - Check for new repos that aren't documented in CLAUDE.md yet
   - Check for repos documented in CLAUDE.md that no longer exist

2. **Read the existing CLAUDE.md** at the project root:
   - If it doesn't exist, tell the user to run `/new-project` first and stop
   - Parse the current structure, tech stack, conventions, and rules sections

3. **Detect changes** — compare what's in CLAUDE.md vs what's on disk:
   - New repos added since last update
   - Repos removed since last update
   - Tech stack changes (new dependencies, language changes)
   - New config files (CI, linters, etc.) that suggest new conventions

4. **Present a diff summary** to the user:
   ```
   ## Changes detected

   ### Repos
   + added: <new-repo>
   - removed: <old-repo>
   ~ updated: <repo> (new dependencies: ...)

   ### Structure changes
   ...

   ### Convention changes
   ...
   ```

5. **Ask for confirmation** before writing. The user may want to adjust.

6. **Update the CLAUDE.md** — preserve user-written sections (Overview, Rules) and update auto-detected sections (Structure, Tech Stack, Repos).

## Important

- NEVER overwrite user-written content in Overview or Rules sections without explicit approval
- NEVER remove repos from CLAUDE.md without confirming they're actually gone from disk
- Keep the same CLAUDE.md format — only update what changed
- If nothing changed, say so and stop
- Be concise in the diff summary — only show actual changes
- Explore each repo thoroughly: read config files, scan key source files, check for new patterns
