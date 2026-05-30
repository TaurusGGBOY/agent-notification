import importlib.util
import json
from pathlib import Path
import unittest


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "send.py"
SPEC = importlib.util.spec_from_file_location("agent_notify_send", SCRIPT_PATH)
send = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(send)


class NormalizePayloadTests(unittest.TestCase):
    def test_uses_workdir_alias_when_cwd_missing(self):
        payload = send.normalize_payload(
            json.dumps(
                {
                    "agent": "codex",
                    "event": "stop",
                    "workdir": "/Users/me/project",
                }
            )
        )

        self.assertEqual(payload["cwd"], "/Users/me/project")


if __name__ == "__main__":
    unittest.main()
