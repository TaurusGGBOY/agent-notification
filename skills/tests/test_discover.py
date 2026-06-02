import importlib.util
import struct
import socket
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
SCRIPT_PATH = SCRIPT_DIR / "discover.py"
SPEC = importlib.util.spec_from_file_location("agent_notify_discover", SCRIPT_PATH)
discover = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(discover)


def dns_name(name):
    encoded = b""
    for label in name.rstrip(".").split("."):
        encoded += bytes([len(label)]) + label.encode("utf-8")
    return encoded + b"\x00"


def dns_rr(name, rr_type, payload, ttl=120, rr_class=1):
    return dns_name(name) + struct.pack("!HHIH", rr_type, rr_class, ttl, len(payload)) + payload


class DiscoverTests(unittest.TestCase):
    def test_native_mdns_query_requests_unicast_response(self):
        packet = discover.mdns_query_packet()

        self.assertTrue(packet.endswith(b"\x00\x0c\x80\x01"))

    def test_native_mdns_response_resolves_service_url(self):
        instance = "Agent Notify TEST._agent-notify._tcp.local."
        hostname = "TEST.local."
        header = struct.pack("!HHHHHH", 0, 0x8400, 0, 4, 0, 0)
        packet = b"".join(
            [
                header,
                dns_rr("_agent-notify._tcp.local.", 12, dns_name(instance)),
                dns_rr(instance, 33, struct.pack("!HHH", 0, 0, 17891) + dns_name(hostname)),
                dns_rr(hostname, 1, socket.inet_aton("192.168.1.100")),
                dns_rr(instance, 16, b"\x05path=/notify"),
            ]
        )

        urls = discover.urls_from_mdns_packet(packet)

        self.assertEqual(urls, ["http://192.168.1.100:17891"])

    def test_native_mdns_discovery_fetches_manifest(self):
        class FakeSocket:
            def __init__(self, *args, **kwargs):
                self.sent = []
                self.closed = False
                self.received = False

            def setsockopt(self, *args):
                pass

            def settimeout(self, timeout):
                pass

            def sendto(self, payload, address):
                self.sent.append((payload, address))

            def recvfrom(self, size):
                if not self.received:
                    self.received = True
                    return b"packet", ("192.168.1.100", 5353)
                raise socket.timeout()

            def close(self):
                self.closed = True

        original_socket = discover.socket.socket
        original_parse = discover.urls_from_mdns_packet
        original_fetch = discover.fetch_manifest
        fake_socket = FakeSocket()
        try:
            discover.socket.socket = lambda *args, **kwargs: fake_socket
            discover.urls_from_mdns_packet = lambda packet: ["http://192.168.1.100:17891"]
            discover.fetch_manifest = lambda url, timeout=2.0: {"name": "Agent Notify Server", "url": url}

            servers = discover.discover_mdns_native(timeout=0.01)
        finally:
            discover.socket.socket = original_socket
            discover.urls_from_mdns_packet = original_parse
            discover.fetch_manifest = original_fetch

        self.assertEqual(servers, [{"name": "Agent Notify Server", "url": "http://192.168.1.100:17891"}])
        self.assertEqual(fake_socket.sent[0][1], ("224.0.0.251", 5353))
        self.assertTrue(fake_socket.closed)

    def test_manual_url_preserves_explicit_port(self):
        self.assertEqual(discover.normalize_manual_url("localhost"), "http://localhost:17891")
        self.assertEqual(discover.normalize_manual_url("localhost:17891"), "http://localhost:17891")
        self.assertEqual(discover.normalize_manual_url("http://localhost:17891/"), "http://localhost:17891")


if __name__ == "__main__":
    unittest.main()
