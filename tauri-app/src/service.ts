import { getBroadcastStatus, getConfig, getManifest, getNotificationHistory, getWindowsNotificationStatus } from "./api";
import { state } from "./state";

export async function refreshState(): Promise<void> {
  state.loading = true;
  state.error = "";
  state.historyError = "";
  state.broadcastError = "";
  state.windowsNotificationError = "";
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
    state.windowsNotifications = await getWindowsNotificationStatus();
  } catch (err) {
    state.windowsNotifications = null;
    state.windowsNotificationError = err instanceof Error ? err.message : String(err);
  }
}
