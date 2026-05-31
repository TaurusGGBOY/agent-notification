# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Windows server (cross-compile from macOS)
cd windows-server
make build          # → agent-notify-server.exe (amd64)
make build-arm      # → agent-notify-server-arm64.exe (ARM64)
make test           # Go tests
make run            # local run (non-Windows: toast stub)

# Install skill
../scripts/install-skill.sh [--test]
```

**Windows project path**: the local checkout directory

## Architecture

```
windows-server/          Go HTTP server (Windows toast notifications)
  main.go                Entry: HTTP :17891 + mDNS broadcast
  handlers.go            /health, /manifest, /notify, /history endpoints
  settings.go            GET/POST /settings, GET /config
  config.go             %APPDATA%\AgentNotify\config.json
  mdns.go               mDNS/DNS-SD advertisement
  toast.go              Cross-platform stub
  toast_windows.go      Windows toast via PowerShell WinRT bridge
  windows_test.go       Unit tests
  Makefile              cross-compile via GOOS=windows GOARCH=amd64

skills/agent-notify-discovery/
  scripts/discover.py          mDNS/DNS-SD discovery
  scripts/send.py              POST /notify, stdin JSON, exit 0 on fail
  scripts/configure_claude.py  Claude hook setup, idempotent
  scripts/configure_codex.py   Codex hook setup, idempotent
  scripts/setup.py             One-command Claude/Codex setup

scripts/install-skill.sh    Copy skill + zeroconf venv → ~/.claude/skills/
```

## Remote Windows Deployment

```bash
# Deploy to Windows dev machine
scp windows-server/agent-notify-server.exe <user>@<host>:C:/Users/<user>/
ssh <user>@<host> "cd C:/Users/<user> && start /B agent-notify-server.exe"
```

**Dev machine**: `<user>@<host>` (SSH key configured)

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
- `GET /history` → latest notification records
- `GET /settings` → HTML settings UI
- `GET /config` → current config JSON
- `POST /settings` → save config

**mDNS discovery**:
- Advertises `_agent-notify._tcp.local.`
- TXT includes `{version, events, styles, path, settings}`
