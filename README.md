# Agent Notification

LAN-only notification receiver. Mac/Unix agents send `start`/`stop` events to a Windows machine → Windows shows desktop toast.

## Components

**Windows server** (`windows-server/`)
- HTTP API on `0.0.0.0:17891`
- UDP discovery on port `17892`
- Toast notifications with style presets
- Settings UI at `http://<windows-ip>:17891/settings`

**Skill** (`skills/agent-notify-discovery/`)
- Installs to `~/.claude/skills/`
- Run `/agent-notify-discovery` or use scripts directly

## Quick Start

### 1. Deploy Windows server

Copy `windows-server/` to Windows machine:
```
start.bat           # Start server
install-startup.bat # Auto-start on login
```

### 2. Install skill

```bash
./scripts/install-skill.sh [--test]
```

### 3. Configure Claude Code hooks

```bash
python skills/agent-notify-discovery/scripts/configure_claude.py \
  --url http://<windows-ip>:17891 \
  --agent claude \
  --events start stop \
  --scope user
```

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

## Build from macOS

```bash
cd windows-server
make build   # → agent-notify-server.exe
```

Requires Go 1.21+.
