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


if __name__ == "__main__":
    unittest.main()
