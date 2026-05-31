import sys
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
sys.path.insert(0, str(SCRIPT_DIR))

import configure_codex  # noqa: E402


class ConfigureCodexTests(unittest.TestCase):
    def test_adds_codex_hooks_without_removing_existing_hooks(self):
        settings = {
            "hooks": {
                "Stop": [
                    {
                        "matcher": "",
                        "hooks": [
                            {
                                "type": "command",
                                "command": "/tmp/other-stop.sh",
                                "timeout": 5,
                                "statusMessage": "Other stop hook",
                            }
                        ],
                    }
                ]
            }
        }

        configure_codex.configure_hooks(settings, "http://localhost:17891", "codex", ["start", "stop"])

        session_hooks = settings["hooks"]["SessionStart"][0]["hooks"]
        self.assertEqual(settings["hooks"]["SessionStart"][0]["matcher"], "startup|resume")
        self.assertEqual(session_hooks[0]["type"], "command")
        self.assertIn("--agent codex", session_hooks[0]["command"])
        self.assertIn("--event start", session_hooks[0]["command"])
        self.assertEqual(session_hooks[0]["timeout"], 30)
        self.assertEqual(session_hooks[0]["statusMessage"], "Sending Codex start notice")

        stop_hooks = settings["hooks"]["Stop"][0]["hooks"]
        self.assertEqual(stop_hooks[0]["command"], "/tmp/other-stop.sh")
        self.assertIn("--agent codex", stop_hooks[1]["command"])
        self.assertIn("--event stop", stop_hooks[1]["command"])
        self.assertEqual(stop_hooks[1]["statusMessage"], "Sending Codex stop notice")

    def test_replaces_existing_agent_notify_hook_instead_of_duplicating(self):
        settings = {
            "hooks": {
                "Stop": [
                    {
                        "matcher": "",
                        "hooks": [
                            {
                                "type": "command",
                                "command": "python3 /old/agent-notify-discovery/scripts/send.py --url http://old --agent codex --event stop",
                                "timeout": 30,
                                "statusMessage": "Sending Codex stop notice",
                            }
                        ],
                    }
                ]
            }
        }

        configure_codex.configure_hooks(settings, "http://new.example:17891", "codex", ["stop"])
        configure_codex.configure_hooks(settings, "http://new.example:17891", "codex", ["stop"])

        stop_hooks = settings["hooks"]["Stop"][0]["hooks"]
        agent_notify_hooks = [
            hook for hook in stop_hooks if hook.get("statusMessage") == "Sending Codex stop notice"
        ]
        self.assertEqual(len(agent_notify_hooks), 1)
        self.assertIn("--url http://new.example:17891", agent_notify_hooks[0]["command"])


if __name__ == "__main__":
    unittest.main()
