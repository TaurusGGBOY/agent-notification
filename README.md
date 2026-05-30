# Agent Notification

LAN-only notification receiver. Mac/Unix agents send `start`/`stop` events to a Windows machine → Windows shows desktop toast.

## Components

**Windows app** (`tauri-app/`)
- One-click Windows desktop app
- Opens the AgentNotify UI and starts the bundled Go server sidecar
- Server listens on `0.0.0.0:17891` for LAN agent notifications
- Tauri UI controls the server through `127.0.0.1:17891`

**Windows server** (`windows-server/`)
- HTTP API on `0.0.0.0:17891`
- mDNS/DNS-SD discovery as `_agent-notify._tcp.local.`
- Toast notifications with style presets
- Settings UI at `http://<windows-ip>:17891/settings`

**Skill** (`skills/agent-notify-discovery/`)
- Configures Claude Code hooks in `~/.claude/settings.json`
- Configures Codex hooks in `~/.codex/hooks.json`
- Run `/agent-notify-discovery` or use scripts directly

## Quick Start

### 1. Start the Windows app

Windows users should install and start the Tauri app from the release package. The one-click Windows app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agent notifications, and advertises itself with mDNS.

### 2. Install skill

```bash
./scripts/install-skill.sh [--test]
```

### 3. Configure agent hooks

One-command setup for Claude Code and Codex:

```bash
python skills/agent-notify-discovery/scripts/setup.py \
  --url http://<windows-ip>:17891 \
  --agents claude codex \
  --events start stop \
  --test
```

Omit `--url` to try mDNS discovery first.

Individual commands:

Claude Code:

```bash
python skills/agent-notify-discovery/scripts/configure_claude.py \
  --url http://<windows-ip>:17891 \
  --agent claude \
  --events start stop \
  --scope user
```

Codex:

```bash
python skills/agent-notify-discovery/scripts/configure_codex.py \
  --url http://<windows-ip>:17891 \
  --agent codex \
  --events start stop
```

## Windows One-Click App

Windows users should install and start the Tauri app. The app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agent notifications, controls the server locally through `127.0.0.1:17891`, and advertises itself with mDNS.

After opening the app, copy the LAN URL shown in the sidebar into the Claude/Codex setup skill, or let the skill discover the server automatically.

## Notification Styles

- `clean` — plain, low-distraction
- `status-color` — blue (start), green (stop)
- `agent-badge` — shows agent badge
- `compact` — shorter layout

## Events

| Agent hook | Notification event |
|------------|-------------------|
| `SessionStart` | `start` |
| `Stop` | `stop` |

## Protocol

```
POST /notify
{
  "agent": "claude",
  "event": "start|stop",
  "project": "...",
  "cwd": "...",
  "message": "...",
  "timestamp": "..."
}
```

## Desktop Client

The Tauri client lives in `tauri-app/`. It bundles the Go server as a sidecar, adds a system tray, and provides the command-first notification client UI.

```bash
cd tauri-app
npm install
npm run tauri:dev
```

## Build from macOS

```bash
cd windows-server
make build   # → agent-notify-server.exe
```

Requires Go 1.21+.
