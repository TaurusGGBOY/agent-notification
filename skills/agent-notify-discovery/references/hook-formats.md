# Hook Formats Reference

This documents the hook configurations for different agents.

## Claude Code

Claude Code uses `SessionStart` and `Stop` hooks.

```json
{
  "hooks": {
    "SessionStart": "python scripts/send.py --url http://IP:17891 --agent claude --event start",
    "Stop": "python scripts/send.py --url http://IP:17891 --agent claude --event stop"
  }
}
```

Hook input format (JSON on stdin):
```json
{
  "agent": "claude",
  "event": "stop",
  "project": "agent-notification",
  "cwd": "/path/to/project",
  "message": "Agent task complete",
  "timestamp": "2026-04-28T14:32:00+08:00",
  "sourcePayload": {}
}
```

## Codex (Future)

Codex uses hooks in `~/.codex/hooks.json`.

## OpenClaw (Future)

OpenClaw uses hooks in `~/.openclaw/hooks/`.
