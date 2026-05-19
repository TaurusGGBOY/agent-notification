import "./styles.css";

const app = document.querySelector<HTMLMainElement>("#app");

if (!app) {
  throw new Error("missing #app root");
}

app.innerHTML = `
  <section class="shell">
    <header class="topbar">
      <div>
        <h1>AgentNotify</h1>
        <p>localhost:17891</p>
      </div>
      <span class="status">Starting</span>
    </header>
    <section class="panel">
      <p>Tauri client shell ready.</p>
    </section>
  </section>
`;