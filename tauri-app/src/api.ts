import { invoke } from "@tauri-apps/api/core";
import { getVersion } from "@tauri-apps/api/app";
import { relaunch } from "@tauri-apps/plugin-process";
import { check, type Update } from "@tauri-apps/plugin-updater";

const BASE_URL = "http://127.0.0.1:17891";
let pendingUpdate: Update | null = null;

export type NotificationStyle = "clean";
export type EventName = "start" | "stop";
export type AppLanguage = "zh" | "en";

export interface AgentConfig {
  notificationStyle: NotificationStyle;
  enabledEvents: EventName[];
  language: AppLanguage;
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

export interface BroadcastStatus {
  enabled: boolean;
}

export interface StartupStatus {
  enabled: boolean;
  supported: boolean;
}

export interface ServiceStatus {
  healthy: boolean;
  managed_by_tauri: boolean;
}

export interface WindowsNotificationStatus {
  enabled: boolean;
  supported: boolean;
}

export interface MacosNotificationStatus {
  enabled: boolean;
  supported: boolean;
}

export interface NotificationHistoryItem {
  time: string;
  agent: string;
  event: EventName;
  project: string;
  message: string;
}

export interface NotificationHistory {
  items: NotificationHistoryItem[];
}

export type UpdateCheckResult =
  | {
      available: true;
      version: string;
      currentVersion: string;
      date?: string;
      body?: string;
    }
  | {
      available: false;
    };

export async function getConfig(): Promise<AgentConfig> {
  return getJson<AgentConfig>("/config");
}

export async function getManifest(): Promise<Manifest> {
  return getJson<Manifest>("/manifest");
}

export async function getAppVersion(): Promise<string> {
  return await getVersion();
}

export async function getBroadcastStatus(): Promise<BroadcastStatus> {
  return getJson<BroadcastStatus>("/broadcast");
}

export async function setBroadcastEnabled(enabled: boolean): Promise<BroadcastStatus> {
  const res = await fetch(`${BASE_URL}/broadcast`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) {
    throw new Error(`set broadcast failed: ${res.status}`);
  }
  return (await res.json()) as BroadcastStatus;
}

export async function getStartupStatus(): Promise<StartupStatus> {
  return await invoke<StartupStatus>("startup_status");
}

export async function setStartupEnabled(enabled: boolean): Promise<StartupStatus> {
  return await invoke<StartupStatus>("set_startup_enabled", { enabled });
}

export async function getNotificationHistory(): Promise<NotificationHistory> {
  return getJson<NotificationHistory>("/history");
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

export async function restartService(): Promise<void> {
  await invoke("restart_service");
}

export async function getServiceStatus(): Promise<ServiceStatus> {
  return await invoke<ServiceStatus>("service_status");
}

export async function getWindowsNotificationStatus(): Promise<WindowsNotificationStatus> {
  return await invoke<WindowsNotificationStatus>("windows_notification_status");
}

export async function openWindowsNotificationSettings(): Promise<void> {
  await invoke("open_windows_notification_settings");
}

export async function getMacosNotificationStatus(): Promise<MacosNotificationStatus> {
  return await invoke<MacosNotificationStatus>("macos_notification_status");
}

export async function openMacosNotificationSettings(): Promise<void> {
  await invoke("open_macos_notification_settings");
}

export async function checkForUpdate(): Promise<UpdateCheckResult> {
  pendingUpdate = await check();
  if (!pendingUpdate) {
    return { available: false };
  }

  return {
    available: true,
    version: pendingUpdate.version,
    currentVersion: pendingUpdate.currentVersion,
    date: pendingUpdate.date,
    body: pendingUpdate.body,
  };
}

export async function installAvailableUpdate(): Promise<void> {
  if (!pendingUpdate) {
    pendingUpdate = await check();
  }
  if (!pendingUpdate) {
    throw new Error("no update available");
  }

  await pendingUpdate.downloadAndInstall();
  await relaunch();
}

export async function sendTestNotification(event: EventName = "start", language: AppLanguage = "zh"): Promise<void> {
  const payload = {
    agent: "tauri",
    event,
    project: "AgentNotify",
    message: language === "en" ? "Test notification from AgentNotify" : "来自 AgentNotify 的测试通知",
    language,
    timestamp: new Date().toISOString(),
    sourcePayload: { agentNotifyTest: true },
  };
  const res = await fetch(`${BASE_URL}/notify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
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
