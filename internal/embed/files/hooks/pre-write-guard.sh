#!/bin/bash
# Prevents writing files directly in the cw project root.
# Only .gitignore and .env.*.override are allowed in root.
# .claude/* is managed by cw and also blocked.

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0

# Use the stable project root from cw, not the potentially-changed cwd
PROJECT_ROOT="${CW_PROJECT_DIR:-}"
[ -z "$PROJECT_ROOT" ] && exit 0

# If file is not under the project root at all, allow it
case "$FILE" in
  "$PROJECT_ROOT"/*) ;;
  *) exit 0 ;;
esac

# Get path relative to project root
REL="${FILE#$PROJECT_ROOT/}"

case "$REL" in
  .claude/*)
    echo "You shouldn't write in .claude/ — it's managed by cw. Try managing this inside a repo's .claude/ directory instead."
    exit 2
    ;;
  */*) exit 0 ;;            # inside a subfolder — allowed
  .gitignore) exit 0 ;;     # gitignore — allowed
  .env.*.override) exit 0 ;; # env override files — allowed
  "") exit 0 ;;             # empty — skip
  *)
    echo "You shouldn't create files in the project root. Try creating '$REL' inside a repo subfolder instead."
    exit 2
    ;;
esac
