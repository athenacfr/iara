#!/usr/bin/env bash
# Auto-compact hook: reads token usage from the Claude session JSONL
# and triggers compact when context usage exceeds CW_AUTO_COMPACT_LIMIT%.

LIMIT="${CW_AUTO_COMPACT_LIMIT:-0}"
[ "$LIMIT" -eq 0 ] 2>/dev/null && exit 0

# Read hook input to get session_id
INPUT=$(cat)
SESSION_ID=$(printf '%s' "$INPUT" | jq -r '.session_id // empty' 2>/dev/null)
[ -z "$SESSION_ID" ] && exit 0

# Derive JSONL path: ~/.claude/projects/{encoded_cwd}/{session_id}.jsonl
PROJECT_DIR="${CW_PROJECT_DIR:-}"
[ -z "$PROJECT_DIR" ] && exit 0

ENCODED_DIR=$(printf '%s' "$PROJECT_DIR" | sed 's|/|-|g')
JSONL="$HOME/.claude/projects/${ENCODED_DIR}/${SESSION_ID}.jsonl"
[ -f "$JSONL" ] || exit 0

# Find the last assistant message with usage data (read file in reverse)
USAGE_LINE=$(tac "$JSONL" | grep -m1 '"input_tokens"')
[ -z "$USAGE_LINE" ] && exit 0

# Extract token counts: total context = input_tokens + cache_creation + cache_read
INPUT_TOKENS=$(printf '%s' "$USAGE_LINE" | grep -o '"input_tokens":[0-9]*' | head -1 | grep -o '[0-9]*')
CACHE_CREATE=$(printf '%s' "$USAGE_LINE" | grep -o '"cache_creation_input_tokens":[0-9]*' | head -1 | grep -o '[0-9]*')
CACHE_READ=$(printf '%s' "$USAGE_LINE" | grep -o '"cache_read_input_tokens":[0-9]*' | head -1 | grep -o '[0-9]*')

INPUT_TOKENS="${INPUT_TOKENS:-0}"
CACHE_CREATE="${CACHE_CREATE:-0}"
CACHE_READ="${CACHE_READ:-0}"

TOTAL=$((INPUT_TOKENS + CACHE_CREATE + CACHE_READ))
[ "$TOTAL" -eq 0 ] && exit 0

# Determine context window from model name
MODEL=$(printf '%s' "$USAGE_LINE" | grep -o '"model":"[^"]*"' | head -1 | sed 's/"model":"//;s/"//')
case "$MODEL" in
  *opus*) CONTEXT_WINDOW=1000000 ;;
  *)      CONTEXT_WINDOW=200000 ;;
esac

# Calculate usage percentage
PCT=$(( (TOTAL * 100) / CONTEXT_WINDOW ))

if [ "$PCT" -ge "$LIMIT" ]; then
  cw internal auto-compact &
fi

exit 0
