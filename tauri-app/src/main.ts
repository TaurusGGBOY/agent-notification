import "./styles.css";
import { refreshState, startAutoRefresh } from "./service";
import { applyTheme, getInitialTheme } from "./theme";
import { render } from "./ui";

async function boot() {
  applyTheme(getInitialTheme());
  render();
  await refreshState();
  render();
  startAutoRefresh(render);
}

boot().catch((err) => {
  const app = document.querySelector("#app") as HTMLElement;
  if (app) {
    app.innerHTML = `<pre>${String(err)}</pre>`;
  }
});
