#!/usr/bin/env python3
"""UDP discovery for Agent Notify Server."""

import socket
import sys
import json
import time

MULTICAST_GROUP = "255.255.255.255"
PORT = 17892
DISCOVERY_MSG = b"AGENT_NOTIFY_DISCOVER v1"
TIMEOUT_SEC = 2


def discover():
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.settimeout(TIMEOUT_SEC)

    servers = []

    try:
        sock.sendto(DISCOVERY_MSG, (MULTICAST_GROUP, PORT))
    except Exception as e:
        print(f"UDP broadcast failed: {e}", file=sys.stderr)
        return servers

    end_time = time.time() + TIMEOUT_SEC
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


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--json":
        servers = discover()
        print(json.dumps(servers))
    else:
        servers = discover()
        if servers:
            print(f"Found {len(servers)} server(s):")
            for s in servers:
                print(f"  - {s.get('name', 'unknown')} at {s.get('url', 'unknown')}")
        else:
            print("No servers found. Use --manual to enter URL manually.")


if __name__ == "__main__":
    main()
