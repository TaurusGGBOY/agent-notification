#!/usr/bin/env python3
"""Discovery for Agent Notify Server."""

import socket
import sys
import json
import time
import argparse
import os
import re
import subprocess
import threading
import urllib.error
import urllib.request
from pathlib import Path

HTTP_PORT = 17891
HTTP_TIMEOUT = 3
SERVICE_TYPE = "_agent-notify._tcp.local."
SCRIPT_DIR = Path(__file__).resolve().parent


def sibling_venv_python():
    """Return installer-created venv Python if present."""
    skill_dir = SCRIPT_DIR.parent
    candidates = [
        skill_dir / ".venv" / "bin" / "python",
        skill_dir / ".venv" / "Scripts" / "python.exe",
    ]
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return None


def should_reexec_with_venv(current_executable=None):
    if os.environ.get("AGENT_NOTIFY_NO_VENV_REEXEC"):
        return False
    if current_executable is None:
        current_executable = sys.executable
    venv_python = sibling_venv_python()
    if not venv_python:
        return False
    return Path(current_executable).absolute() != venv_python.absolute()


def reexec_with_venv_if_available():
    """Use installed venv so default skill commands can import zeroconf."""
    if not should_reexec_with_venv():
        return
    venv_python = sibling_venv_python()
    os.execv(str(venv_python), [str(venv_python), *sys.argv])


def fetch_manifest(url, timeout=HTTP_TIMEOUT):
    """Fetch server manifest and ensure it contains a URL."""
    manifest_url = url.rstrip("/") + "/manifest"
    req = urllib.request.Request(manifest_url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
    payload.setdefault("url", url.rstrip("/"))
    return payload


def collect_pty_output(cmd, timeout):
    """Collect output from tools like dns-sd that buffer under plain pipes."""
    try:
        import pty
        import select
    except ImportError:
        return ""

    try:
        master, slave = pty.openpty()
    except OSError:
        return ""

    chunks = []
    proc = None
    try:
        proc = subprocess.Popen(cmd, stdout=slave, stderr=slave, close_fds=True)
        os.close(slave)
        slave = None
        end_time = time.time() + timeout
        while time.time() < end_time:
            ready, _, _ = select.select([master], [], [], 0.1)
            if not ready:
                if proc.poll() is not None:
                    break
                continue
            try:
                chunk = os.read(master, 4096)
            except OSError:
                break
            if not chunk:
                break
            chunks.append(chunk)
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=1)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=1)
    except (OSError, subprocess.SubprocessError):
        return ""
    finally:
        if slave is not None:
            try:
                os.close(slave)
            except OSError:
                pass
        try:
            os.close(master)
        except OSError:
            pass

    return b"".join(chunks).decode("utf-8", errors="replace")


def parse_dns_sd_browse(output):
    instances = []
    for line in output.splitlines():
        parts = line.split()
        if "Add" not in parts or "_agent-notify._tcp." not in parts:
            continue
        service_index = parts.index("_agent-notify._tcp.")
        instance = " ".join(parts[service_index + 1 :]).strip()
        if instance and instance not in instances:
            instances.append(instance)
    return instances


def parse_dns_sd_resolve(output):
    match = re.search(r"can be reached at\s+(.+?):(\d+)\s", output)
    if not match:
        return None
    return match.group(1), int(match.group(2))


def parse_dns_sd_ipv4(output):
    addresses = []
    for match in re.finditer(r"\b(?:\d{1,3}\.){3}\d{1,3}\b", output):
        address = match.group(0)
        if address.startswith(("127.", "169.254.")):
            continue
        if address not in addresses:
            addresses.append(address)
    return addresses


def discover_mdns_dns_sd(timeout=3.0):
    """Fallback to macOS dns-sd when python zeroconf cannot see an interface."""
    browse_output = collect_pty_output(["dns-sd", "-B", "_agent-notify._tcp", "local."], timeout)
    servers = []
    for instance in parse_dns_sd_browse(browse_output):
        resolve_output = collect_pty_output(["dns-sd", "-L", instance, "_agent-notify._tcp", "local."], 2)
        resolved = parse_dns_sd_resolve(resolve_output)
        if not resolved:
            continue
        hostname, port = resolved
        address_output = collect_pty_output(["dns-sd", "-G", "v4", hostname], 2)
        for address in parse_dns_sd_ipv4(address_output):
            try:
                servers.append(fetch_manifest(f"http://{address}:{port}"))
                break
            except (OSError, urllib.error.URLError, json.JSONDecodeError) as err:
                print(f"dns-sd candidate did not answer at {address}:{port}: {err}", file=sys.stderr)
    return servers


def load_zeroconf():
    """Import zeroconf lazily so manual URL mode works without it."""
    try:
        from zeroconf import ServiceBrowser, ServiceListener, Zeroconf
        from zeroconf import InterfaceChoice
    except ImportError as err:
        raise RuntimeError("zeroconf is required for mDNS discovery; run install-skill.sh") from err

    def zeroconf_factory():
        return Zeroconf(interfaces=InterfaceChoice.All)

    return ServiceBrowser, ServiceListener, zeroconf_factory


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
        self._lock = threading.Lock()

    def add_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def update_service(self, zeroconf, service_type, name):
        self._record(zeroconf, service_type, name)

    def remove_service(self, zeroconf, service_type, name):
        with self._lock:
            self.urls.pop(name, None)

    def _record(self, zeroconf, service_type, name):
        info = zeroconf.get_service_info(service_type, name, timeout=1500)
        if not info:
            return
        url = url_from_service_info(info)
        if url:
            with self._lock:
                self.urls[name] = url

    def snapshot_urls(self):
        with self._lock:
            return list(dict.fromkeys(self.urls.values()))


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
        urls = listener.snapshot_urls()
    finally:
        zeroconf.close()

    for url in urls:
        try:
            servers.append(fetch_manifest(url))
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as err:
            print(f"mDNS candidate did not answer at {url}: {err}", file=sys.stderr)
    if not servers:
        servers = discover_mdns_dns_sd(timeout=timeout)
    return servers


def normalize_manual_url(value):
    if value.startswith("http://") or value.startswith("https://"):
        return value.rstrip("/")
    if ":" in value:
        return f"http://{value}".rstrip("/")
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
    reexec_with_venv_if_available()

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
