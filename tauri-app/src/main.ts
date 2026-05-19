import "./styles.css";
import { refreshState } from "./service";
import { render } from "./ui";

async function boot() {
  await refreshState();
  render();
}

boot().catch((err) => {
  const app = document.querySelector("#app") as HTMLElement;
  if (app) {
    app.innerHTML = `<pre>${String(err)}</pre>`;
  }
});