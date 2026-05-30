#!/usr/bin/env python3
"""Configure Claude Code SessionStart/Stop hooks for agent notifications."""

import argparse
import json
import os
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


def configure_hooks(settings, server_url, agent, events):
    send_script = os.path.join(SCRIPT_DIR, "send.py")
    hooks = settings.setdefault("hooks", {})

    if "start" in events:
        start_cmd = (
            f'python3 {send_script} --url {server_url} --agent {agent} '
            '--event start --project "${CLAUDE_PROJECT_DIR:-unknown}" '
            '--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"'
        )
        hooks["SessionStart"] = [hook_entry(start_cmd)]

    if "stop" in events:
        stop_cmd = (
            f'python3 {send_script} --url {server_url} --agent {agent} '
            '--event stop --project "${CLAUDE_PROJECT_DIR:-unknown}" '
            '--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"'
        )
        hooks["Stop"] = [hook_entry(stop_cmd)]

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
