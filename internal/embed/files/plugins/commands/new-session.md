---
description: Close the current session and start a fresh one with updated configuration.
---

# CW New Session

Close the current session and start a fresh one. Unlike `/cw:reload` which resumes the conversation, this starts a completely new conversation with updated system prompts and configuration.

## Process

1. Tell the user: "Starting new session..."
2. Run this exact command using the Bash tool:
   ```
   cw internal new-session
   ```
3. The session will close and a new one will start with fresh context.

**Important:** Do NOT do anything else after running the command. The session will restart immediately.
