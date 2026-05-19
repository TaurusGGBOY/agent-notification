import type { AgentConfig, Manifest } from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  serviceHealthy: boolean;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  serviceHealthy: false,
};