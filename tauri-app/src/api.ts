const BASE_URL = "http://127.0.0.1:17891";

export type NotificationStyle = "clean" | "status-color" | "agent-badge" | "compact" | "custom-card";
export type EventName = "start" | "stop";

export interface AgentConfig {
  notificationStyle: NotificationStyle;
  enabledEvents: EventName[];
  futureOverrides: Record<string, string>;
  _path?: string;
}

export interface Manifest {
  name: string;
  version: string;
  url: string;
  hostname: string;
  protocol: string;
  serviceType: string;
  supportedEvents: EventName[];
  supportedStyles: NotificationStyle[];
}

export async function getConfig(): Promise<AgentConfig> {
  return getJson<AgentConfig>("/config");
}

export async function getManifest(): Promise<Manifest> {
  return getJson<Manifest>("/manifest");
}

export async function saveConfig(config: AgentConfig): Promise<void> {
  const res = await fetch(`${BASE_URL}/settings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  if (!res.ok) {
    throw new Error(`save config failed: ${res.status}`);
  }
}

import { invoke } from "@tauri-apps/api/core";

export async function restartService(): Promise<void> {
  await invoke("restart_service");
}

export async function sendTestNotification(event: EventName = "start"): Promise<void> {
  const res = await fetch(`${BASE_URL}/notify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      agent: "tauri",
      event,
      project: "AgentNotify",
      message: "来自 AgentNotify 的测试通知",
      timestamp: new Date().toISOString(),
      sourcePayload: {},
    }),
  });
  if (!res.ok) {
    throw new Error(`send test notification failed: ${res.status}`);
  }
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`);
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return (await res.json()) as T;
}
