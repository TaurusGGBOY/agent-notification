import importlib.util
import io
import json
import os
import sys
import tempfile
import unittest
from argparse import Namespace
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
SETUP_PATH = SCRIPT_DIR / "setup.py"


def load_setup_module():
    sys.path.insert(0, str(SCRIPT_DIR))
    spec = importlib.util.spec_from_file_location("agent_notify_setup", SETUP_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class SetupTests(unittest.TestCase):
    def test_setup_writes_claude_codex_and_openclaw_hooks(self):
        setup = load_setup_module()
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                args = Namespace(
                    url="http://localhost:17891",
                    agents=["claude", "codex", "openclaw"],
                    events=["start", "stop"],
                    scope="user",
                    codex_path=None,
                    openclaw_config_path=None,
                    openclaw_plugin_dir=None,
                    dry_run=False,
                    test=False,
                    non_interactive=True,
                )

                result = self.run_quietly(setup, args)

                self.assertEqual(result, 0)
                claude_settings = json.loads(Path(home, ".claude", "settings.json").read_text())
                codex_hooks = json.loads(Path(home, ".codex", "hooks.json").read_text())
                openclaw_config = json.loads(Path(home, ".openclaw", "openclaw.json").read_text())
                self.assertIn("--agent claude", claude_settings["hooks"]["Stop"][0]["hooks"][0]["command"])
                self.assertIn("--agent codex", codex_hooks["hooks"]["Stop"][0]["hooks"][0]["command"])
                self.assertEqual(
                    openclaw_config["plugins"]["entries"]["agent-notify"]["config"]["serverUrl"],
                    "http://localhost:17891",
                )
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home

    def test_dry_run_does_not_write_hook_files(self):
        setup = load_setup_module()
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                args = Namespace(
                    url="http://localhost:17891",
                    agents=["claude", "codex", "openclaw"],
                    events=["start", "stop"],
                    scope="user",
                    codex_path=None,
                    openclaw_config_path=None,
                    openclaw_plugin_dir=None,
                    dry_run=True,
                    test=False,
                    non_interactive=True,
                )

                result = self.run_quietly(setup, args)

                self.assertEqual(result, 0)
                self.assertFalse(Path(home, ".claude", "settings.json").exists())
                self.assertFalse(Path(home, ".codex", "hooks.json").exists())
                self.assertFalse(Path(home, ".openclaw", "openclaw.json").exists())
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home

    def test_dry_run_preview_omits_non_hook_settings(self):
        setup = load_setup_module()
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                claude_dir = Path(home, ".claude")
                claude_dir.mkdir()
                claude_dir.joinpath("settings.json").write_text(
                    json.dumps({"env": {"ANTHROPIC_AUTH_TOKEN": "secret-token"}}),
                    encoding="utf-8",
                )
                args = Namespace(
                    url="http://localhost:17891",
                    agents=["claude"],
                    events=["stop"],
                    scope="user",
                    codex_path=None,
                    openclaw_config_path=None,
                    openclaw_plugin_dir=None,
                    dry_run=True,
                    test=False,
                    non_interactive=True,
                )
                output = io.StringIO()

                with redirect_stdout(output), redirect_stderr(io.StringIO()):
                    result = setup.run_setup(args)

                self.assertEqual(result, 0)
                self.assertIn('"hooks"', output.getvalue())
                self.assertNotIn("secret-token", output.getvalue())
                self.assertNotIn('"env"', output.getvalue())
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home

    def test_discovery_uses_mdns_when_url_omitted(self):
        setup = load_setup_module()

        class FakeDiscover:
            reexec_called = False
            timeout = None

            @classmethod
            def reexec_with_venv_if_available(cls):
                cls.reexec_called = True

            @classmethod
            def discover_mdns(cls, timeout=3.0):
                cls.timeout = timeout
                return [{"url": "http://localhost:17891"}]

        setup.discover = FakeDiscover

        self.assertEqual(setup.discover_server_url(), "http://localhost:17891")
        self.assertTrue(FakeDiscover.reexec_called)
        self.assertEqual(FakeDiscover.timeout, 8.0)

    def test_non_interactive_setup_requires_url_when_discovery_fails(self):
        setup = load_setup_module()
        args = Namespace(
            url="",
            agents=["codex"],
            events=["stop"],
            scope="user",
            codex_path=None,
            openclaw_config_path=None,
            openclaw_plugin_dir=None,
            dry_run=True,
            test=False,
            non_interactive=True,
        )
        setup.discover_server_url = lambda timeout=8.0: None

        self.assertEqual(self.run_quietly(setup, args), 2)

    def run_quietly(self, setup, args):
        with redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()):
            return setup.run_setup(args)


if __name__ == "__main__":
    unittest.main()
