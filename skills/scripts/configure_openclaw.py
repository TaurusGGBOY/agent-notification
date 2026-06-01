#!/usr/bin/env python3
"""Configure OpenClaw lifecycle plugin for agent notifications."""

import argparse
import json
import os
import sys
import urllib.request
from pathlib import Path

PLUGIN_ID = "agent-notify"
PLUGIN_NAME = "Agent Notify"
PLUGIN_VERSION = "1.0.0"


def openclaw_config_path(path=None):
    if path:
        return os.path.abspath(os.path.expanduser(path))
    return os.path.join(os.path.expanduser("~"), ".openclaw", "openclaw.json")


def openclaw_plugin_dir(path=None):
    if path:
        return os.path.abspath(os.path.expanduser(path))
    return os.path.join(os.path.expanduser("~"), ".openclaw", "plugins", PLUGIN_ID)


def load_settings(path):
    if not os.path.exists(path):
        return {}
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def save_settings(path, settings):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(settings, f, indent=2)
        f.write("\n")


def fetch_manifest(url):
    req = urllib.request.Request(url.rstrip("/") + "/manifest", headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))


def dedupe_append(values, value):
    result = []
    for item in values:
        if item not in result:
            result.append(item)
    if value not in result:
        result.append(value)
    return result


def plugin_manifest():
    return {
        "id": PLUGIN_ID,
        "name": PLUGIN_NAME,
        "description": "Send Agent Notify start/stop notifications from OpenClaw lifecycle hooks",
        "version": PLUGIN_VERSION,
        "kind": "integration",
        "configSchema": {
            "type": "object",
            "additionalProperties": False,
            "properties": {
                "serverUrl": {"type": "string", "minLength": 1},
                "events": {
                    "type": "array",
                    "items": {"type": "string", "enum": ["start", "stop"]},
                    "default": ["start", "stop"],
                },
            },
            "required": ["serverUrl"],
        },
    }


def plugin_package():
    return {
        "name": "openclaw-agent-notify",
        "version": PLUGIN_VERSION,
        "type": "module",
        "main": "index.js",
        "private": True,
    }


def plugin_handler():
    return """const PLUGIN_ID = "agent-notify";

function normalizeConfig(raw) {
  const config = raw && typeof raw === "object" ? raw : {};
  const serverUrl = typeof config.serverUrl === "string" ? config.serverUrl.replace(/\\/+$/, "") : "";
  const events = Array.isArray(config.events) ? config.events : ["start", "stop"];
  return {
    serverUrl,
    start: events.includes("start"),
    stop: events.includes("stop"),
  };
}

function firstString(...values) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }
  return "";
}

function projectFromPath(value) {
  if (!value) return "";
  const normalized = value.replace(/[\\\\/]+$/, "");
  const parts = normalized.split(/[\\\\/]+/);
  return parts[parts.length - 1] || "";
}

function buildPayload(eventName, event, ctx) {
  const context = event && typeof event === "object" && event.context && typeof event.context === "object"
    ? event.context
    : {};
  const cwd = firstString(
    context.cwd,
    context.workspaceDir,
    ctx && ctx.workspaceDir,
    ctx && ctx.workspace,
  );
  const project = firstString(
    context.project,
    context.projectName,
    ctx && ctx.project,
    projectFromPath(cwd),
  );
  return {
    agent: "openclaw",
    event: eventName,
    project,
    cwd,
    message: eventName === "start" ? "OpenClaw task started" : "OpenClaw task complete",
    timestamp: new Date().toISOString(),
    sourcePayload: {
      hook: eventName === "start" ? "before_model_resolve" : "agent_end",
      sessionKey: firstString(event && event.sessionKey, context.sessionKey, ctx && ctx.sessionKey),
      sessionId: firstString(event && event.sessionId, context.sessionId, ctx && ctx.sessionId),
      agentId: firstString(event && event.agentId, context.agentId, ctx && ctx.agentId),
    },
  };
}

async function sendNotification(api, config, eventName, event, ctx) {
  if (!config.serverUrl) {
    api.logger.warn(`${PLUGIN_ID}: missing serverUrl; notification skipped`);
    return;
  }

  try {
    const response = await fetch(`${config.serverUrl}/notify`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(buildPayload(eventName, event, ctx)),
      signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) {
      api.logger.warn(`${PLUGIN_ID}: ${eventName} notification failed: HTTP ${response.status}`);
    }
  } catch (err) {
    api.logger.warn(`${PLUGIN_ID}: ${eventName} notification failed: ${String(err)}`);
  }
}

const plugin = {
  id: PLUGIN_ID,
  name: "Agent Notify",
  description: "Send Agent Notify notifications from OpenClaw lifecycle hooks",
  kind: "integration",
  register(api) {
    const config = normalizeConfig(api.pluginConfig);
    if (config.start) {
      api.on("before_model_resolve", (event, ctx) => {
        void sendNotification(api, config, "start", event, ctx);
      });
    }
    if (config.stop) {
      api.on("agent_end", (event, ctx) => {
        void sendNotification(api, config, "stop", event, ctx);
      });
    }
  },
};

export default plugin;
"""


def write_plugin_files(plugin_dir):
    os.makedirs(plugin_dir, exist_ok=True)
    Path(plugin_dir, "openclaw.plugin.json").write_text(
        json.dumps(plugin_manifest(), indent=2) + "\n",
        encoding="utf-8",
    )
    Path(plugin_dir, "package.json").write_text(
        json.dumps(plugin_package(), indent=2) + "\n",
        encoding="utf-8",
    )
    Path(plugin_dir, "index.js").write_text(plugin_handler(), encoding="utf-8")


def configure_plugin_entry(settings, plugin_dir, server_url, events):
    plugins = settings.setdefault("plugins", {})
    plugins["allow"] = dedupe_append(plugins.get("allow", []), PLUGIN_ID)

    load = plugins.setdefault("load", {})
    load["paths"] = dedupe_append(load.get("paths", []), plugin_dir)

    entries = plugins.setdefault("entries", {})
    entries[PLUGIN_ID] = {
        **entries.get(PLUGIN_ID, {}),
        "enabled": True,
        "config": {
            "serverUrl": server_url.rstrip("/"),
            "events": events,
        },
    }
    return settings


def configure_hooks(config_path=None, plugin_dir=None, server_url="", events=None, dry_run=False):
    resolved_config_path = openclaw_config_path(config_path)
    resolved_plugin_dir = openclaw_plugin_dir(plugin_dir)
    normalized_events = list(events or ["start", "stop"])

    settings = load_settings(resolved_config_path)
    configure_plugin_entry(settings, resolved_plugin_dir, server_url, normalized_events)

    if not dry_run:
        write_plugin_files(resolved_plugin_dir)
        save_settings(resolved_config_path, settings)

    return settings


def main():
    parser = argparse.ArgumentParser(description="Configure OpenClaw Agent Notify lifecycle plugin")
    parser.add_argument("--url", required=True, help="Agent Notify server URL")
    parser.add_argument("--events", nargs="+", default=["start", "stop"], choices=["start", "stop"])
    parser.add_argument("--config-path", help="OpenClaw config path; default is ~/.openclaw/openclaw.json")
    parser.add_argument("--plugin-dir", help="OpenClaw plugin directory; default is ~/.openclaw/plugins/agent-notify")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--test", action="store_true", help="Send a stop test notification after saving")
    args = parser.parse_args()

    try:
        manifest = fetch_manifest(args.url)
        print(f"Connected to {manifest.get('name', 'unknown')} v{manifest.get('version', '?')}")
    except Exception as err:
        print(f"Warning: cannot reach {args.url}: {err}", file=sys.stderr)

    settings = configure_hooks(
        config_path=args.config_path,
        plugin_dir=args.plugin_dir,
        server_url=args.url,
        events=args.events,
        dry_run=args.dry_run,
    )

    if args.dry_run:
        print(json.dumps({"plugins": settings.get("plugins", {})}, indent=2))
        return 0

    print(f"OpenClaw plugin installed in {openclaw_plugin_dir(args.plugin_dir)}")
    print(f"OpenClaw config updated in {openclaw_config_path(args.config_path)}")
    print("Restart OpenClaw Gateway for plugin changes to take effect.")

    if args.test:
        import send

        return send.send_notification(
            args.url,
            "openclaw",
            "stop",
            project="test",
            message="Test notification",
            strict=True,
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
