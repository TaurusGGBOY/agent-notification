import contextlib
import io
import json
import sys
from pathlib import Path
import unittest


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
sys.path.insert(0, str(SCRIPT_DIR))

import send  # noqa: E402


class SendTests(unittest.TestCase):
    def test_normalize_payload_only_overrides_fields_present_on_stdin(self):
        payload = send.normalize_payload('{"cwd": "/Users/me/project/agent-notification"}')

        self.assertNotIn("agent", payload)
        self.assertNotIn("event", payload)
        self.assertEqual(payload["cwd"], "/Users/me/project/agent-notification")
        self.assertEqual(payload["sourcePayload"]["cwd"], "/Users/me/project/agent-notification")

    def test_project_name_defaults_to_cwd_basename(self):
        self.assertEqual(
            send.default_project_from_cwd("/Users/me/project/agent-notification"),
            "agent-notification",
        )
        self.assertEqual(send.default_project_from_cwd(""), "")

    def test_send_notification_accepts_openclaw_agent(self):
        captured = {}

        class FakeResponse:
            status = 204

            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, tb):
                return False

        def fake_urlopen(req, timeout):
            captured["url"] = req.full_url
            captured["timeout"] = timeout
            captured["payload"] = json.loads(req.data.decode("utf-8"))
            return FakeResponse()

        original_urlopen = send.urllib.request.urlopen
        send.urllib.request.urlopen = fake_urlopen
        try:
            with contextlib.redirect_stdout(io.StringIO()):
                result = send.send_notification(
                    "http://127.0.0.1:17891",
                    "openclaw",
                    "stop",
                    project="agent-notification",
                    strict=True,
                )
        finally:
            send.urllib.request.urlopen = original_urlopen

        self.assertEqual(result, 0)
        self.assertEqual(captured["url"], "http://127.0.0.1:17891/notify")
        self.assertEqual(captured["timeout"], 5)
        self.assertEqual(captured["payload"]["agent"], "openclaw")
        self.assertEqual(captured["payload"]["event"], "stop")

    def test_send_notification_includes_language(self):
        captured = {}

        class FakeResponse:
            status = 204

            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, tb):
                return False

        def fake_urlopen(req, timeout):
            captured["payload"] = json.loads(req.data.decode("utf-8"))
            return FakeResponse()

        original_urlopen = send.urllib.request.urlopen
        send.urllib.request.urlopen = fake_urlopen
        try:
            with contextlib.redirect_stdout(io.StringIO()):
                result = send.send_notification(
                    "http://127.0.0.1:17891",
                    "codex",
                    "start",
                    language="en",
                    strict=True,
                )
        finally:
            send.urllib.request.urlopen = original_urlopen

        self.assertEqual(result, 0)
        self.assertEqual(captured["payload"]["language"], "en")


if __name__ == "__main__":
    unittest.main()
