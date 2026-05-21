import { restartService, saveConfig, sendTestNotification, type EventName, type NotificationStyle } from "./api";
import { runCommand } from "./commands";
import { refreshState } from "./service";
import { state } from "./state";
import { applyTheme, getInitialTheme, toggleTheme, type AppTheme } from "./theme";

const notificationStyles: NotificationStyle[] = ["clean", "status-color", "agent-badge", "compact", "custom-card"];
let commandMessage = "";
let currentTheme: AppTheme = getInitialTheme();

export function render(): void {
  const app = document.querySelector("#app") as HTMLElement;
  if (!app) {
    throw new Error("missing #app root");
  }

  const config = state.config;
  const currentStyle = config?.notificationStyle ?? "custom-card";
  const enabledEvents = config?.enabledEvents ?? ["start", "stop"];
  const isPaused = enabledEvents.length === 0;
  const serviceUrl = state.manifest?.url ?? "127.0.0.1:17891";
  const version = state.manifest?.version ?? "未知";

  app.innerHTML = `
    <section class="app-shell">
      <aside class="sidebar">
        <div class="brand">
          <div class="brand-icon">A</div>
          <div>
            <strong>AgentNotify</strong>
            <span>通知控制台</span>
          </div>
        </div>

        <section class="side-summary">
          <div>
            <span>服务</span>
            <strong>${state.serviceHealthy ? "在线" : "离线"}</strong>
          </div>
          <div>
            <span>样式</span>
            <strong>${labelForStyle(currentStyle)}</strong>
          </div>
          <div>
            <span>模式</span>
            <strong>${isPaused ? "暂停" : "活跃"}</strong>
          </div>
        </section>

        <div class="traffic-card">
          <div class="sparkline" aria-hidden="true"><span></span><span></span></div>
          <dl>
            <dt>通知</dt><dd>${enabledEvents.length}</dd>
            <dt>状态</dt><dd>${isPaused ? "暂停" : "活跃"}</dd>
          </dl>
        </div>
      </aside>

      <main class="main-surface">
        <header class="topbar">
          <div>
            <h1>通知控制台</h1>
            <p>本机 Agent 通知服务、样式、预览和事件开关</p>
          </div>
          <div class="top-actions">
            <form class="command-box" data-command-form>
              <span class="search-mark">⌕</span>
              <input name="command" autocomplete="off" placeholder="输入命令或搜索操作..." />
            </form>
            <button class="icon-button" data-action="theme" title="切换明暗模式">${currentTheme === "light" ? "☾" : "☀"}</button>
            <span class="service-pill ${state.serviceHealthy ? "is-running" : "is-offline"}">
              ${state.serviceHealthy ? "运行中" : "离线"}
            </span>
          </div>
        </header>

        ${commandMessage ? `<section class="command-message">${escapeHtml(commandMessage)}</section>` : ""}
        ${state.error ? `<section class="notice">${escapeHtml(formatError(state.error))}</section>` : ""}

        <section class="dashboard-grid">
          <section class="card subscription-card">
            <div class="card-head">
              <div class="card-icon blue">☁</div>
              <div>
                <h2>通知配置</h2>
                <p>当前样式：${labelForStyle(currentStyle)}</p>
              </div>
            </div>
            <div class="meta-list">
              <div><span>服务地址</span><strong>${escapeHtml(serviceUrl)}</strong></div>
              <div><span>客户端版本</span><strong>${escapeHtml(version)}</strong></div>
              <div><span>事件模式</span><strong>${isPaused ? "已暂停" : "启动 / 停止"}</strong></div>
            </div>
          </section>

          <section class="card node-card">
            <div class="card-head">
              <div class="card-icon green">◆</div>
              <div>
                <h2>当前通知</h2>
                <p>${state.serviceHealthy ? "本地服务正常响应" : "等待服务连接"}</p>
              </div>
            </div>
            <div class="select-like">
              <span>通知样式</span>
              <strong>${labelForStyle(currentStyle)}</strong>
            </div>
            <div class="select-like">
              <span>通知通道</span>
              <strong>Windows Toast</strong>
            </div>
          </section>

          <section class="card wide-card">
            <div class="card-head">
              <div class="card-icon blue">▤</div>
              <div>
                <h2>通知样式</h2>
                <p>选择系统通知的显示方式</p>
              </div>
            </div>
            <div class="segmented">
              ${notificationStyles
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

          <section class="card preview-panel">
            <div class="card-head">
              <div class="card-icon orange">◴</div>
              <div>
                <h2>通知预览</h2>
                <p>${previewText(currentStyle)}</p>
              </div>
            </div>
            <div class="toast-preview">
              <div class="toast-logo">A</div>
              <div>
                <strong>${currentStyle === "custom-card" ? "自定义卡片预览" : "系统通知预览"}</strong>
                <span>agent-notification</span>
                <p>${previewText(currentStyle)}</p>
              </div>
            </div>
          </section>

          <section class="card event-card">
            <div class="card-head">
              <div class="card-icon blue">↔</div>
              <div>
                <h2>事件开关</h2>
                <p>控制哪些 Agent 生命周期事件会触发通知</p>
              </div>
            </div>
            <div class="toggle-grid">
              ${eventToggle("start", enabledEvents.includes("start"))}
              ${eventToggle("stop", enabledEvents.includes("stop"))}
            </div>
          </section>

          <section class="card actions-card">
            <div class="card-head">
              <div class="card-icon green">✓</div>
              <div>
                <h2>快捷操作</h2>
                <p>测试通知、重启服务或刷新状态</p>
              </div>
            </div>
            <div class="button-row">
              <button class="primary" data-action="test">测试</button>
              <button data-action="restart">重启</button>
              <button data-action="refresh">刷新</button>
            </div>
          </section>
        </section>
      </main>
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

  document.querySelector<HTMLButtonElement>('[data-action="restart"]')?.addEventListener("click", async () => {
    await restartService();
    await refreshState();
    render();
  });

  document.querySelector<HTMLButtonElement>('[data-action="refresh"]')?.addEventListener("click", async () => {
    await refreshState();
    render();
  });

  document.querySelector<HTMLButtonElement>('[data-action="theme"]')?.addEventListener("click", () => {
    currentTheme = toggleTheme(currentTheme);
    applyTheme(currentTheme);
    render();
  });
}

function eventToggle(event: EventName, enabled: boolean): string {
  return `
    <button class="event-toggle ${enabled ? "enabled" : ""}" data-event="${event}">
      <span>${event === "start" ? "启动事件" : "停止事件"}</span>
      <strong>${enabled ? "开" : "关"}</strong>
    </button>
  `;
}

function labelForStyle(style: NotificationStyle): string {
  const labels: Record<NotificationStyle, string> = {
    clean: "简洁",
    "status-color": "状态",
    "agent-badge": "徽章",
    compact: "紧凑",
    "custom-card": "卡片",
  };
  return labels[style];
}

function previewText(style: NotificationStyle): string {
  if (style === "custom-card") return "在系统通知中显示生成的 PNG 卡片。";
  if (style === "compact") return "适合高频事件的一行紧凑通知。";
  return "由 Go 通知引擎管理的系统通知布局。";
}

function formatError(value: string): string {
  if (value === "Failed to fetch") return "无法连接本地通知服务";
  return value;
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
