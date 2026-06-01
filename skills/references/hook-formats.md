# Hook Formats Reference

This documents hook configurations for supported agents.

## Claude Code

Claude Code stores user hooks in `~/.claude/settings.json`. Project hooks use `.claude/settings.json`.

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /path/to/scripts/send.py --url http://IP:17891 --agent claude --event start --project \"${CLAUDE_PROJECT_DIR:-unknown}\""
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /path/to/scripts/send.py --url http://IP:17891 --agent claude --event stop --project \"${CLAUDE_PROJECT_DIR:-unknown}\""
          }
        ]
      }
    ]
  }
}
```

Use script:

```bash
python scripts/configure_claude.py \
  --url http://IP:17891 \
  --agent claude \
  --events start stop \
  --scope user
```

## Codex

Codex stores user hooks in `~/.codex/hooks.json`. It may prompt to trust hooks after config changes; review the file before approving.

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /path/to/scripts/send.py --url http://IP:17891 --agent codex --event start --project \"$(basename \"$PWD\")\" --cwd \"$PWD\" --message 'Codex task started' >/dev/null 2>&1 || true",
            "timeout": 30,
            "statusMessage": "Sending Codex start notice"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /path/to/scripts/send.py --url http://IP:17891 --agent codex --event stop --project \"$(basename \"$PWD\")\" --cwd \"$PWD\" --message 'Codex task complete' >/dev/null 2>&1 || true",
            "timeout": 30,
            "statusMessage": "Sending Codex stop notice"
          }
        ]
      }
    ]
  }
}
```

Use script:

```bash
python scripts/configure_codex.py \
  --url http://IP:17891 \
  --agent codex \
  --events start stop
```

`configure_codex.py` preserves unrelated hook commands and replaces only prior Agent Notify Codex hooks.

## Hook Payload

Hook input is JSON on stdin when the agent provides it. `send.py` uses explicit CLI args as defaults and lets stdin override only fields present in the payload.

```json
{
  "agent": "claude|codex|openclaw",
  "event": "stop",
  "project": "agent-notification",
  "cwd": "/path/to/project",
  "message": "Agent task complete",
  "timestamp": "2026-04-28T14:32:00+08:00",
  "sourcePayload": {}
}
```

## OpenClaw

OpenClaw support is installed as a workspace plugin in `~/.openclaw/plugins/agent-notify` and enabled from `~/.openclaw/openclaw.json`.

```json
{
  "plugins": {
    "allow": ["agent-notify"],
    "load": {
      "paths": ["~/.openclaw/plugins/agent-notify"]
    },
    "entries": {
      "agent-notify": {
        "enabled": true,
        "config": {
          "serverUrl": "http://IP:17891",
          "events": ["start", "stop"]
        }
      }
    }
  }
}
```

The generated plugin registers:

- `before_model_resolve` → `start`
- `agent_end` → `stop`

Use script:

```bash
python scripts/configure_openclaw.py \
  --url http://IP:17891 \
  --events start stop
```

Restart OpenClaw Gateway after changing the plugin or config.
