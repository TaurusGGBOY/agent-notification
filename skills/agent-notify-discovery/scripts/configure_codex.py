#!/usr/bin/env python3
"""Configure Codex SessionStart/Stop hooks for agent notifications."""

import argparse
import json
import os
import shlex
import sys
import urllib.request

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))


def hooks_path(path=None):
    if path:
        return os.path.abspath(os.path.expanduser(path))
    return os.path.join(os.path.expanduser("~"), ".codex", "hooks.json")


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
    message = "Codex task started" if event == "start" else "Codex task complete"
    return (
        f"python3 {shlex.quote(send_script)} "
        f"--url {shlex.quote(server_url)} "
        f"--agent {shlex.quote(agent)} "
        f"--event {event} "
        '--project "$(basename "$PWD")" '
        '--cwd "$PWD" '
        f"--message {shlex.quote(message)} "
        ">/dev/null 2>&1 || true"
    )


def hook_entry(server_url, agent, event):
    return {
        "type": "command",
        "command": command_for(server_url, agent, event),
        "timeout": 30,
        "statusMessage": f"Sending Codex {event} notice",
    }


def is_agent_notify_hook(hook, event):
    status = hook.get("statusMessage", "")
    command = hook.get("command", "")
    if status == f"Sending Codex {event} notice":
        return True
    return "send.py" in command and "--agent codex" in command and f"--event {event}" in command


def upsert_event_hook(hooks, event_name, matcher, hook, normalized_event):
    entries = hooks.setdefault(event_name, [])
    for entry in entries:
        existing_hooks = entry.get("hooks", [])
        entry["hooks"] = [
            existing for existing in existing_hooks if not is_agent_notify_hook(existing, normalized_event)
        ]

    entries[:] = [entry for entry in entries if entry.get("hooks")]

    target = None
    for entry in entries:
        if entry.get("matcher", "") == matcher:
            target = entry
            break

    if target is None:
        target = {"matcher": matcher, "hooks": []}
        entries.append(target)

    target.setdefault("hooks", []).append(hook)


def configure_hooks(settings, server_url, agent, events):
    hooks = settings.setdefault("hooks", {})

    if "start" in events:
        upsert_event_hook(
            hooks,
            "SessionStart",
            "startup|resume",
            hook_entry(server_url, agent, "start"),
            "start",
        )

    if "stop" in events:
        upsert_event_hook(
            hooks,
            "Stop",
            "",
            hook_entry(server_url, agent, "stop"),
            "stop",
        )

    return settings


def main():
    parser = argparse.ArgumentParser(description="Configure Codex hooks")
    parser.add_argument("--url", required=True, help="Agent Notify server URL")
    parser.add_argument("--agent", default="codex", help="Agent name")
    parser.add_argument("--events", nargs="+", default=["start", "stop"], choices=["start", "stop"])
    parser.add_argument("--path", help="hooks.json path; default is ~/.codex/hooks.json")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--test", action="store_true", help="Send a stop test notification after saving")
    args = parser.parse_args()

    try:
        manifest = fetch_manifest(args.url)
        print(f"Connected to {manifest.get('name', 'unknown')} v{manifest.get('version', '?')}")
    except Exception as err:
        print(f"Warning: cannot reach {args.url}: {err}", file=sys.stderr)

    path = hooks_path(args.path)
    settings = load_settings(path)
    configure_hooks(settings, args.url, args.agent, args.events)

    if args.dry_run:
        hooks = settings.get("hooks", {})
        print(json.dumps({k: hooks[k] for k in ("SessionStart", "Stop") if k in hooks}, indent=2))
        return 0

    save_settings(path, settings)
    print(f"Hooks configured in {path}")
    print("Restart Codex for changes to take effect. If Codex asks to trust hooks, approve only after reviewing this file.")

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
