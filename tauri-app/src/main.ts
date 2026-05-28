import "./styles.css";
import { refreshState } from "./service";
import { applyTheme, getInitialTheme } from "./theme";
import { render } from "./ui";
import { isTauri } from "@tauri-apps/api/core";

async function boot() {
  configureRuntimeScale();
  applyTheme(getInitialTheme());
  render();
  await refreshState();
  render();
}

function configureRuntimeScale(): void {
  if (!isTauri()) {
    return;
  }

  const scaleFactor = Math.max(1, window.devicePixelRatio || 1);
  document.documentElement.dataset.runtime = "tauri";
  document.documentElement.style.setProperty("--runtime-scale", String(1 / scaleFactor));
  document.documentElement.style.setProperty("--runtime-scale-inverse", String(scaleFactor));
}

boot().catch((err) => {
  const app = document.querySelector("#app") as HTMLElement;
  if (app) {
    app.innerHTML = `<pre>${String(err)}</pre>`;
  }
});
