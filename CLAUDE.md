# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Windows server (cross-compile from macOS)
cd windows-server
make build          # → agent-notify-server.exe (amd64)
make build-arm      # → agent-notify-server-arm64.exe (ARM64)
make test           # 46 tests
make run            # local run (non-Windows: toast stub)

# Install skill
../scripts/install-skill.sh [--test]
```

**Windows project path**: `D:\project\agent-notification`
**Dev machine**: `Administrator@192.168.31.167` (SSH key configured)

## Architecture

```
windows-server/          Go HTTP server (Windows toast notifications)
  main.go                Entry: HTTP :17891 + UDP discovery :17892
  handlers.go            /health, /manifest, /notify endpoints
  settings.go            GET/POST /settings, GET /config
  config.go             %APPDATA%\AgentNotify\config.json
  udp.go                UDP broadcast listener
  toast.go              Cross-platform stub
  toast_windows.go      Windows toast via PowerShell WinRT bridge
  windows_test.go       34 unit tests
  Makefile              cross-compile via GOOS=windows GOARCH=amd64

skills/agent-notify-discovery/    Claude Code skill (separate repo)
  scripts/discover.py   UDP broadcast discovery
  scripts/send.py      POST /notify, stdin JSON, exit 0 on fail
  scripts/configure_claude.py  Hook setup, idempotent

scripts/install-skill.sh    Symlink skill → ~/.claude/skills/
```

## Remote Windows Deployment

```bash
# Deploy to Windows dev machine (192.168.31.167)
scp windows-server/agent-notify-server.exe Administrator@192.168.31.167:C:/Users/Administrator/
ssh Administrator@192.168.31.167 "cd C:/Users/Administrator && start /B agent-notify-server.exe"
```

**Dev machine**: `Administrator@192.168.31.167` (key already configured)

**Files**:
- `toast_xml.go` - ToastGeneric XML generation (cross-platform)
- `toast_card.go` - PNG card renderer with hero image
- `toast_windows.go` - PowerShell WinRT sender (replaces go-toast)
- `toast_stub.go` - Non-Windows stub

**Supported styles**: `clean`, `status-color`, `agent-badge`, `compact`, `custom-card`

**HTTP** (`0.0.0.0:17891`):
- `GET /health` → `{status, version}`
- `GET /manifest` → `{name, version, url, supportedEvents, supportedStyles}`
- `POST /notify` → payload, returns 204 or 400
- `GET /settings` → HTML settings UI
- `GET /config` → current config JSON
- `POST /settings` → save config

**UDP discovery** (`17892`):
- Listen for `AGENT_NOTIFY_DISCOVER v1`
- Response JSON: `{url, hostname, supportedEvents, supportedStyles}`
