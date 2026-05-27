import type { AgentConfig, BroadcastStatus, Manifest, NotificationHistoryItem, WindowsNotificationStatus } from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  broadcast: BroadcastStatus | null;
  windowsNotifications: WindowsNotificationStatus | null;
  history: NotificationHistoryItem[];
  historyError: string;
  broadcastError: string;
  windowsNotificationError: string;
  serviceHealthy: boolean;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  broadcast: null,
  windowsNotifications: null,
  history: [],
  historyError: "",
  broadcastError: "",
  windowsNotificationError: "",
  serviceHealthy: false,
};
