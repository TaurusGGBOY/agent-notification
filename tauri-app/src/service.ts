import { getConfig, getManifest } from "./api";
import { state } from "./state";

export async function refreshState(): Promise<void> {
  state.loading = true;
  state.error = "";
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
}