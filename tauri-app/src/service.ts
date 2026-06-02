import {
  getBroadcastStatus,
  getConfig,
  getMacosNotificationStatus,
  getManifest,
  getNotificationHistory,
  getServiceStatus,
  getStartupStatus,
  getWindowsNotificationStatus,
} from "./api";
import { state } from "./state";

const AUTO_REFRESH_INTERVAL_MS = 3000;
let autoRefreshTimer: number | null = null;
let autoRefreshInFlight = false;

export async function refreshState(): Promise<void> {
  state.loading = true;
  state.error = "";
  state.historyError = "";
  state.broadcastError = "";
  state.startupError = "";
  state.windowsNotificationError = "";
  state.macosNotificationError = "";
  try {
    const [config, manifest] = await Promise.all([getConfig(), getManifest()]);
    state.config = config;
    state.manifest = manifest;
    state.serviceHealthy = true;
  } catch (err) {
    state.error = err instanceof Error ? err.message : String(err);
    state.serviceHealthy = false;
  } finally {
    state.loading = false;
  }

  try {
    state.serviceStatus = await getServiceStatus();
  } catch {
    state.serviceStatus = null;
  }

  try {
    state.history = (await getNotificationHistory()).items ?? [];
  } catch (err) {
    state.historyError = err instanceof Error ? err.message : String(err);
  }

  try {
    state.broadcast = await getBroadcastStatus();
  } catch (err) {
    state.broadcast = null;
    state.broadcastError = err instanceof Error ? err.message : String(err);
  }

  try {
    state.startup = await getStartupStatus();
  } catch (err) {
    state.startup = null;
    state.startupError = err instanceof Error ? err.message : String(err);
  }

  try {
    state.windowsNotifications = await getWindowsNotificationStatus();
  } catch (err) {
    state.windowsNotifications = null;
    state.windowsNotificationError = err instanceof Error ? err.message : String(err);
  }

  try {
    state.macosNotifications = await getMacosNotificationStatus();
  } catch (err) {
    state.macosNotifications = null;
    state.macosNotificationError = err instanceof Error ? err.message : String(err);
  }
}

export function startAutoRefresh(render: () => void): void {
  if (autoRefreshTimer !== null) {
    return;
  }

  autoRefreshTimer = window.setInterval(() => {
    if (autoRefreshInFlight) {
      return;
    }
    autoRefreshInFlight = true;
    void refreshState().then(render).finally(() => {
      autoRefreshInFlight = false;
    });
  }, AUTO_REFRESH_INTERVAL_MS);
}
