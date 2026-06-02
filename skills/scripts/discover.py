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
MDNS_GROUP = "224.0.0.251"
MDNS_PORT = 5353


def fetch_manifest(url, timeout=HTTP_TIMEOUT):
    """Fetch server manifest and ensure it contains a URL."""
    manifest_url = url.rstrip("/") + "/manifest"
    req = urllib.request.Request(manifest_url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        payload = json.loads(resp.read().decode("utf-8"))
    payload.setdefault("url", url.rstrip("/"))
    return payload


def encode_dns_name(name):
    payload = bytearray()
    for label in name.rstrip(".").split("."):
        raw = label.encode("utf-8")
        if len(raw) > 63:
            raise ValueError(f"DNS label too long: {label}")
        payload.append(len(raw))
        payload.extend(raw)
    payload.append(0)
    return bytes(payload)


def read_dns_name(packet, offset):
    labels = []
    jumped = False
    end_offset = offset
    seen_offsets = set()

    while True:
        if offset >= len(packet):
            raise ValueError("DNS name extends past packet")
        length = packet[offset]
        if length & 0xC0 == 0xC0:
            if offset + 1 >= len(packet):
                raise ValueError("DNS pointer extends past packet")
            pointer = ((length & 0x3F) << 8) | packet[offset + 1]
            if pointer in seen_offsets:
                raise ValueError("DNS pointer loop")
            seen_offsets.add(pointer)
            if not jumped:
                end_offset = offset + 2
                jumped = True
            offset = pointer
            continue
        if length & 0xC0:
            raise ValueError("Unsupported DNS label type")
        offset += 1
        if length == 0:
            if not jumped:
                end_offset = offset
            break
        if offset + length > len(packet):
            raise ValueError("DNS label extends past packet")
        labels.append(packet[offset : offset + length].decode("utf-8", errors="replace"))
        offset += length

    return ".".join(labels) + ".", end_offset


def parse_mdns_records(packet):
    if len(packet) < 12:
        return []
    qdcount = int.from_bytes(packet[4:6], "big")
    ancount = int.from_bytes(packet[6:8], "big")
    nscount = int.from_bytes(packet[8:10], "big")
    arcount = int.from_bytes(packet[10:12], "big")
    offset = 12

    try:
        for _ in range(qdcount):
            _name, offset = read_dns_name(packet, offset)
            offset += 4

        records = []
        for _ in range(ancount + nscount + arcount):
            name, offset = read_dns_name(packet, offset)
            if offset + 10 > len(packet):
                raise ValueError("DNS record header extends past packet")
            rr_type = int.from_bytes(packet[offset : offset + 2], "big")
            rr_class = int.from_bytes(packet[offset + 2 : offset + 4], "big") & 0x7FFF
            ttl = int.from_bytes(packet[offset + 4 : offset + 8], "big")
            rdlength = int.from_bytes(packet[offset + 8 : offset + 10], "big")
            offset += 10
            rdata_offset = offset
            offset += rdlength
            if offset > len(packet):
                raise ValueError("DNS record data extends past packet")
            records.append((name, rr_type, rr_class, ttl, rdata_offset, rdlength))
    except ValueError:
        return []

    return records


def urls_from_mdns_packet(packet):
    ptr_instances = set()
    srv_targets = {}
    addresses = {}

    for name, rr_type, _rr_class, _ttl, rdata_offset, rdlength in parse_mdns_records(packet):
        try:
            if rr_type == 12:
                ptr_name, _ = read_dns_name(packet, rdata_offset)
                if name.lower() == SERVICE_TYPE:
                    ptr_instances.add(ptr_name)
            elif rr_type == 33 and rdlength >= 7:
                port = int.from_bytes(packet[rdata_offset + 4 : rdata_offset + 6], "big")
                target, _ = read_dns_name(packet, rdata_offset + 6)
                srv_targets[name] = (target, port)
            elif rr_type == 1 and rdlength == 4:
                addresses.setdefault(name, []).append(socket.inet_ntoa(packet[rdata_offset : rdata_offset + 4]))
            elif rr_type == 28 and rdlength == 16:
                addresses.setdefault(name, []).append(socket.inet_ntop(socket.AF_INET6, packet[rdata_offset : rdata_offset + 16]))
        except (OSError, ValueError):
            continue

    urls = []
    for instance in ptr_instances:
        target = srv_targets.get(instance)
        if not target:
            continue
        hostname, port = target
        host = choose_address(addresses.get(hostname, []))
        if not host:
            continue
        if ":" in host and not host.startswith("["):
            host = f"[{host}]"
        url = f"http://{host}:{port or HTTP_PORT}"
        if url not in urls:
            urls.append(url)
    return urls


def mdns_query_packet(service_type=SERVICE_TYPE):
    header = b"\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00"
    question = encode_dns_name(service_type) + b"\x00\x0c\x80\x01"
    return header + question


def discover_mdns_native(timeout=3.0):
    """Dependency-free mDNS/DNS-SD browse for Agent Notify services."""
    servers = []
    seen_urls = set()
    sock = None
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        sock.setsockopt(socket.IPPROTO_IP, socket.IP_MULTICAST_TTL, 1)
        sock.settimeout(0.25)
        sock.sendto(mdns_query_packet(), (MDNS_GROUP, MDNS_PORT))
        end_time = time.time() + timeout
        while time.time() < end_time:
            try:
                packet, _addr = sock.recvfrom(9000)
            except socket.timeout:
                continue
            except OSError:
                break
            for url in urls_from_mdns_packet(packet):
                if url in seen_urls:
                    continue
                seen_urls.add(url)
                try:
                    servers.append(fetch_manifest(url))
                except (OSError, urllib.error.URLError, json.JSONDecodeError) as err:
                    print(f"native mDNS candidate did not answer at {url}: {err}", file=sys.stderr)
    except OSError as err:
        print(f"native mDNS discovery unavailable: {err}", file=sys.stderr)
    finally:
        if sock is not None:
            sock.close()
    return servers


def choose_address(addresses):
    """Choose first non-loopback IPv4 address, then any non-loopback address."""
    non_loopback = [address for address in addresses if not address.startswith(("127.", "::1"))]
    for address in non_loopback:
        if "." in address:
            return address
    return non_loopback[0] if non_loopback else (addresses[0] if addresses else "")


def discover_mdns(timeout=3.0):
    """Discover servers through mDNS/DNS-SD."""
    return discover_mdns_native(timeout=timeout)


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
