import importlib.util
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "mac-notify-listener.py"
SPEC = importlib.util.spec_from_file_location("mac_notify_listener", SCRIPT_PATH)
listener = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(listener)


class MacNotifyListenerTests(unittest.TestCase):
    def test_unseen_notifications_do_not_repeat_history_rows(self):
        seen = set()
        items = [
            {"time": "2026-05-30T10:00:01Z", "agent": "codex", "event": "stop", "project": "p"},
            {"time": "2026-05-30T10:00:00Z", "agent": "codex", "event": "start", "project": "p"},
        ]

        first = listener.unseen_notifications(items, seen)
        second = listener.unseen_notifications(items, seen)

        self.assertEqual([item["event"] for item in first], ["start", "stop"])
        self.assertEqual(second, [])

    def test_format_notification_includes_project_and_message(self):
        title, body = listener.format_notification(
            {
                "agent": "codex",
                "event": "stop",
                "project": "agent-notification",
                "message": "done",
            }
        )

        self.assertEqual(title, "Agent: codex [agent-notification]")
        self.assertEqual(body, "Event: stop\ndone")


if __name__ == "__main__":
    unittest.main()
