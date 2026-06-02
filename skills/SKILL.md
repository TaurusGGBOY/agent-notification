---
name: agent-notify-discovery
description: Use when users need to discover Agent Notify Windows servers or configure Claude Code, Codex, or OpenClaw start/stop notification hooks.
---

# Agent Notify Discovery Skill

Discover Agent Notify Windows servers and configure agent hooks for Claude Code, Codex, or OpenClaw.

## Default Workflow

Do not only explain setup steps when this skill is invoked to configure notifications. Run the one-command setup script from this skill directory by default:

```bash
python scripts/setup.py --agents claude codex openclaw --events start stop --test
```

Use an absolute path to `scripts/setup.py` if the current working directory is not this skill directory.

If discovery finds no server, ask the user for the Agent Notify URL, then rerun:

```bash
python scripts/setup.py --url http://IP:17891 --agents claude codex openclaw --events start stop --test
```

If the user asks for a preview or safety check, run the same command with `--dry-run` first. If Codex asks to trust hooks, tell the user to review `~/.codex/hooks.json` before approving. OpenClaw changes require restarting the OpenClaw Gateway.

## Usage

```
/agent-notify-discovery
```

Or use individual scripts.

## Scripts

### discover.py
mDNS/DNS-SD discovery for `_agent-notify._tcp.local.`. Use `--manual http://IP:17891` if multicast discovery is blocked.

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
- `zeroconf` library for automatic mDNS discovery
