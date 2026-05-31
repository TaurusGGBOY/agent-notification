import {
  openMacosNotificationSettings,
  openWindowsNotificationSettings,
  sendTestNotification,
  setBroadcastEnabled,
} from "./api";
import { refreshState } from "./service";
import { state } from "./state";
import { applyTheme, getInitialTheme, toggleTheme, type AppTheme } from "./theme";
import { getCurrentWindow } from "@tauri-apps/api/window";

const SKILL_INSTALL_COMMAND = "npx skills add TaurusGGBOY/agent-notification";
let currentTheme: AppTheme = getInitialTheme();

export function render(): void {
  const app = document.querySelector("#app") as HTMLElement;
  if (!app) {
    throw new Error("missing #app root");
  }

  const serviceUrl = state.manifest?.url ?? "等待服务地址";
  const version = state.manifest?.version ?? "未知";
  const broadcastEnabled = state.broadcast?.enabled ?? false;

  // 统一通知状态：优先使用当前平台的原生通知
  const macosNotificationsSupported = state.macosNotifications?.supported === true;
  const windowsNotificationsSupported = state.windowsNotifications?.supported === true;
  const nativeNotificationsSupported = macosNotificationsSupported || windowsNotificationsSupported;
  const nativeNotificationsEnabled = macosNotificationsSupported
    ? state.macosNotifications?.enabled === true
    : windowsNotificationsSupported
      ? state.windowsNotifications?.enabled === true
      : false;
  const nativeNotificationStatus = state.macosNotificationError || state.windowsNotificationError
    ? "不可用"
    : nativeNotificationsSupported
      ? ""
      : "不支持";
  const nativeNotificationDisabled = !nativeNotificationsSupported || Boolean(state.macosNotificationError || state.windowsNotificationError);
  const nativeNotificationTitle = nativeNotificationDisabled ? "当前环境不可用" : "打开系统通知设置";

  app.innerHTML = `
    <section class="app-shell">
      <aside class="sidebar">
        <div class="brand drag-region" data-drag-region>
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
          <div class="address-row">
            <span>地址</span>
            <strong class="side-address">${escapeHtml(serviceUrl)}</strong>
          </div>
          <div class="broadcast-row">
            <span>广播</span>
            <button class="switch ${broadcastEnabled ? "on" : ""}" data-action="broadcast" ${state.broadcastError ? "disabled" : ""}>
              <i></i>
            </button>
          </div>
          <div class="notification-row">
            <span class="summary-label">
              允许通知
              ${nativeNotificationStatus ? `<em>${escapeHtml(nativeNotificationStatus)}</em>` : ""}
            </span>
            <button
              class="switch ${nativeNotificationsEnabled ? "on" : ""}"
              data-action="native-notifications"
              title="${escapeHtml(nativeNotificationTitle)}"
              aria-pressed="${nativeNotificationsEnabled}"
              ${nativeNotificationDisabled ? "disabled" : ""}
            >
              <i></i>
            </button>
          </div>
          <div class="button-row">
            <button class="primary" data-action="test">测试</button>
            <button data-action="refresh">刷新</button>
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
          <div class="topbar-title drag-region" data-drag-region>
            <h1>通知控制台</h1>
            <p>本机 Agent 通知服务、广播和历史</p>
          </div>
          <div class="top-actions">
            <button class="icon-button" data-action="theme" title="切换明暗模式">${currentTheme === "light" ? "☾" : "☀"}</button>
          </div>
        </header>

        ${state.error ? `<section class="notice">${escapeHtml(formatError(state.error))}</section>` : ""}

        <section class="dashboard-grid">
          <section class="card install-command-card">
            <div class="card-head">
              <div class="card-icon green">⌘</div>
              <div>
                <h2>安装 skill 命令</h2>
                <p>在 Agent 环境中运行后安装通知发现 skill</p>
              </div>
            </div>
            <div class="command-copy-row">
              <code>${escapeHtml(SKILL_INSTALL_COMMAND)}</code>
              <button class="copy-button" data-action="copy-skill-install-command" type="button" aria-label="复制安装 skill 命令">
                复制
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
        </section>
      </main>
    </section>
  `;

  bindEvents();
}

function bindEvents(): void {
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

  document.querySelector<HTMLButtonElement>('[data-action="native-notifications"]')?.addEventListener("click", async () => {
    const isMacos = state.macosNotifications?.supported === true;
    const isWindows = state.windowsNotifications?.supported === true;
    if (!isMacos && !isWindows) return;
    try {
      if (isMacos) {
        await openMacosNotificationSettings();
      } else if (isWindows) {
        await openWindowsNotificationSettings();
      }
      pollNativeNotificationStatus();
    } catch (err) {
      if (isMacos) {
        state.macosNotificationError = err instanceof Error ? err.message : String(err);
      } else {
        state.windowsNotificationError = err instanceof Error ? err.message : String(err);
      }
      render();
    }
  });

  document.querySelector<HTMLButtonElement>('[data-action="copy-skill-install-command"]')?.addEventListener("click", async (event) => {
    const button = event.currentTarget as HTMLButtonElement;
    button.disabled = true;
    try {
      await copySkillInstallCommand();
      button.textContent = "已复制";
      button.classList.add("copied");
    } catch {
      button.textContent = "复制失败";
      button.classList.remove("copied");
    }
    window.setTimeout(() => {
      button.disabled = false;
      button.textContent = "复制";
      button.classList.remove("copied");
    }, 1400);
  });

  document.querySelectorAll<HTMLElement>("[data-drag-region]").forEach((element) => {
    element.addEventListener("mousedown", (event) => {
      if (event.button !== 0) return;
      void getCurrentWindow().startDragging();
    });
  });
}

async function copySkillInstallCommand(): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(SKILL_INSTALL_COMMAND);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = SKILL_INSTALL_COMMAND;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();

  try {
    const copied = document.execCommand("copy");
    if (!copied) {
      throw new Error("copy command failed");
    }
  } finally {
    textarea.remove();
  }
}

function pollNativeNotificationStatus(): void {
  let attempts = 0;
  const refresh = async () => {
    attempts += 1;
    await refreshState();
    render();
    if (attempts < 15) {
      window.setTimeout(() => {
        void refresh();
      }, 1000);
    }
  };

  window.setTimeout(() => {
    void refresh();
  }, 600);
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
