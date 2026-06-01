import json
import os
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
sys.path.insert(0, str(SCRIPT_DIR))

import configure_openclaw  # noqa: E402


class ConfigureOpenClawTests(unittest.TestCase):
    def test_installs_plugin_and_enables_openclaw_hooks(self):
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                config_path = Path(home, ".openclaw", "openclaw.json")
                config_path.parent.mkdir(parents=True)
                config_path.write_text(
                    json.dumps(
                        {
                            "plugins": {
                                "allow": ["existing-plugin"],
                                "load": {"paths": ["/opt/existing-plugin"]},
                                "entries": {
                                    "existing-plugin": {"enabled": True},
                                },
                            }
                        }
                    ),
                    encoding="utf-8",
                )

                settings = configure_openclaw.configure_hooks(
                    config_path=str(config_path),
                    plugin_dir=None,
                    server_url="http://127.0.0.1:17891",
                    events=["start", "stop"],
                    dry_run=False,
                )

                plugin_dir = Path(home, ".openclaw", "plugins", "agent-notify")
                manifest = json.loads(plugin_dir.joinpath("openclaw.plugin.json").read_text(encoding="utf-8"))
                handler = plugin_dir.joinpath("index.js").read_text(encoding="utf-8")
                saved = json.loads(config_path.read_text(encoding="utf-8"))
                entry = saved["plugins"]["entries"]["agent-notify"]

                self.assertEqual(settings, saved)
                self.assertEqual(manifest["id"], "agent-notify")
                self.assertIn("before_model_resolve", handler)
                self.assertIn("agent_end", handler)
                self.assertIn(str(plugin_dir), saved["plugins"]["load"]["paths"])
                self.assertIn("existing-plugin", saved["plugins"]["allow"])
                self.assertIn("agent-notify", saved["plugins"]["allow"])
                self.assertTrue(entry["enabled"])
                self.assertEqual(entry["config"]["serverUrl"], "http://127.0.0.1:17891")
                self.assertEqual(entry["config"]["events"], ["start", "stop"])
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home

    def test_replaces_existing_agent_notify_plugin_without_duplicates(self):
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                config_path = Path(home, ".openclaw", "openclaw.json")
                plugin_dir = Path(home, ".openclaw", "plugins", "agent-notify")
                config_path.parent.mkdir(parents=True)
                config_path.write_text(
                    json.dumps(
                        {
                            "plugins": {
                                "allow": ["agent-notify", "agent-notify"],
                                "load": {"paths": [str(plugin_dir), str(plugin_dir)]},
                                "entries": {
                                    "agent-notify": {
                                        "enabled": True,
                                        "config": {
                                            "serverUrl": "http://old.example:17891",
                                            "events": ["stop"],
                                        },
                                    }
                                },
                            }
                        }
                    ),
                    encoding="utf-8",
                )

                configure_openclaw.configure_hooks(
                    config_path=str(config_path),
                    plugin_dir=None,
                    server_url="http://new.example:17891",
                    events=["start"],
                    dry_run=False,
                )

                saved = json.loads(config_path.read_text(encoding="utf-8"))
                self.assertEqual(saved["plugins"]["allow"].count("agent-notify"), 1)
                self.assertEqual(saved["plugins"]["load"]["paths"].count(str(plugin_dir)), 1)
                self.assertEqual(
                    saved["plugins"]["entries"]["agent-notify"]["config"],
                    {"serverUrl": "http://new.example:17891", "events": ["start"]},
                )
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home

    def test_dry_run_does_not_write_config_or_plugin_files(self):
        with tempfile.TemporaryDirectory() as home:
            old_home = os.environ.get("HOME")
            os.environ["HOME"] = home
            try:
                config_path = Path(home, ".openclaw", "openclaw.json")

                settings = configure_openclaw.configure_hooks(
                    config_path=str(config_path),
                    plugin_dir=None,
                    server_url="http://127.0.0.1:17891",
                    events=["stop"],
                    dry_run=True,
                )

                self.assertFalse(config_path.exists())
                self.assertFalse(Path(home, ".openclaw", "plugins", "agent-notify").exists())
                self.assertEqual(
                    settings["plugins"]["entries"]["agent-notify"]["config"],
                    {"serverUrl": "http://127.0.0.1:17891", "events": ["stop"]},
                )
            finally:
                if old_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = old_home


if __name__ == "__main__":
    unittest.main()
