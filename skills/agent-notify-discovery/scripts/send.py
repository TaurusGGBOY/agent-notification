#!/usr/bin/env python3
"""Send notification to Agent Notify Server."""

import sys
import json
import argparse
import requests

DEFAULT_TIMEOUT = 3


def normalize_payload(source_json):
    """Extract fields for normalized payload."""
    try:
        data = json.loads(source_json)
    except json.JSONDecodeError:
        data = {}

    return {
        "agent": data.get("agent", "claude"),
        "event": data.get("event", "stop"),
        "project": data.get("project", ""),
        "cwd": data.get("cwd", ""),
        "message": data.get("message", ""),
        "timestamp": data.get("timestamp", ""),
        "sourcePayload": data.get("sourcePayload", {}),
    }


def send_notification(url, agent, event, project="", cwd="", message="", timestamp="", strict=False):
    payload = {
        "agent": agent,
        "event": event,
        "project": project,
        "cwd": cwd,
        "message": message,
        "timestamp": timestamp,
        "sourcePayload": {},
    }

    try:
        resp = requests.post(f"{url}/notify", json=payload, timeout=DEFAULT_TIMEOUT)
        if resp.status_code == 204:
            print("Notification sent.", file=sys.stderr)
            return 0
        else:
            print(f"Server returned {resp.status_code}", file=sys.stderr)
            return 1 if strict else 0
    except requests.exceptions.Timeout:
        print("Timeout reaching server", file=sys.stderr)
        return 0 if not strict else 2
    except requests.exceptions.ConnectionError:
        print("Cannot connect to server", file=sys.stderr)
        return 0 if not strict else 1
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return 0 if not strict else 1


def main():
    parser = argparse.ArgumentParser(description="Send agent notification")
    parser.add_argument("--url", required=True, help="Server URL")
    parser.add_argument("--agent", default="claude", help="Agent name")
    parser.add_argument("--event", required=True, choices=["start", "stop"], help="Event type")
    parser.add_argument("--project", default="", help="Project name")
    parser.add_argument("--cwd", default="", help="Working directory")
    parser.add_argument("--message", default="", help="Notification message")
    parser.add_argument("--timestamp", default="", help="ISO timestamp")
    parser.add_argument("--strict", action="store_true", help="Exit non-zero on failure")

    args = parser.parse_args()

    # Try read stdin for hook JSON
    stdin_data = ""
    if not sys.stdin.isatty():
        stdin_data = sys.stdin.read().strip()

    if stdin_data:
        payload = normalize_payload(stdin_data)
        args.agent = payload.get("agent", args.agent)
        args.event = payload.get("event", args.event)
        args.project = payload.get("project", args.project)
        args.cwd = payload.get("cwd", args.cwd)
        args.message = payload.get("message", args.message)
        args.timestamp = payload.get("timestamp", args.timestamp)

    return send_notification(
        args.url, args.agent, args.event,
        args.project, args.cwd, args.message, args.timestamp,
        args.strict
    )


if __name__ == "__main__":
    sys.exit(main())
