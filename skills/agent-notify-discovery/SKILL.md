# Agent Notify Discovery Skill

Discover and configure the Agent Notification Windows server for Claude Code.

## Usage

```
/agent-notify-discovery
```

Or use individual scripts.

## Scripts

### discover.py
UDP broadcast discovery on port 17892. Collects servers, prompts for manual URL if none found.

### send.py
```
python scripts/send.py --url http://IP:17891 --agent claude --event start|stop
```
Reads JSON from stdin (Claude Code hook payload). Exits 0 on network failure.

### configure_claude.py
Interactive setup: queries manifest, selects events, writes Claude Code hooks.

## Events

- `start` → SessionStart hook
- `stop` → Stop hook

## Requirements

- Python 3.7+
- requests library (for HTTP calls)
