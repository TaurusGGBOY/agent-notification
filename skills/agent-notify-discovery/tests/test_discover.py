import importlib.util
import socket
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
SCRIPT_PATH = SCRIPT_DIR / "discover.py"
SPEC = importlib.util.spec_from_file_location("agent_notify_discover", SCRIPT_PATH)
discover = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(discover)


class FakeServiceInfo:
    def __init__(self, addresses, port=17891, server="agent-notify.local."):
        self.addresses = addresses
        self.port = port
        self.server = server
        self.properties = {
            b"path": b"/notify",
            b"events": b"start,stop",
        }


class FakeZeroconf:
    def __init__(self, info):
        self.info = info
        self.closed = False

    def get_service_info(self, service_type, name, timeout=1500):
        return self.info

    def close(self):
        self.closed = True


class DiscoverTests(unittest.TestCase):
    def test_sibling_venv_python_is_used_for_reexec(self):
        with tempfile.TemporaryDirectory() as root:
            root_path = Path(root)
            scripts_dir = root_path / "scripts"
            venv_bin = root_path / ".venv" / "bin"
            venv_python = venv_bin / "python"
            system_python = root_path / "system-python"
            scripts_dir.mkdir()
            venv_bin.mkdir(parents=True)
            system_python.write_text("", encoding="utf-8")
            try:
                venv_python.symlink_to(system_python)
            except OSError:
                venv_python.write_text("", encoding="utf-8")

            original_script_dir = discover.SCRIPT_DIR
            try:
                discover.SCRIPT_DIR = scripts_dir

                self.assertEqual(discover.sibling_venv_python(), venv_python)
                self.assertTrue(discover.should_reexec_with_venv("/usr/bin/python3"))
                self.assertFalse(discover.should_reexec_with_venv(str(venv_python)))
                if venv_python.is_symlink():
                    self.assertTrue(discover.should_reexec_with_venv(str(system_python)))
            finally:
                discover.SCRIPT_DIR = original_script_dir

    def test_url_from_service_info_uses_ipv4_address_and_port(self):
        info = FakeServiceInfo([socket.inet_aton("192.168.31.167")], port=17891)

        self.assertEqual(discover.url_from_service_info(info), "http://192.168.31.167:17891")

    def test_discover_mdns_fetches_manifest_for_resolved_service(self):
        fake_info = FakeServiceInfo([socket.inet_aton("192.168.31.167")], port=17891)
        fake_zeroconf = FakeZeroconf(fake_info)
        fetched_urls = []

        class FakeBrowser:
            def __init__(self, zeroconf, service_type, listener):
                self.zeroconf = zeroconf
                self.service_type = service_type
                listener.add_service(
                    zeroconf,
                    service_type,
                    "Agent Notify._agent-notify._tcp.local.",
                )

        original_loader = discover.load_zeroconf
        original_fetch = discover.fetch_manifest
        try:
            discover.load_zeroconf = lambda: (FakeBrowser, FakeZeroconf, lambda: fake_zeroconf)

            def fake_fetch(url, timeout=2.0):
                fetched_urls.append(url)
                return {"name": "Agent Notify Server", "url": url}

            discover.fetch_manifest = fake_fetch

            servers = discover.discover_mdns(timeout=0)
        finally:
            discover.load_zeroconf = original_loader
            discover.fetch_manifest = original_fetch

        self.assertEqual(fetched_urls, ["http://192.168.31.167:17891"])
        self.assertEqual(servers, [{"name": "Agent Notify Server", "url": "http://192.168.31.167:17891"}])
        self.assertTrue(fake_zeroconf.closed)

    def test_dns_sd_fallback_resolves_instance_to_ipv4_manifest(self):
        fetched_urls = []

        def fake_collect(cmd, timeout):
            if cmd[:2] == ["dns-sd", "-B"]:
                return (
                    "Timestamp A/R Flags if Domain Service Type Instance Name\n"
                    "23:01:13 Add 2 18 local. _agent-notify._tcp. Agent Notify PC-202305021618\n"
                )
            if cmd[:2] == ["dns-sd", "-L"]:
                return (
                    "Agent\\032Notify\\032PC-202305021618._agent-notify._tcp.local. "
                    "can be reached at PC-202305021618.local.:17891 (interface 18)\n"
                )
            if cmd[:2] == ["dns-sd", "-G"]:
                return (
                    "PC-202305021618.local. 169.254.102.171 120\n"
                    "PC-202305021618.local. 192.168.31.167 120\n"
                )
            return ""

        original_collect = discover.collect_pty_output
        original_fetch = discover.fetch_manifest
        try:
            discover.collect_pty_output = fake_collect

            def fake_fetch(url, timeout=2.0):
                fetched_urls.append(url)
                return {"name": "Agent Notify Server", "url": url}

            discover.fetch_manifest = fake_fetch

            servers = discover.discover_mdns_dns_sd(timeout=0)
        finally:
            discover.collect_pty_output = original_collect
            discover.fetch_manifest = original_fetch

        self.assertEqual(fetched_urls, ["http://192.168.31.167:17891"])
        self.assertEqual(servers, [{"name": "Agent Notify Server", "url": "http://192.168.31.167:17891"}])

    def test_listener_snapshot_deduplicates_under_lock(self):
        listener = discover.AgentNotifyListener()
        listener.urls = {
            "Agent Notify._agent-notify._tcp.local.": "http://192.168.31.167:17891",
            "Agent Notify duplicate._agent-notify._tcp.local.": "http://192.168.31.167:17891",
        }

        self.assertEqual(listener.snapshot_urls(), ["http://192.168.31.167:17891"])


if __name__ == "__main__":
    unittest.main()
