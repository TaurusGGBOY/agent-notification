#!/usr/bin/env python3
"""Poll Agent Notify history and display macOS native notifications."""

import argparse
import json
import subprocess
import sys
import time
import urllib.error
import urllib.request


def applescript_string(value):
    return '"' + str(value).replace("\\", "\\\\").replace('"', '\\"') + '"'


def send_mac_notification(title, message, sound="Glass"):
    script = (
        f"display notification {applescript_string(message)} "
        f"with title {applescript_string(title)} "
        f"sound name {applescript_string(sound)}"
    )
    subprocess.run(["osascript", "-e", script], check=False)


def fetch_history(url):
    req = urllib.request.Request(url.rstrip("/") + "/history")
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read())


def notification_key(item):
    return (
        item.get("time"),
        item.get("agent"),
        item.get("event"),
        item.get("project"),
        item.get("message"),
    )


def unseen_notifications(items, seen):
    fresh = []
    for item in reversed(items):
        key = notification_key(item)
        if key in seen:
            continue
        seen.add(key)
        fresh.append(item)
    return fresh


def format_notification(item):
    agent = item.get("agent", "Unknown")
    event = item.get("event", "unknown")
    project = item.get("project", "")
    message = item.get("message", "")

    title = f"Agent: {agent}"
    if project:
        title += f" [{project}]"

    body = f"Event: {event}"
    if message:
        body += f"\n{message}"

    return title, body


def poll_notifications(url, interval=5, include_existing=False):
    print(f"Listening for notifications from {url} (polling every {interval}s)")
    print("Press Ctrl+C to stop\n")

    seen = set()
    first_poll = True

    while True:
        try:
            data = fetch_history(url)
            items = data.get("items", [])
            if first_poll and not include_existing:
                for item in items:
                    seen.add(notification_key(item))
            else:
                for item in unseen_notifications(items, seen):
                    title, body = format_notification(item)
                    send_mac_notification(title, body)
                    print(f"  -> {title}: {body}")
            first_poll = False

        except urllib.error.URLError as err:
            print(f"Connection error: {err}", file=sys.stderr)
        except Exception as err:
            print(f"Error: {err}", file=sys.stderr)

        time.sleep(interval)


def main(argv=None):
    parser = argparse.ArgumentParser(description="Mac notification listener")
    parser.add_argument("--url", required=True, help="Agent Notify server URL")
    parser.add_argument("--interval", type=int, default=5, help="Poll interval in seconds")
    parser.add_argument("--once", action="store_true", help="Fetch once and exit")
    parser.add_argument("--include-existing", action="store_true", help="Notify existing history rows on first poll")

    args = parser.parse_args(argv)

    if args.once:
        try:
            print(json.dumps(fetch_history(args.url), indent=2))
            return 0
        except Exception as err:
            print(f"Error: {err}", file=sys.stderr)
            return 1

    try:
        poll_notifications(args.url, args.interval, args.include_existing)
    except KeyboardInterrupt:
        print("\nStopped.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
