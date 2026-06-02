---
name: agent-notify-discovery
description: Use when users need to discover Agent Notify Windows servers or configure Claude Code, Codex, or OpenClaw start/stop notification hooks.
---

# Agent Notify Discovery Skill

Discover Agent Notify Windows servers and configure agent hooks for Claude Code, Codex, or OpenClaw.

## Default Workflow

Do not only explain setup steps when this skill is invoked to configure notifications. Run the one-command setup script from this skill directory by default. Automatic discovery uses the bundled standard library mDNS/DNS-SD client, so there is no package installation step.

```bash
python scripts/setup.py --agents claude codex openclaw --events start stop --test
```

Use an absolute path to `scripts/setup.py` if the current working directory is not this skill directory.

If discovery finds no server, ask the user for the Agent Notify URL, then rerun:

```bash
python scripts/setup.py --url http://IP:17891 --agents claude codex openclaw --events start stop --test
```

If the user asks for a preview or safety check, run the same command with `--dry-run` first. If Codex asks to trust hooks, tell the user to review `~/.codex/hooks.json` before approving. OpenClaw changes require restarting the OpenClaw Gateway.

Use `--url http://IP:17891` when multicast discovery is blocked or the Windows server is on a different network.

## Usage

```
/agent-notify-discovery
```

Or use individual scripts.

## Scripts

### discover.py
mDNS/DNS-SD discovery for `_agent-notify._tcp.local.`. It runs with the Python standard library. Use `--manual http://IP:17891` if multicast discovery is blocked.

```
python scripts/discover.py --timeout 5 --json
```

### setup.py
One-command setup for Claude Code, Codex, and OpenClaw. Discovers a server when `--url` is omitted, writes hooks/plugins, and optionally sends a strict test notification.

```
python scripts/setup.py --url http://IP:17891 --agents claude codex openclaw --events start stop --test
```

### send.py
```
python scripts/send.py --url http://IP:17891 --agent claude|codex|openclaw --event start|stop --language zh|en
```
Reads JSON from stdin when run as a hook. CLI args remain defaults unless stdin explicitly includes the same field. `language` accepts `zh` or `en`; when omitted, the server's configured notification language is used. Exits 0 on network failure unless `--strict` is set.

### windows_screenshot.py
Windows desktop screenshot helper. Use it when the user asks to inspect the Windows client UI or notification popups. It uses Pillow `ImageGrab.grab(all_screens=True)` to capture all monitors into one PNG, similar to `Win+PrtSc`, and writes a JSON status file.

Install Pillow on the Windows host if needed:

```
python -m pip install pillow
```

For older hosts where `python` is Python 2.7, install the last compatible Pillow:

```
C:\Python27\Scripts\pip.exe install Pillow==6.2.2
```

Run manually for debugging:

```
python scripts/windows_screenshot.py --output C:\Users\Administrator\Desktop\agentnotify-python-screen.png
```

Run without a visible console by using `pythonw.exe` from the active Windows desktop session:

```
C:\Python27\pythonw.exe scripts\windows_screenshot.py --quiet --output C:\Users\Administrator\Desktop\agentnotify-python-screen.png
```

When launching remotely, schedule it as an interactive task (`schtasks /Create ... /IT`) so it runs in the logged-in desktop session. Do not rely on SSH-only execution for screenshots; non-interactive sessions often fail with `screen grab failed` or capture the wrong desktop.

### configure_claude.py
Queries manifest, selects events, writes Claude Code hooks to `~/.claude/settings.json` or project `.claude/settings.json`.

### configure_codex.py
Queries manifest, selects events, writes Codex hooks to `~/.codex/hooks.json` while preserving existing hooks.

```
python scripts/configure_codex.py \
  --url http://IP:17891 \
  --agent codex \
  --events start stop
```

### configure_openclaw.py
Installs the Agent Notify OpenClaw plugin in `~/.openclaw/plugins/agent-notify` and enables it in `~/.openclaw/openclaw.json`.

```
python scripts/configure_openclaw.py \
  --url http://IP:17891 \
  --events start stop
```

## Events

- `start` → SessionStart hook
- `stop` → Stop hook
- OpenClaw `start` → `before_model_resolve` plugin hook
- OpenClaw `stop` → `agent_end` plugin hook

## Hook Reference

For manual hook JSON and per-agent notes, read `references/hook-formats.md`.

## Requirements

- Python 3.7+
- Network access to the Agent Notify server on port `17891`
