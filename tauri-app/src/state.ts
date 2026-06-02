import type {
  AgentConfig,
  BroadcastStatus,
  MacosNotificationStatus,
  Manifest,
  NotificationHistoryItem,
  ServiceStatus,
  UpdateCheckResult,
  WindowsNotificationStatus,
} from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  broadcast: BroadcastStatus | null;
  windowsNotifications: WindowsNotificationStatus | null;
  macosNotifications: MacosNotificationStatus | null;
  serviceStatus: ServiceStatus | null;
  history: NotificationHistoryItem[];
  historyError: string;
  broadcastError: string;
  windowsNotificationError: string;
  macosNotificationError: string;
  serviceHealthy: boolean;
  updateStatus: "idle" | "checking" | "available" | "current" | "installing" | "error";
  updateResult: UpdateCheckResult | null;
  updateError: string;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  broadcast: null,
  windowsNotifications: null,
  macosNotifications: null,
  serviceStatus: null,
  history: [],
  historyError: "",
  broadcastError: "",
  windowsNotificationError: "",
  macosNotificationError: "",
  serviceHealthy: false,
  updateStatus: "idle",
  updateResult: null,
  updateError: "",
};
