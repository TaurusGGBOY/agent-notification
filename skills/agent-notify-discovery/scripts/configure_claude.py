#!/usr/bin/env python3
"""Configure Claude Code SessionStart/Stop hooks for agent notifications."""

import argparse
import json
import os
import shlex
import sys
import urllib.request

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))


def settings_path(scope):
    if scope == "project":
        return os.path.join(os.getcwd(), ".claude", "settings.json")
    return os.path.join(os.path.expanduser("~"), ".claude", "settings.json")


def load_settings(path):
    if not os.path.exists(path):
        return {}
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def save_settings(path, settings):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(settings, f, indent=2)
        f.write("\n")


def fetch_manifest(url):
    req = urllib.request.Request(url.rstrip("/") + "/manifest", headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))


def command_for(server_url, agent, event):
    send_script = os.path.join(SCRIPT_DIR, "send.py")
    return (
        f"python3 {shlex.quote(send_script)} "
        f"--url {shlex.quote(server_url)} "
        f"--agent {shlex.quote(agent)} "
        f"--event {event} "
        '--project "${CLAUDE_PROJECT_DIR:-unknown}" '
        '--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"'
    )


def hook_entry(command):
    return {
        "matcher": "",
        "hooks": [
            {
                "type": "command",
                "command": command,
            }
        ],
    }


def is_agent_notify_hook(hook, event, agent):
    command = hook.get("command", "")
    return "send.py" in command and f"--agent {agent}" in command and f"--event {event}" in command


def upsert_event_hook(hooks, event_name, hook, normalized_event, agent):
    entries = hooks.setdefault(event_name, [])
    for entry in entries:
        existing_hooks = entry.get("hooks", [])
        entry["hooks"] = [
            existing for existing in existing_hooks if not is_agent_notify_hook(existing, normalized_event, agent)
        ]

    entries[:] = [entry for entry in entries if entry.get("hooks")]

    target = None
    for entry in entries:
        if entry.get("matcher", "") == "":
            target = entry
            break

    if target is None:
        target = {"matcher": "", "hooks": []}
        entries.append(target)

    target.setdefault("hooks", []).append(hook["hooks"][0])


def configure_hooks(settings, server_url, agent, events):
    hooks = settings.setdefault("hooks", {})

    if "start" in events:
        start_cmd = command_for(server_url, agent, "start")
        upsert_event_hook(hooks, "SessionStart", hook_entry(start_cmd), "start", agent)

    if "stop" in events:
        stop_cmd = command_for(server_url, agent, "stop")
        upsert_event_hook(hooks, "Stop", hook_entry(stop_cmd), "stop", agent)

    return settings


def main():
    parser = argparse.ArgumentParser(description="Configure Claude Code hooks")
    parser.add_argument("--url", required=True, help="Agent Notify server URL")
    parser.add_argument("--agent", default="claude", help="Agent name")
    parser.add_argument("--events", nargs="+", default=["start", "stop"], choices=["start", "stop"])
    parser.add_argument("--scope", choices=["user", "project"], default="user")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--test", action="store_true", help="Send a stop test notification after saving")
    args = parser.parse_args()

    try:
        manifest = fetch_manifest(args.url)
        print(f"Connected to {manifest.get('name', 'unknown')} v{manifest.get('version', '?')}")
    except Exception as err:
        print(f"Warning: cannot reach {args.url}: {err}", file=sys.stderr)

    path = settings_path(args.scope)
    settings = load_settings(path)
    configure_hooks(settings, args.url, args.agent, args.events)

    if args.dry_run:
        hooks = settings.get("hooks", {})
        print(json.dumps({k: hooks[k] for k in ("SessionStart", "Stop") if k in hooks}, indent=2))
        return 0

    save_settings(path, settings)
    print(f"Hooks configured in {path}")
    print("Restart Claude Code for changes to take effect.")

    if args.test:
        import subprocess

        send_script = os.path.join(SCRIPT_DIR, "send.py")
        return subprocess.call(
            [
                sys.executable,
                send_script,
                "--url",
                args.url,
                "--agent",
                args.agent,
                "--event",
                "stop",
                "--project",
                "test",
                "--message",
                "Test notification",
                "--strict",
            ]
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
