import { invoke } from "@tauri-apps/api/core";

export type AppTheme = "light" | "dark";

const STORAGE_KEY = "agentnotify.theme";

export function getInitialTheme(): AppTheme {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") {
    return saved;
  }
  return "light";
}

export function applyTheme(theme: AppTheme): void {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
  localStorage.setItem(STORAGE_KEY, theme);
  void invoke("set_app_theme", { theme }).catch(() => {
    // Browser-only previews do not have the Tauri IPC bridge.
  });
}

export function toggleTheme(theme: AppTheme): AppTheme {
  return theme === "light" ? "dark" : "light";
}
