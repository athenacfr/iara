---
description: Run development commands (dev servers, watchers, type generators) in the background. Auto-discovers commands on first run. Usage: /dev [stop|restart|status|update|logs]
---

# Dev — Background Development Commands

Run development commands (dev servers, build watchers, type generators) in the background for all subprojects.

## Argument

The argument is: `$ARGUMENTS`

## Config File

Dev commands are persisted at `$IARA_TASK_DIR/dev-config.json`:

```json
{
  "portBase": 4200,
  "subprojects": [
    {
      "path": "frontend",
      "port": 4200,
      "commands": [
        {
          "cmd": "npm run generate:types",
          "description": "Generate GraphQL types from backend schema",
          "type": "one-shot",
          "priority": 2
        },
        {
          "cmd": "npm run dev",
          "description": "Vite dev server with HMR",
          "type": "long-running",
          "priority": 3
        }
      ]
    },
    {
      "path": "backend",
      "venv": ".venv",
      "port": 4201,
      "commands": [
        {
          "cmd": "alembic upgrade head",
          "description": "Run database migrations",
          "type": "one-shot",
          "priority": 0
        },
        {
          "cmd": "uvicorn main:app --reload",
          "description": "FastAPI dev server",
          "type": "long-running",
          "priority": 1
        }
      ]
    }
  ]
}
```

In this example:

- **Priority 0**: `alembic upgrade head` runs first (migrations must complete before anything)
- **Priority 1**: `uvicorn` starts (backend API must be running for codegen)
- **Priority 2**: `npm run generate:types` runs after the backend is up (needs the live API to generate types)
- **Priority 3**: `npm run dev` starts last (needs the generated types to compile)

**Command fields:**

- `type` — `one-shot` (runs once: codegen, migrations) or `long-running` (continuous: dev servers, watchers)
- `priority` (number) — execution order. Commands with the same priority run in parallel; lower priorities run first. Example: migrations at priority 0 complete before dev servers at priority 1 start.

**Optional fields per subproject:**

- `venv` (string) — path to a Python virtual environment relative to the subproject root. When set, commands are prefixed with `source <venv>/bin/activate &&`.
- `port` (number) — the port this subproject's dev server listens on.

## Process

### /dev (no argument or first run)

First, check if a config exists and is current:

```bash
iara internal dev-delete-outdated
```

- `Config is up to date` (exit 0) → proceed to **launch**
- `Outdated config removed` (exit 1) → config was deleted, proceed to **discover**
- `No dev config found` (exit 1) → proceed to **discover**

#### If config is current — launch

Run:

```bash
iara internal dev-launch
```

This single command handles everything: port conflict detection, one-shot execution, long-running process launch, log management, and process supervision. It runs as a background task — use `run_in_background: true`.

The command outputs a summary table with status and URLs when ready.

#### If NO config or outdated — discover and confirm

1. List all subdirectories in the project root (these are subprojects)
2. For each subproject, look for:
   - `package.json` → check `scripts` for dev, start, watch, generate, build:watch, codegen, typecheck entries
   - `Makefile` → check for dev, watch, serve, run, generate targets
   - `Cargo.toml` → cargo watch, cargo run
   - `go.mod` → check Makefile or common go run/air/templ patterns
   - `pyproject.toml` / `manage.py` → check for runserver, celery, uvicorn patterns. Also check if `.venv/` or `venv/` exists — if so, set `venv` field.
   - `docker-compose.yml` → check for dev services
   - `Procfile` / `Procfile.dev` → dev process definitions
3. For each discovered command, classify as `one-shot` or `long-running`:
   - **long-running**: dev, start, watch, serve, runserver
   - **one-shot**: generate, codegen, build, typecheck, migrate
4. Assign priorities based on dependency order:
   - **Priority 0**: Infrastructure one-shots with no dependencies (migrations, db setup)
   - **Priority 1**: Services that other commands depend on (e.g., backend API that codegen needs)
   - **Priority 2**: One-shots that depend on a running service (e.g., type generation from a live API)
   - **Priority 3**: Services that depend on generated output (e.g., frontend dev server needing generated types)
   - Use your judgement — trace the dependency chain and assign increasing priorities accordingly.
5. Compute a `portBase` from the project name (hash mod 5000 + 3000, range 3000-7999). Assign sequential ports to subprojects with long-running dev servers. Modify commands to use the assigned port.
6. Present the discovered config to the user using **AskUserQuestion**:

   ```
   Discovered dev commands:

   frontend/ (Node.js) — port 4200
     - npm run dev          → Vite dev server [long-running]
     - npm run generate     → Generate GraphQL types [one-shot]

   backend/ (Python, venv: .venv) — port 4201
     - uvicorn main:app --reload  → FastAPI dev server [long-running]
     - alembic upgrade head       → Run migrations [one-shot]

   Does this look right? You can:
   - Confirm to save and start
   - Add/remove/modify commands
   - Change ports or venv paths
   - Skip a subproject
   ```

7. If user confirms, write the config to `$IARA_TASK_DIR/dev-config.json` and launch via `iara internal dev-launch`

**Port flag conventions by stack:**

- Node.js/Vite: `--port <N>` or `PORT=<N>`
- Python/uvicorn: `--port <N>`
- Python/Django: `0.0.0.0:<N>` as positional arg to runserver
- Go/air/custom: `PORT=<N>` env var prefix

### /dev stop

Run:

```bash
iara internal dev-stop
```

### /dev restart

Run:

```bash
iara internal dev-restart
```

Stops all running dev commands, clears log files, and re-launches everything from config. Runs as a background task — use `run_in_background: true`.

### /dev update

Re-discover and merge changes into the existing config.

1. Read the existing config from `$IARA_TASK_DIR/dev-config.json`
2. Run the full discovery process (same as first run)
3. Diff the discovered config against the existing and present changes using **AskUserQuestion**
4. Merge accepted changes, preserving manual edits and existing ports
5. Write updated config
6. If dev commands are running, ask: "Restart with updated config?"

### /dev status

Run:

```bash
iara internal dev-status
```

For any failed processes shown, you can offer to restart them.

### /dev logs [subproject] [lines]

Run:

```bash
iara internal dev-logs [subproject] [lines]
```

Default: last 50 lines per subproject.

## Error Handling

- When a background task notification arrives (the dev-launch process exited), run `iara internal dev-status` to check what happened. Offer to restart failed services.
- If the config file is malformed, show the error and offer to re-discover.

## Important

- Discovery (reading package.json, Makefile, etc.) is YOUR job — read actual file contents, don't guess
- Execution (port checks, process management, logs) is handled by `iara internal dev-*` commands
- Always use `run_in_background: true` when launching `iara internal dev-launch`
- **Python venv**: During discovery, check for `.venv/` or `venv/` directories. If found, set the `venv` field in config.
- **Port awareness**: During discovery, infer default ports from config files and command flags.
