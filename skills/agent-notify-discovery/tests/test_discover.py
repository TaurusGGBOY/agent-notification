import importlib.util
import socket
import sys
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


if __name__ == "__main__":
    unittest.main()
