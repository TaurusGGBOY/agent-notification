import type { AgentConfig, BroadcastStatus, MacosNotificationStatus, Manifest, NotificationHistoryItem, WindowsNotificationStatus } from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  broadcast: BroadcastStatus | null;
  windowsNotifications: WindowsNotificationStatus | null;
  macosNotifications: MacosNotificationStatus | null;
  history: NotificationHistoryItem[];
  historyError: string;
  broadcastError: string;
  windowsNotificationError: string;
  macosNotificationError: string;
  serviceHealthy: boolean;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  broadcast: null,
  windowsNotifications: null,
  macosNotifications: null,
  history: [],
  historyError: "",
  broadcastError: "",
  windowsNotificationError: "",
  macosNotificationError: "",
  serviceHealthy: false,
};
