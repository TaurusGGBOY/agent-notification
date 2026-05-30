import importlib.util
from pathlib import Path
import unittest


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "configure_claude.py"
SPEC = importlib.util.spec_from_file_location("agent_notify_configure_claude", SCRIPT_PATH)
configure_claude = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(configure_claude)


class ConfigureClaudeHookTests(unittest.TestCase):
    def test_hooks_send_cwd_so_toasts_can_show_directory_first(self):
        settings = configure_claude.configure_hooks({}, "http://127.0.0.1:17891", "claude", ["start", "stop"])

        start_cmd = settings["hooks"]["SessionStart"][0]["hooks"][0]["command"]
        stop_cmd = settings["hooks"]["Stop"][0]["hooks"][0]["command"]

        self.assertIn('--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"', start_cmd)
        self.assertIn('--cwd "${CLAUDE_PROJECT_DIR:-$PWD}"', stop_cmd)


if __name__ == "__main__":
    unittest.main()
