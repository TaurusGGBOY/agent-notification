#!/usr/bin/env python3
"""One-command Agent Notify setup for Claude Code, Codex, and OpenClaw."""

import argparse
import json
import sys

import configure_claude
import configure_codex
import configure_openclaw
import send

discover = None
DISCOVERY_TIMEOUT = 8.0


def load_discover():
    global discover
    if discover is None:
        import discover as discover_module

        discover = discover_module
    return discover


def discover_server_url(timeout=DISCOVERY_TIMEOUT):
    try:
        discover_module = load_discover()
        servers = discover_module.discover_mdns(timeout=timeout)
    except Exception as err:
        print(f"Warning: discovery failed: {err}", file=sys.stderr)
        return None

    for server in servers:
        url = server.get("url")
        if url:
            return url.rstrip("/")
    return None


def prompt_for_url(non_interactive):
    if non_interactive or not sys.stdin.isatty():
        return None
    value = input("Agent Notify server URL (example: http://localhost:17891): ").strip()
    return value or None


def resolve_url(args):
    if args.url:
        return args.url.rstrip("/")

    discovered = discover_server_url(getattr(args, "timeout", DISCOVERY_TIMEOUT))
    if discovered:
        print(f"Discovered Agent Notify server: {discovered}")
        return discovered

    return prompt_for_url(args.non_interactive)


def print_preview(agent, path, settings):
    print(f"\n# {agent} -> {path}")
    print(json.dumps({"hooks": settings.get("hooks", {})}, indent=2))


def configure_claude_agent(server_url, events, scope, dry_run):
    path = configure_claude.settings_path(scope)
    settings = configure_claude.load_settings(path)
    configure_claude.configure_hooks(settings, server_url, "claude", events)
    if dry_run:
        print_preview("claude", path, settings)
    else:
        configure_claude.save_settings(path, settings)
        print(f"Claude hooks configured in {path}")


def configure_codex_agent(server_url, events, path_arg, dry_run):
    path = configure_codex.hooks_path(path_arg)
    settings = configure_codex.load_settings(path)
    configure_codex.configure_hooks(settings, server_url, "codex", events)
    if dry_run:
        print_preview("codex", path, settings)
    else:
        configure_codex.save_settings(path, settings)
        print(f"Codex hooks configured in {path}")


def configure_openclaw_agent(server_url, events, config_path, plugin_dir, dry_run):
    settings = configure_openclaw.configure_hooks(
        config_path=config_path,
        plugin_dir=plugin_dir,
        server_url=server_url,
        events=events,
        dry_run=dry_run,
    )
    if dry_run:
        print(f"\n# openclaw -> {configure_openclaw.openclaw_config_path(config_path)}")
        print(json.dumps({"plugins": settings.get("plugins", {})}, indent=2))
    else:
        print(f"OpenClaw Agent Notify plugin installed in {configure_openclaw.openclaw_plugin_dir(plugin_dir)}")
        print(f"OpenClaw config updated in {configure_openclaw.openclaw_config_path(config_path)}")


def send_test_notification(server_url, agents):
    agent = "codex" if "codex" in agents else agents[0]
    return send.send_notification(
        server_url,
        agent,
        "stop",
        project="agent-notify-setup",
        message="Agent Notify setup test",
        strict=True,
    )


def run_setup(args):
    server_url = resolve_url(args)
    if not server_url:
        print("No Agent Notify server URL found. Rerun with --url http://<windows-ip>:17891.", file=sys.stderr)
        return 2

    if "claude" in args.agents:
        configure_claude_agent(server_url, args.events, args.scope, args.dry_run)
    if "codex" in args.agents:
        configure_codex_agent(server_url, args.events, args.codex_path, args.dry_run)
    if "openclaw" in args.agents:
        configure_openclaw_agent(
            server_url,
            args.events,
            args.openclaw_config_path,
            args.openclaw_plugin_dir,
            args.dry_run,
        )

    if args.dry_run:
        print("\nDry run only; no hook files written and no test notification sent.")
        return 0

    print("Restart Claude Code/Codex for hook changes to take effect.")
    if "openclaw" in args.agents:
        print("Restart OpenClaw Gateway for plugin changes to take effect.")
    print("If Codex asks to trust hooks, review ~/.codex/hooks.json before approving.")

    if args.test:
        return send_test_notification(server_url, args.agents)
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description="Configure Agent Notify hooks for Claude Code, Codex, and OpenClaw")
    parser.add_argument("--url", default="", help="Agent Notify server URL; discovers or prompts when omitted")
    parser.add_argument("--agents", nargs="+", default=["claude", "codex", "openclaw"], choices=["claude", "codex", "openclaw"])
    parser.add_argument("--events", nargs="+", default=["start", "stop"], choices=["start", "stop"])
    parser.add_argument("--scope", choices=["user", "project"], default="user", help="Claude hook scope")
    parser.add_argument("--codex-path", help="Codex hooks.json path; default is ~/.codex/hooks.json")
    parser.add_argument("--openclaw-config-path", help="OpenClaw config path; default is ~/.openclaw/openclaw.json")
    parser.add_argument("--openclaw-plugin-dir", help="OpenClaw plugin directory; default is ~/.openclaw/plugins/agent-notify")
    parser.add_argument("--dry-run", action="store_true", help="Print generated settings without writing files")
    parser.add_argument("--test", action="store_true", help="Send a strict test notification after saving hooks")
    parser.add_argument("--non-interactive", action="store_true", help="Do not prompt for URL if discovery fails")
    parser.add_argument("--timeout", type=float, default=DISCOVERY_TIMEOUT, help="mDNS discovery timeout in seconds")
    return parser


def main(argv=None):
    parser = build_parser()
    return run_setup(parser.parse_args(argv))


if __name__ == "__main__":
    raise SystemExit(main())
