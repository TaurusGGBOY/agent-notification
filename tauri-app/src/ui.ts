import { saveConfig, sendTestNotification, type EventName, type NotificationStyle } from "./api";
import { runCommand } from "./commands";
import { refreshState } from "./service";
import { state } from "./state";

const styles: NotificationStyle[] = ["clean", "status-color", "agent-badge", "compact", "custom-card"];
let commandMessage = "";

export function render(): void {
  const app = document.querySelector("#app") as HTMLElement;
  if (!app) {
    throw new Error("missing #app root");
  }

  const config = state.config;
  const currentStyle = config?.notificationStyle ?? "custom-card";
  const enabledEvents = config?.enabledEvents ?? ["start", "stop"];
  const isPaused = enabledEvents.length === 0;

  app.innerHTML = `
    <section class="shell">
      <header class="window-strip">
        <span></span><span></span><span></span>
        <strong>AgentNotify</strong>
      </header>

      <section class="command-row">
        <form class="command-box" data-command-form>
          <span class="search-mark">⌕</span>
          <input name="command" autocomplete="off" placeholder="Ask or search actions..." />
        </form>
        <span class="status ${state.serviceHealthy ? "is-running" : "is-offline"}">
          ${state.serviceHealthy ? "Running" : "Offline"}
        </span>
      </section>

      ${commandMessage ? `<section class="command-message">${escapeHtml(commandMessage)}</section>` : ""}
      ${state.error ? `<section class="notice">${escapeHtml(state.error)}</section>` : ""}

      <section class="workspace">
        <nav class="nav-rail" aria-label="Sections">
          <button class="nav-item active" title="Notifications">●</button>
          <button class="nav-item" title="Preview">▣</button>
          <button class="nav-item" title="Settings">⚙</button>
        </nav>

        <div>
          <section class="panel">
            <div class="section-label">Notification style</div>
            <div class="segmented">
              ${styles
                .map(
                  (style) => `
                    <button class="segment ${style === currentStyle ? "active" : ""}" data-style="${style}">
                      ${labelForStyle(style)}
                    </button>
                  `,
                )
                .join("")}
            </div>
          </section>

          <section class="preview-card">
            <div class="preview-top">
              <div class="avatar">C</div>
              <div class="preview-copy">
                <strong>${currentStyle === "custom-card" ? "Custom card preview" : "Native toast preview"}</strong>
                <span>agent-notification</span>
              </div>
            </div>
            <p>${previewText(currentStyle)}</p>
          </section>

          <section class="toggle-grid">
            ${eventToggle("start", enabledEvents.includes("start"))}
            ${eventToggle("stop", enabledEvents.includes("stop"))}
          </section>
        </div>

        <aside class="context-panel">
          <div class="section-label">Context</div>
          <dl>
            <dt>Mode</dt>
            <dd>${isPaused ? "Paused" : "Active"}</dd>
            <dt>Server</dt>
            <dd>${state.manifest?.url ?? "127.0.0.1:17891"}</dd>
            <dt>Version</dt>
            <dd>${state.manifest?.version ?? "unknown"}</dd>
          </dl>
          <button class="primary block" data-action="test">Test</button>
          <button class="block" data-action="refresh">Refresh</button>
        </aside>
      </section>
    </section>
  `;

  bindEvents();
}

function bindEvents(): void {
  const formEl = document.querySelector<HTMLFormElement>("[data-command-form]");
  formEl?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const input = new FormData(event.currentTarget as HTMLFormElement).get("command")?.toString() ?? "";
    const result = await runCommand(input, state.config);
    commandMessage = result.message;
    if (result.config) state.config = result.config;
    await refreshState();
    (event.currentTarget as HTMLFormElement).reset();
    render();
  });

  document.querySelectorAll<HTMLButtonElement>("[data-style]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.config) return;
      const style = button.dataset.style as NotificationStyle;
      state.config = { ...state.config, notificationStyle: style };
      render();
      await saveConfig(state.config);
      await refreshState();
      render();
    });
  });

  document.querySelectorAll<HTMLButtonElement>("[data-event]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.config) return;
      const event = button.dataset.event as EventName;
      const enabled = new Set(state.config.enabledEvents);
      if (enabled.has(event)) enabled.delete(event);
      else enabled.add(event);
      state.config = { ...state.config, enabledEvents: [...enabled] as EventName[] };
      render();
      await saveConfig(state.config);
      await refreshState();
      render();
    });
  });

  document.querySelector<HTMLButtonElement>('[data-action="test"]')?.addEventListener("click", async () => {
    await sendTestNotification("start");
  });

  document.querySelector<HTMLButtonElement>('[data-action="refresh"]')?.addEventListener("click", async () => {
    await refreshState();
    render();
  });
}

function eventToggle(event: EventName, enabled: boolean): string {
  return `
    <button class="event-toggle ${enabled ? "enabled" : ""}" data-event="${event}">
      <span>${event === "start" ? "Start events" : "Stop events"}</span>
      <strong>${enabled ? "On" : "Off"}</strong>
    </button>
  `;
}

function labelForStyle(style: NotificationStyle): string {
  const labels: Record<NotificationStyle, string> = {
    clean: "Clean",
    "status-color": "Status",
    "agent-badge": "Badge",
    compact: "Compact",
    "custom-card": "Card",
  };
  return labels[style];
}

function previewText(style: NotificationStyle): string {
  if (style === "custom-card") return "Generated PNG hero card inside a native Windows toast.";
  if (style === "compact") return "One-line native toast for high-frequency events.";
  return "Native toast layout managed by the Go notification engine.";
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => {
    const map: Record<string, string> = {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#039;",
    };
    return map[char];
  });
}