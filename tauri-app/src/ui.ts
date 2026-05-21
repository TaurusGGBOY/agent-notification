import { saveConfig, sendTestNotification, setBroadcastEnabled, type NotificationStyle } from "./api";
import { refreshState } from "./service";
import { state } from "./state";
import { applyTheme, getInitialTheme, toggleTheme, type AppTheme } from "./theme";

const notificationStyles: NotificationStyle[] = ["clean", "status-color", "agent-badge", "compact"];
let currentTheme: AppTheme = getInitialTheme();

export function render(): void {
  const app = document.querySelector("#app") as HTMLElement;
  if (!app) {
    throw new Error("missing #app root");
  }

  const config = state.config;
  const configuredStyle = config?.notificationStyle as NotificationStyle | undefined;
  const currentStyle = configuredStyle && notificationStyles.includes(configuredStyle) ? configuredStyle : "clean";
  const serviceUrl = state.manifest?.url ?? "等待服务地址";
  const version = state.manifest?.version ?? "未知";
  const broadcastEnabled = state.broadcast?.enabled ?? false;

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
            <span>地址</span>
            <strong class="side-address">${escapeHtml(serviceUrl)}</strong>
          </div>
        </section>

        <div class="traffic-card">
          <div class="sparkline" aria-hidden="true"><span></span><span></span></div>
          <dl>
            <dt>历史</dt><dd>${state.history.length}</dd>
            <dt>版本</dt><dd>v${escapeHtml(version)}</dd>
          </dl>
        </div>
      </aside>

      <main class="main-surface">
        <header class="topbar">
          <div>
            <h1>通知控制台</h1>
            <p>本机 Agent 通知服务、样式、广播和历史</p>
          </div>
          <div class="top-actions">
            <button class="icon-button" data-action="theme" title="切换明暗模式">${currentTheme === "light" ? "☾" : "☀"}</button>
            <span class="service-pill ${state.serviceHealthy ? "is-running" : "is-offline"}">
              ${state.serviceHealthy ? "运行中" : "离线"}
            </span>
          </div>
        </header>

        ${state.error ? `<section class="notice">${escapeHtml(formatError(state.error))}</section>` : ""}

        <section class="dashboard-grid">
          <section class="card style-card">
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
            ${previewMarkup(currentStyle)}
          </section>

          <section class="card broadcast-card">
            <div class="card-head">
              <div class="card-icon green">◉</div>
              <div>
                <h2>局域网广播</h2>
                <p>控制 mDNS 发现，HTTP 服务不会停止</p>
              </div>
            </div>
            <div class="broadcast-row">
              <div>
                <strong>${broadcastEnabled ? "广播开启" : "广播关闭"}</strong>
                <span>${state.broadcastError ? "广播状态未知" : "局域网设备可发现服务"}</span>
              </div>
              <button class="switch ${broadcastEnabled ? "on" : ""}" data-action="broadcast" ${state.broadcastError ? "disabled" : ""}>
                <i></i>
              </button>
            </div>
          </section>

          <section class="card history-card">
            <div class="card-head">
              <div class="card-icon blue">◷</div>
              <div>
                <h2>通知历史</h2>
                <p>最近 3 次通知请求</p>
              </div>
            </div>
            <div class="history-list">
              ${historyMarkup()}
            </div>
          </section>

          <section class="card actions-card">
            <div class="card-head">
              <div class="card-icon green">✓</div>
              <div>
                <h2>快捷操作</h2>
                <p>发送测试通知或刷新服务状态</p>
              </div>
            </div>
            <div class="button-row">
              <button class="primary" data-action="test">测试</button>
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

  document.querySelector<HTMLButtonElement>('[data-action="test"]')?.addEventListener("click", async () => {
    await sendTestNotification("start");
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

  document.querySelector<HTMLButtonElement>('[data-action="broadcast"]')?.addEventListener("click", async () => {
    const next = !(state.broadcast?.enabled ?? false);
    state.broadcast = await setBroadcastEnabled(next);
    await refreshState();
    render();
  });
}

function labelForStyle(style: NotificationStyle): string {
  const labels: Record<NotificationStyle, string> = {
    clean: "简洁",
    "status-color": "状态",
    "agent-badge": "徽章",
    compact: "紧凑",
  };
  return labels[style];
}

function previewText(style: NotificationStyle): string {
  if (style === "status-color") return "用状态色条突出启动或停止事件。";
  if (style === "agent-badge") return "突出 Agent 身份和项目来源。";
  if (style === "compact") return "适合高频事件的一行紧凑通知。";
  return "最接近系统默认通知的清爽布局。";
}

function previewMarkup(style: NotificationStyle): string {
  if (style === "status-color") {
    return `
      <div class="toast-preview preview-status">
        <span class="status-bar"></span>
        <div>
          <strong>Agent 已启动</strong>
          <span class="state-tag">START</span>
          <p>AgentNotify 正在处理本地任务通知。</p>
        </div>
      </div>
    `;
  }
  if (style === "agent-badge") {
    return `
      <div class="toast-preview preview-badge">
        <div class="agent-badge">AI</div>
        <div>
          <strong>Claude Code</strong>
          <span>agent-notification</span>
          <p>任务完成，通知已发送到 Windows。</p>
        </div>
      </div>
    `;
  }
  if (style === "compact") {
    return `
      <div class="toast-preview preview-compact">
        <strong>22:30</strong>
        <span>START</span>
        <p>AgentNotify · 测试通知</p>
      </div>
    `;
  }
  return `
    <div class="toast-preview preview-clean">
      <div class="toast-logo">A</div>
      <div>
        <strong>系统通知预览</strong>
        <span>agent-notification</span>
        <p>适合多数事件的标准通知。</p>
      </div>
    </div>
  `;
}

function historyMarkup(): string {
  if (state.historyError) {
    return `<div class="empty-history">历史加载失败</div>`;
  }
  if (state.history.length === 0) {
    return `<div class="empty-history">暂无通知记录</div>`;
  }
  return state.history
    .slice(0, 3)
    .map(
      (item) => `
        <article class="history-item">
          <time>${formatTime(item.time)}</time>
          <strong>${escapeHtml(item.project || item.agent || "未知项目")}</strong>
          <span>${item.event === "start" ? "启动" : "停止"} · ${escapeHtml(item.message || item.agent || "无内容")}</span>
        </article>
      `,
    )
    .join("");
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--:--";
  return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
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
