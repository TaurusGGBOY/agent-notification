import sys
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
sys.path.insert(0, str(SCRIPT_DIR))

import configure_claude  # noqa: E402


class ConfigureClaudeTests(unittest.TestCase):
    def test_hooks_send_cwd_so_toasts_can_show_directory_first(self):
        settings = configure_claude.configure_hooks({}, "http://127.0.0.1:17891", "claude", ["start", "stop"])

        start_cmd = settings["hooks"]["SessionStart"][0]["hooks"][0]["command"]
        stop_cmd = settings["hooks"]["Stop"][0]["hooks"][0]["command"]

        self.assertIn('--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"', start_cmd)
        self.assertIn('--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"', stop_cmd)

    def test_adds_claude_hooks_without_removing_existing_hooks(self):
        settings = {
            "hooks": {
                "Stop": [
                    {
                        "matcher": "",
                        "hooks": [
                            {
                                "type": "command",
                                "command": "/tmp/other-stop.sh",
                            }
                        ],
                    }
                ]
            }
        }

        configure_claude.configure_hooks(
            settings,
            "http://localhost:17891",
            "claude",
            ["start", "stop"],
        )

        session_hooks = settings["hooks"]["SessionStart"][0]["hooks"]
        self.assertEqual(settings["hooks"]["SessionStart"][0]["matcher"], "")
        self.assertIn("--agent claude", session_hooks[0]["command"])
        self.assertIn("--event start", session_hooks[0]["command"])

        stop_hooks = settings["hooks"]["Stop"][0]["hooks"]
        self.assertEqual(stop_hooks[0]["command"], "/tmp/other-stop.sh")
        self.assertIn("--agent claude", stop_hooks[1]["command"])
        self.assertIn("--event stop", stop_hooks[1]["command"])

    def test_replaces_existing_agent_notify_hook_instead_of_duplicating(self):
        settings = {
            "hooks": {
                "Stop": [
                    {
                        "matcher": "",
                        "hooks": [
                            {
                                "type": "command",
                                "command": "python3 /old/agent-notify-discovery/scripts/send.py --url http://old --agent claude --event stop",
                            }
                        ],
                    }
                ]
            }
        }

        configure_claude.configure_hooks(settings, "http://new.example:17891", "claude", ["stop"])
        configure_claude.configure_hooks(settings, "http://new.example:17891", "claude", ["stop"])

        stop_hooks = settings["hooks"]["Stop"][0]["hooks"]
        agent_notify_hooks = [
            hook
            for hook in stop_hooks
            if "send.py" in hook.get("command", "") and "--agent claude" in hook.get("command", "")
        ]
        self.assertEqual(len(agent_notify_hooks), 1)
        self.assertIn("--url http://new.example:17891", agent_notify_hooks[0]["command"])


if __name__ == "__main__":
    unittest.main()
