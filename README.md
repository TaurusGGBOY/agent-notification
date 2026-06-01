# Agent Notification

LAN-only notification receiver. Agents send `start`/`stop` events to AgentNotify → the desktop shows a native notification banner.

## Components

**Desktop app** (`tauri-app/`)
- One-click desktop app for release packages such as macOS `.dmg`
- Opens the AgentNotify UI and starts the bundled Go server sidecar
- Server listens on `0.0.0.0:17891` for LAN agent notifications
- Tauri UI controls the server through `127.0.0.1:17891`

**Windows server** (`windows-server/`)
- HTTP API on `0.0.0.0:17891`
- mDNS/DNS-SD discovery as `_agent-notify._tcp.local.`
- Desktop notifications
- Settings UI at `http://<windows-ip>:17891/settings`

**Skill** (`skills/agent-notify-discovery/`)
- Configures Claude Code hooks in `~/.claude/settings.json`
- Configures Codex hooks in `~/.codex/hooks.json`
- Configures OpenClaw through an Agent Notify plugin in `~/.openclaw/plugins/agent-notify`
- Run `/agent-notify-discovery` or use scripts directly

## Quick Start

### 1. Install and start AgentNotify

On macOS:

1. Double-click the `.dmg` file.
2. Drag `AgentNotify` into `Applications`.
3. Open `Applications`, then launch `AgentNotify`.
4. When macOS asks for notification permission, click **Allow**. If the prompt does not appear, open **System Settings** → **Notifications** → **AgentNotify**, then enable notifications.
5. Click **Test** in AgentNotify. A notification banner should appear in the top-right corner.

The desktop app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agent notifications, and advertises itself with mDNS.

### 2. Install skill

In the system where your agent runs, copy the skill install command from AgentNotify, or run:

```bash
npx skills add TaurusGGBOY/agent-notification
```

### 3. Discover AgentNotify and configure hooks

In your agent, run the installed skill:

```text
/agent-notify-discovery
```

Ask the skill to discover AgentNotify and configure notification hooks for your agent. It can configure both `start` and `stop` events; use `stop` if you only want a notification when a task finishes.

One-command setup for Claude Code, Codex, and OpenClaw:

```bash
python skills/agent-notify-discovery/scripts/setup.py \
  --url http://<agentnotify-ip>:17891 \
  --agents claude codex openclaw \
  --events start stop \
  --test
```

Omit `--url` to try mDNS discovery first.

If discovery fails, copy the LAN URL shown in AgentNotify and pass it explicitly:

```bash
python skills/agent-notify-discovery/scripts/setup.py \
  --url http://<your-ip>:17891 \
  --agents claude codex openclaw \
  --events start stop \
  --test
```

After setup, restart Claude Code or Codex so the new hooks take effect. Restart OpenClaw Gateway so the Agent Notify plugin is loaded. If Codex asks whether to trust hooks, review `~/.codex/hooks.json` before approving.

### 4. Confirm notifications

Start the agent again and send a short message such as:

```text
hi
```

When the agent run finishes, a notification banner should appear in the top-right corner. If you configured `start` events too, you should also see a banner when the agent starts working.

Individual commands:

Claude Code:

```bash
python skills/agent-notify-discovery/scripts/configure_claude.py \
  --url http://<agentnotify-ip>:17891 \
  --agent claude \
  --events start stop \
  --scope user
```

Codex:

```bash
python skills/agent-notify-discovery/scripts/configure_codex.py \
  --url http://<agentnotify-ip>:17891 \
  --agent codex \
  --events start stop
```

OpenClaw:

```bash
python skills/agent-notify-discovery/scripts/configure_openclaw.py \
  --url http://<agentnotify-ip>:17891 \
  --events start stop
```

## Windows One-Click App

Windows users should install and start the Tauri app. The app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agent notifications, controls the server locally through `127.0.0.1:17891`, and advertises itself with mDNS.

After opening the app, copy the LAN URL shown in the sidebar into the Claude/Codex/OpenClaw setup skill, or let the skill discover the server automatically.

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
| OpenClaw `before_model_resolve` | `start` |
| OpenClaw `agent_end` | `stop` |

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
