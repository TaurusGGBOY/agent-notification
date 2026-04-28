#!/usr/bin/env python3
"""Configure Claude Code hooks for Agent Notification."""

import os
import json
import sys
import argparse
import requests
from pathlib import Path

SETTINGS_PATH = Path.home() / ".claude" / "settings.json"
PROJECT_SETTINGS_DIR = Path.cwd() / ".claude"

DEFAULT_URL = "http://localhost:17891"


def load_settings():
    if SETTINGS_PATH.exists():
        with open(SETTINGS_PATH) as f:
            return json.load(f)
    return {}


def save_settings(settings):
    SETTINGS_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(SETTINGS_PATH, "w") as f:
        json.dump(settings, f, indent=2)


def load_project_settings():
    if PROJECT_SETTINGS_DIR.exists():
        p = PROJECT_SETTINGS_DIR / "settings.json"
        if p.exists():
            with open(p) as f:
                return json.load(f)
    return {}


def save_project_settings(settings):
    PROJECT_SETTINGS_DIR.mkdir(parents=True, exist_ok=True)
    p = PROJECT_SETTINGS_DIR / "settings.json"
    with open(p, "w") as f:
        json.dump(settings, f, indent=2)


def get_manifest(url):
    resp = requests.get(f"{url}/manifest", timeout=5)
    resp.raise_for_status()
    return resp.json()


def configure_hook(settings, url, agent, event, scope="user"):
    hooks = settings.get("hooks", {})

    start_cmd = f'python scripts/send.py --url {url} --agent {agent} --event start'
    stop_cmd = f'python scripts/send.py --url {url} --agent {agent} --event stop'

    if scope == "project":
        hooks["SessionStart"] = start_cmd
        hooks["Stop"] = stop_cmd
    else:
        if "hooks" not in settings:
            settings["hooks"] = {}
        settings["hooks"]["SessionStart"] = start_cmd
        settings["hooks"]["Stop"] = stop_cmd

    settings["hooks"] = hooks
    return settings


def main():
    parser = argparse.ArgumentParser(description="Configure Claude Code hooks")
    parser.add_argument("--url", default=DEFAULT_URL, help="Server URL")
    parser.add_argument("--agent", default="claude", help="Agent name")
    parser.add_argument("--events", nargs="+", default=["start", "stop"],
                        choices=["start", "stop"], help="Events to enable")
    parser.add_argument("--scope", choices=["user", "project"], default="user",
                        help="User-level or project-level config")
    parser.add_argument("--test", action="store_true", help="Send test notification")

    args = parser.parse_args()

    try:
        manifest = get_manifest(args.url)
        print(f"Connected to {manifest.get('name', 'unknown')} v{manifest.get('version', '?')}")
        print(f"Server URL: {args.url}")
    except Exception as e:
        print(f"Warning: Cannot reach server at {args.url}: {e}", file=sys.stderr)
        print("Proceeding with configuration anyway...", file=sys.stderr)

    if args.scope == "project":
        settings = load_project_settings()
        configure_hook(settings, args.url, args.agent, None, "project")
        save_project_settings(settings)
        print(f"Hooks configured in project (.claude/settings.json)")
    else:
        settings = load_settings()
        configure_hook(settings, args.url, args.agent, None, "user")
        save_settings(settings)
        print(f"Hooks configured in user settings (~/.claude/settings.json)")

    if args.test:
        from scripts.send import send_notification
        result = send_notification(args.url, args.agent, "stop",
                                   project="test", message="Test notification",
                                   strict=True)
        sys.exit(result)


if __name__ == "__main__":
    main()
