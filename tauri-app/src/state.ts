import type { AgentConfig, BroadcastStatus, Manifest, NotificationHistoryItem } from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  broadcast: BroadcastStatus | null;
  history: NotificationHistoryItem[];
  historyError: string;
  broadcastError: string;
  serviceHealthy: boolean;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  broadcast: null,
  history: [],
  historyError: "",
  broadcastError: "",
  serviceHealthy: false,
};
