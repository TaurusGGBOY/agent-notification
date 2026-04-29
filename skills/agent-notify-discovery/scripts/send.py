#!/usr/bin/env python3
"""Send a normalized notification to Agent Notify Server."""

import argparse
import json
import sys
import urllib.error
import urllib.request


def normalize_payload(source_json):
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
        "sourcePayload": data,
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

    notify_url = url.rstrip("/") + "/notify"
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        notify_url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            if 200 <= resp.status < 300:
                print("Notification sent successfully")
                return 0
            print(f"Notification failed: HTTP {resp.status}", file=sys.stderr)
            return 1 if strict else 0
    except urllib.error.URLError as err:
        reason = getattr(err, "reason", err)
        print(f"Notification failed: {reason}", file=sys.stderr)
        return 1 if strict else 0
    except Exception as err:
        print(f"Notification failed: {err}", file=sys.stderr)
        return 1 if strict else 0


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

    stdin_data = ""
    if not sys.stdin.isatty():
        stdin_data = sys.stdin.read().strip()

    if stdin_data:
        payload = normalize_payload(stdin_data)
        args.agent = payload.get("agent") or args.agent
        args.event = payload.get("event") or args.event
        args.project = payload.get("project") or args.project
        args.cwd = payload.get("cwd") or args.cwd
        args.message = payload.get("message") or args.message
        args.timestamp = payload.get("timestamp") or args.timestamp

    return send_notification(
        args.url,
        args.agent,
        args.event,
        args.project,
        args.cwd,
        args.message,
        args.timestamp,
        args.strict,
    )


if __name__ == "__main__":
    raise SystemExit(main())
