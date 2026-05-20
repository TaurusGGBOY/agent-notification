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
}

export function toggleTheme(theme: AppTheme): AppTheme {
  return theme === "light" ? "dark" : "light";
}