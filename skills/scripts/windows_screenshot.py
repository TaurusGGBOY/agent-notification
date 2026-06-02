#!/usr/bin/env python
from __future__ import print_function

import argparse
import json
import os
import sys
import time
import traceback


def default_output_path():
    return os.path.join(os.path.expanduser("~"), "Desktop", "agentnotify-python-screen.png")


def default_status_path(output_path):
    root, _ext = os.path.splitext(output_path)
    return root + ".json"


def ensure_parent(path):
    parent = os.path.dirname(os.path.abspath(path))
    if parent and not os.path.isdir(parent):
        os.makedirs(parent)


def write_status(path, data):
    if not path:
        return
    ensure_parent(path)
    with open(path, "w") as f:
        json.dump(data, f, sort_keys=True)


def capture_screenshot(output_path, status_path=None, all_screens=True, image_grab=None):
    started = time.time()
    data = {
        "ok": False,
        "path": output_path,
        "allScreens": bool(all_screens),
        "time": started,
    }
    try:
        if image_grab is None:
            from PIL import ImageGrab  # noqa: WPS433

            image_grab = ImageGrab

        ensure_parent(output_path)
        image = image_grab.grab(all_screens=all_screens)
        image.save(output_path)

        width, height = image.size
        data.update(
            {
                "ok": True,
                "path": output_path,
                "size": [width, height],
                "bytes": os.path.getsize(output_path),
                "elapsedSeconds": round(time.time() - started, 3),
            }
        )
        return 0, data
    except Exception as exc:  # pragma: no cover - exercised through callers/tests
        data.update(
            {
                "ok": False,
                "error": repr(exc),
                "traceback": traceback.format_exc(),
                "elapsedSeconds": round(time.time() - started, 3),
            }
        )
        return 2, data
    finally:
        write_status(status_path, data)


def parse_args(argv):
    parser = argparse.ArgumentParser(
        description="Capture the Windows desktop to a PNG file with Pillow ImageGrab."
    )
    parser.add_argument(
        "--output",
        default=default_output_path(),
        help="PNG output path. Defaults to the current user's Desktop.",
    )
    parser.add_argument(
        "--status",
        default=None,
        help="JSON status path. Defaults to OUTPUT with .json extension.",
    )
    parser.add_argument(
        "--single-screen",
        action="store_true",
        help="Capture only the primary screen instead of all monitors.",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Do not print the JSON status to stdout.",
    )
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv or sys.argv[1:])
    status_path = args.status or default_status_path(args.output)
    code, data = capture_screenshot(
        args.output,
        status_path=status_path,
        all_screens=not args.single_screen,
    )
    if not args.quiet:
        print(json.dumps(data, sort_keys=True))
    return code


if __name__ == "__main__":
    sys.exit(main())
