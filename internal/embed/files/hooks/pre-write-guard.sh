#!/bin/bash
# Prevents writing files directly in the cw project root.
# Only CLAUDE.md, .cw-*, and .claude/* are allowed in root.

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')

[ -z "$FILE" ] || [ -z "$CWD" ] && exit 0

# Get path relative to project root
REL="${FILE#$CWD/}"

# If the file starts with CWD but has no slash, it's in the root
case "$REL" in
  */*) exit 0 ;;            # inside a subfolder — allowed
  CLAUDE.md) exit 0 ;;      # project instructions — allowed
  .cw-*) exit 0 ;;          # cw internal files — allowed
  .claude/*) exit 0 ;;      # claude config — allowed
  .gitignore) exit 0 ;;     # gitignore — allowed
  "") exit 0 ;;             # empty — skip
  *)
    echo "Blocked: cannot write '$REL' in project root. Use a repo subfolder."
    exit 2
    ;;
esac
