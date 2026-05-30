#!/usr/bin/env python3
"""Discovery for Agent Notify Server."""

import socket
import sys
import json
import time
import argparse
import urllib.error
import urllib.request

HTTP_PORT = 17891
HTTP_TIMEOUT = 3
SERVICE_TYPE = "_agent-notify._tcp.local."


def fetch_manifest(url, timeout=HTTP_TIMEOUT):
    """Fetch server manifest and ensure it contains a URL."""
    manifest_url = url.rstrip("/") + "/manifest"
    req = urllib.request.Request(manifest_url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
    payload.setdefault("url", url.rstrip("/"))
    return payload


def load_zeroconf():
    """Import zeroconf lazily so manual URL mode works without it."""
    try:
        from zeroconf import ServiceBrowser, ServiceListener, Zeroconf
    except ImportError as err:
        raise RuntimeError("zeroconf is required for mDNS discovery; run install-skill.sh") from err
    return ServiceBrowser, ServiceListener, Zeroconf


def service_addresses(info):
    """Return parsed service addresses, preferring APIs across zeroconf versions."""
    if hasattr(info, "parsed_addresses"):
        try:
            parsed = list(info.parsed_addresses())
            if parsed:
                return parsed
        except TypeError:
            pass

    addresses = []
    for raw in getattr(info, "addresses", []):
        if isinstance(raw, str):
            addresses.append(raw)
            continue
        try:
            if len(raw) == 4:
                addresses.append(socket.inet_ntoa(raw))
            elif len(raw) == 16:
                addresses.append(socket.inet_ntop(socket.AF_INET6, raw))
        except OSError:
            continue
    return addresses


def choose_address(addresses):
    """Choose first non-loopback IPv4 address, then any non-loopback address."""
    non_loopback = [address for address in addresses if not address.startswith(("127.", "::1"))]
    for address in non_loopback:
        if "." in address:
            return address
    return non_loopback[0] if non_loopback else (addresses[0] if addresses else "")


def url_from_service_info(info):
    """Build HTTP URL from zeroconf service info."""
    host = choose_address(service_addresses(info))
    if not host:
        return None
    if ":" in host and not host.startswith("["):
        host = f"[{host}]"
    return f"http://{host}:{getattr(info, 'port', 0) or HTTP_PORT}"


class AgentNotifyListener:
    def __init__(self):
        self.urls = {}

    def add_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def update_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def remove_service(self, zeroconf, service_type, name):
        self.urls.pop(name, None)

    def _record(self, zeroconf, service_type, name):
        info = zeroconf.get_service_info(service_type, name, timeout=1500)
        if not info:
            return
        url = url_from_service_info(info)
        if url:
            self.urls[name] = url


def discover_mdns(timeout=3.0):
    """Discover servers through mDNS/DNS-SD."""
    servers = []
    try:
        ServiceBrowser, _ServiceListener, Zeroconf = load_zeroconf()
        zeroconf = Zeroconf()
    except Exception as err:
        print(f"mDNS discovery unavailable: {err}", file=sys.stderr)
        return servers

    listener = AgentNotifyListener()
    try:
        ServiceBrowser(zeroconf, SERVICE_TYPE, listener)
        time.sleep(timeout)
        urls = list(dict.fromkeys(listener.urls.values()))
    finally:
        zeroconf.close()

    for url in urls:
        try:
            servers.append(fetch_manifest(url))
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as err:
            print(f"mDNS candidate did not answer at {url}: {err}", file=sys.stderr)
    return servers


def normalize_manual_url(value):
    if value.startswith("http://") or value.startswith("https://"):
        return value.rstrip("/")
    return f"http://{value}:{HTTP_PORT}"


def check_http(url):
    """Check if server responds at URL's /manifest endpoint."""
    try:
        return fetch_manifest(url)
    except Exception:
        return None


def discover_manual(value):
    manifest = check_http(normalize_manual_url(value))
    return [manifest] if manifest else []


def discover_host(host):
    """Direct HTTP check for single host."""
    url = f"http://{host}:{HTTP_PORT}"
    manifest = check_http(url)
    if manifest:
        return [manifest]
    print(f"No server found at {url}", file=sys.stderr)
    return []


def discover_subnet(subnet):
    """Scan subnet for servers. subnet format: 192.168.31.0/24"""
    try:
        import ipaddress

        network = ipaddress.ip_network(subnet, strict=False)
    except ValueError:
        print(f"Invalid subnet: {subnet}", file=sys.stderr)
        return []

    servers = []
    print(f"Scanning {network.num_addresses} hosts on {subnet}...", file=sys.stderr)

    for ip in network.hosts():
        manifest = check_http(f"http://{ip}:{HTTP_PORT}")
        if manifest:
            servers.append(manifest)
            print(f"  Found: {ip}", file=sys.stderr)

    return servers


def discover(args):
    if args.host:
        return discover_host(args.host)
    if args.subnet:
        return discover_subnet(args.subnet)
    if args.manual:
        return discover_manual(args.manual)
    return discover_mdns(timeout=args.timeout)


def main(argv=None):
    parser = argparse.ArgumentParser(description="Discover Agent Notify Servers")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    parser.add_argument("--host", metavar="IP", help="Check single host directly")
    parser.add_argument("--subnet", metavar="NETWORK", help="Scan subnet (e.g. 192.168.31.0/24)")
    parser.add_argument("--manual", metavar="URL", help="Use manual URL directly")
    parser.add_argument("--timeout", type=float, default=3.0, help="mDNS browse timeout in seconds")

    args = parser.parse_args(argv)
    servers = discover(args)

    if args.json:
        print(json.dumps(servers))
    elif servers:
        print(f"Found {len(servers)} server(s):")
        for server in servers:
            print(f"  - {server.get('name', 'unknown')} at {server.get('url', 'unknown')}")
    else:
        print("No servers found.")
        sys.exit(1)


if __name__ == "__main__":
    main()
