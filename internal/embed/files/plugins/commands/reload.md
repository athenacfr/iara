---
description: Reload the cw session to pick up new commands, rules, and config changes.
---

# CW Reload

Reload the current cw session. This re-syncs commands, rules, and configuration, then resumes the conversation.

## Process

1. Tell the user: "Reloading session..."
2. Run this exact command using the Bash tool:
   ```
   cw internal reload
   ```
3. The session will close and automatically resume with updated configuration.

**Important:** Do NOT do anything else after running the command. The session will restart immediately.
