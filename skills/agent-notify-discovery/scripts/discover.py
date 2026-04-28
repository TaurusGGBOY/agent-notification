#!/usr/bin/env python3
"""Discovery for Agent Notify Server.

Supports multiple discovery methods:
1. UDP broadcast (default)
2. Direct HTTP check via --host
3. Subnet scan via --subnet
4. Manual URL via --manual
"""

import socket
import sys
import json
import time
import argparse
import requests

UDP_PORT = 17892
UDP_MSG = b"AGENT_NOTIFY_DISCOVER v1"
UDP_TIMEOUT = 2
HTTP_TIMEOUT = 3


def discover_udp():
    """UDP broadcast discovery."""
    servers = []
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.settimeout(UDP_TIMEOUT)

    try:
        sock.sendto(UDP_MSG, ("255.255.255.255", UDP_PORT))
    except PermissionError:
        print("UDP broadcast permission denied, skipping UDP discovery", file=sys.stderr)
        sock.close()
        return servers
    except Exception as e:
        print(f"UDP broadcast failed: {e}", file=sys.stderr)
        sock.close()
        return servers

    end_time = time.time() + UDP_TIMEOUT
    while time.time() < end_time:
        try:
            data, addr = sock.recvfrom(4096)
            try:
                server = json.loads(data.decode("utf-8"))
                servers.append(server)
            except json.JSONDecodeError:
                pass
        except socket.timeout:
            break

    sock.close()
    return servers


def check_http(url):
    """Check if server responds at URL's /manifest endpoint."""
    try:
        resp = requests.get(f"{url}/manifest", timeout=HTTP_TIMEOUT)
        if resp.status_code == 200:
            return resp.json()
    except Exception:
        pass
    return None


def discover_host(host):
    """Direct HTTP check for single host."""
    url = f"http://{host}:17891"
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
        manifest = check_http(f"http://{ip}:17891")
        if manifest:
            servers.append(manifest)
            print(f"  Found: {ip}", file=sys.stderr)

    return servers


def main():
    parser = argparse.ArgumentParser(description="Discover Agent Notify Servers")
    parser.add_argument("--json", action="store_true", help="Output JSON")
    parser.add_argument("--host", metavar="IP", help="Check single host directly")
    parser.add_argument("--subnet", metavar="NETWORK", help="Scan subnet (e.g. 192.168.31.0/24)")
    parser.add_argument("--manual", metavar="URL", help="Use manual URL directly")

    args = parser.parse_args()

    # Direct host check
    if args.host:
        servers = discover_host(args.host)
    elif args.subnet:
        servers = discover_subnet(args.subnet)
    elif args.manual:
        manifest = check_http(args.manual)
        servers = [manifest] if manifest else []
    else:
        # UDP discovery with HTTP fallback
        servers = discover_udp()

        # Fallback: scan local /24 subnet
        if not servers:
            print("UDP discovery found nothing, falling back to subnet scan...", file=sys.stderr)
            try:
                local_ip = [(s.connect(("8.8.8.8", 80)), s.getsockname()[0], s.close()) for s in [socket.socket(socket.AF_INET, socket.SOCK_STREAM)]][0][1]
                localSubnet = local_ip.rsplit(".", 1)[0] + ".0/24"
                servers = discover_subnet(localSubnet)
            except Exception as e:
                print(f"Subnet scan failed: {e}", file=sys.stderr)

    if args.json:
        print(json.dumps(servers))
    elif servers:
        print(f"Found {len(servers)} server(s):")
        for s in servers:
            print(f"  - {s.get('name', 'unknown')} at {s.get('url', 'unknown')}")
    else:
        print("No servers found.")
        sys.exit(1)


if __name__ == "__main__":
    main()
