import {
  checkForUpdate,
  installAvailableUpdate,
  openMacosNotificationSettings,
  openWindowsNotificationSettings,
  saveConfig,
  sendTestNotification,
  setBroadcastEnabled,
} from "./api";
import { refreshState } from "./service";
import { state } from "./state";
import { applyTheme, getInitialTheme, toggleTheme, type AppTheme } from "./theme";
import { getCurrentWindow } from "@tauri-apps/api/window";
import agentNotifyIconUrl from "./assets/agentnotify-icon.png";

const SKILL_INSTALL_COMMAND = "npx skills add TaurusGGBOY/agent-notification";
let currentTheme: AppTheme = getInitialTheme();

type TranslationKey =
  | "address"
  | "allowNotifications"
  | "broadcast"
  | "broadcastNotice"
  | "cannotConnect"
  | "checkUpdate"
  | "copied"
  | "copy"
  | "copyFailed"
  | "copyInstallCommand"
  | "external"
  | "history"
  | "historyLoadFailed"
  | "installAndRestart"
  | "installCommand"
  | "installCommandHint"
  | "installingUpdate"
  | "language"
  | "nativeUnavailable"
  | "noHistory"
  | "noMessage"
  | "notificationConsole"
  | "notificationHistory"
  | "offline"
  | "online"
  | "openSystemNotificationSettings"
  | "recentNotifications"
  | "refresh"
  | "service"
  | "serviceSubtitle"
  | "start"
  | "stop"
  | "test"
  | "themeToggle"
  | "unknown"
  | "unknownProject"
  | "updateAvailable"
  | "updateCheckFailed"
  | "updateChecking"
  | "updateCurrent"
  | "updateHint"
  | "unsupported"
  | "version";

const translations: Record<"zh" | "en", Record<TranslationKey, string>> = {
  zh: {
    address: "地址",
    allowNotifications: "允许通知",
    broadcast: "广播",
    broadcastNotice: "17891 端口由外部 Agent Notify 服务占用，退出客户端不会关闭该服务。",
    cannotConnect: "无法连接本地通知服务",
    checkUpdate: "检查更新",
    copied: "已复制",
    copy: "复制",
    copyFailed: "复制失败",
    copyInstallCommand: "复制安装 skill 命令",
    external: "外部",
    history: "历史",
    historyLoadFailed: "历史加载失败",
    installAndRestart: "安装并重启",
    installCommand: "安装 skill 命令",
    installCommandHint: "在 Agent 环境中运行后安装通知发现 skill",
    installingUpdate: "正在安装更新，完成后将重启。",
    language: "语言",
    nativeUnavailable: "当前环境不可用",
    noHistory: "暂无通知记录",
    noMessage: "无内容",
    notificationConsole: "通知控制台",
    notificationHistory: "通知历史",
    offline: "离线",
    online: "在线",
    openSystemNotificationSettings: "打开系统通知设置",
    recentNotifications: "最近 3 次通知请求",
    refresh: "刷新",
    service: "服务",
    serviceSubtitle: "本机 Agent 通知服务、广播和历史",
    start: "启动",
    stop: "停止",
    test: "测试",
    themeToggle: "切换明暗模式",
    unknown: "未知",
    unknownProject: "未知项目",
    updateAvailable: "发现新版本",
    updateCheckFailed: "更新检查失败",
    updateChecking: "正在检查更新...",
    updateCurrent: "当前已是最新版本。",
    updateHint: "手动检查 GitHub Release 上的签名更新。",
    unsupported: "不支持",
    version: "版本",
  },
  en: {
    address: "Address",
    allowNotifications: "Notifications",
    broadcast: "Broadcast",
    broadcastNotice: "Port 17891 is used by an external Agent Notify service. Closing the client will not stop it.",
    cannotConnect: "Cannot connect to the local notification service",
    checkUpdate: "Check for updates",
    copied: "Copied",
    copy: "Copy",
    copyFailed: "Copy failed",
    copyInstallCommand: "Copy skill install command",
    external: "External",
    history: "History",
    historyLoadFailed: "Failed to load history",
    installAndRestart: "Install and restart",
    installCommand: "Install skill command",
    installCommandHint: "Run this in an agent environment to install the notification discovery skill",
    installingUpdate: "Installing update. The app will restart when complete.",
    language: "Language",
    nativeUnavailable: "Unavailable in this environment",
    noHistory: "No notifications yet",
    noMessage: "No message",
    notificationConsole: "Notification Console",
    notificationHistory: "Notification History",
    offline: "Offline",
    online: "Online",
    openSystemNotificationSettings: "Open system notification settings",
    recentNotifications: "Last 3 notification requests",
    refresh: "Refresh",
    service: "Service",
    serviceSubtitle: "Local agent notification service, broadcast, and history",
    start: "Start",
    stop: "Stop",
    test: "Test",
    themeToggle: "Toggle theme",
    unknown: "Unknown",
    unknownProject: "Unknown project",
    updateAvailable: "New version available",
    updateCheckFailed: "Update check failed",
    updateChecking: "Checking for updates...",
    updateCurrent: "You are on the latest version.",
    updateHint: "Manually check signed updates from GitHub Releases.",
    unsupported: "Unsupported",
    version: "Version",
  },
};

function currentLanguage(): "zh" | "en" {
  return state.config?.language === "en" ? "en" : "zh";
}

function t(key: TranslationKey): string {
  return translations[currentLanguage()][key];
}

export function render(): void {
  const app = document.querySelector("#app") as HTMLElement;
  if (!app) {
    throw new Error("missing #app root");
  }

  const serviceUrl = state.manifest?.url ?? (currentLanguage() === "en" ? "Waiting for service URL" : "等待服务地址");
  const version = state.manifest?.version ?? t("unknown");
  const broadcastEnabled = state.broadcast?.enabled ?? false;
  const serviceManagedByTauri = state.serviceStatus?.managed_by_tauri !== false;
  const serviceStatusLabel = state.serviceHealthy
    ? serviceManagedByTauri
      ? t("online")
      : t("external")
    : t("offline");
  const updateAvailable = state.updateStatus === "available" && state.updateResult?.available === true;
  const updateLabel = updateStatusLabel();

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
    ? t("nativeUnavailable")
    : nativeNotificationsSupported
      ? ""
      : t("unsupported");
  const nativeNotificationDisabled = !nativeNotificationsSupported || Boolean(state.macosNotificationError || state.windowsNotificationError);
  const nativeNotificationTitle = nativeNotificationDisabled ? t("nativeUnavailable") : t("openSystemNotificationSettings");
  const nextLanguage = currentLanguage() === "zh" ? "en" : "zh";

  app.innerHTML = `
    <section class="app-shell">
      <aside class="sidebar">
        <div class="brand drag-region" data-drag-region>
          <img class="brand-icon" src="${agentNotifyIconUrl}" alt="" aria-hidden="true" />
          <div>
            <strong>AgentNotify</strong>
            <span>${t("notificationConsole")}</span>
          </div>
        </div>

        <section class="side-summary">
          <div>
            <span>${t("service")}</span>
            <strong>${serviceStatusLabel}</strong>
          </div>
          <div class="address-row">
            <span>${t("address")}</span>
            <strong class="side-address">${escapeHtml(serviceUrl)}</strong>
          </div>
          <div class="broadcast-row">
            <span>${t("broadcast")}</span>
            <button class="switch ${broadcastEnabled ? "on" : ""}" data-action="broadcast" ${state.broadcastError ? "disabled" : ""}>
              <i></i>
            </button>
          </div>
          <div class="notification-row">
            <span class="summary-label">
              ${t("allowNotifications")}
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
            <button class="primary" data-action="test">${t("test")}</button>
            <button data-action="refresh">${t("refresh")}</button>
          </div>
        </section>

        <div class="traffic-card">
          <div class="sparkline" aria-hidden="true"><span></span><span></span></div>
          <dl>
            <dt>${t("history")}</dt><dd>${state.history.length}</dd>
            <dt>${t("version")}</dt><dd>v${escapeHtml(version)}</dd>
          </dl>
        </div>
      </aside>

      <main class="main-surface">
        <header class="topbar">
          <div class="topbar-title drag-region" data-drag-region>
            <h1>${t("notificationConsole")}</h1>
            <p>${t("serviceSubtitle")}</p>
          </div>
          <div class="top-actions">
            <button class="language-button" data-action="language" title="${t("language")}">${nextLanguage.toUpperCase()}</button>
            <div class="about-control">
              <button class="icon-button about-button" data-action="about" type="button" aria-label="${currentLanguage() === "en" ? "About AgentNotify" : "关于 AgentNotify"}">
                i
              </button>
              <section class="about-popover" aria-label="${currentLanguage() === "en" ? "About AgentNotify" : "关于 AgentNotify"}">
                <h2>${currentLanguage() === "en" ? "About AgentNotify" : "关于 AgentNotify"}</h2>
                <p>${currentLanguage() === "en" ? "Local agent notification service for task start and completion events." : "本机 Agent 通知服务，用于接收 Agent 任务开始和结束通知。"}</p>
                <dl>
                  <div><dt>${t("version")}</dt><dd>v${escapeHtml(version)}</dd></div>
                  <div><dt>${t("service")}</dt><dd>${escapeHtml(serviceStatusLabel)}</dd></div>
                  <div><dt>${t("address")}</dt><dd>${escapeHtml(serviceUrl)}</dd></div>
                  <div><dt>Skill</dt><dd>${escapeHtml(SKILL_INSTALL_COMMAND)}</dd></div>
                </dl>
                <div class="update-panel">
                  <p>${escapeHtml(updateLabel)}</p>
                  <div>
                    <button
                      class="mini-button"
                      data-action="check-update"
                      type="button"
                      ${state.updateStatus === "checking" || state.updateStatus === "installing" ? "disabled" : ""}
                    >
                      ${t("checkUpdate")}
                    </button>
                    ${
                      updateAvailable
                        ? `<button class="mini-button primary" data-action="install-update" type="button">${t("installAndRestart")}</button>`
                        : ""
                    }
                  </div>
                </div>
              </section>
            </div>
            <button class="icon-button" data-action="theme" title="${t("themeToggle")}">${currentTheme === "light" ? "☾" : "☀"}</button>
          </div>
        </header>

        ${state.error ? `<section class="notice">${escapeHtml(formatError(state.error))}</section>` : ""}
        ${
          state.serviceHealthy && !serviceManagedByTauri
            ? `<section class="notice">${t("broadcastNotice")}</section>`
            : ""
        }

        <section class="dashboard-grid">
          <section class="card install-command-card">
            <div class="card-head">
              <div class="card-icon green">⌘</div>
              <div>
                <h2>${t("installCommand")}</h2>
                <p>${t("installCommandHint")}</p>
              </div>
            </div>
            <div class="command-copy-row">
              <code>${escapeHtml(SKILL_INSTALL_COMMAND)}</code>
              <button class="copy-button" data-action="copy-skill-install-command" type="button" aria-label="${t("copyInstallCommand")}">
                ${t("copy")}
              </button>
            </div>
          </section>

          <section class="card history-card">
            <div class="card-head">
              <div class="card-icon blue">◷</div>
              <div>
                <h2>${t("notificationHistory")}</h2>
                <p>${t("recentNotifications")}</p>
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
    await sendTestNotification("start", currentLanguage());
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

  document.querySelector<HTMLButtonElement>('[data-action="language"]')?.addEventListener("click", async () => {
    if (!state.config) return;
    const nextLanguage = currentLanguage() === "zh" ? "en" : "zh";
    await saveConfig({ ...state.config, language: nextLanguage });
    await refreshState();
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
        await sendTestNotification("start", currentLanguage());
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
      button.textContent = t("copied");
      button.classList.add("copied");
    } catch {
      button.textContent = t("copyFailed");
      button.classList.remove("copied");
    }
    window.setTimeout(() => {
      button.disabled = false;
      button.textContent = t("copy");
      button.classList.remove("copied");
    }, 1400);
  });

  document.querySelector<HTMLButtonElement>('[data-action="check-update"]')?.addEventListener("click", async () => {
    state.updateStatus = "checking";
    state.updateError = "";
    render();
    try {
      state.updateResult = await checkForUpdate();
      state.updateStatus = state.updateResult.available ? "available" : "current";
    } catch (err) {
      state.updateStatus = "error";
      state.updateError = err instanceof Error ? err.message : String(err);
    }
    render();
  });

  document.querySelector<HTMLButtonElement>('[data-action="install-update"]')?.addEventListener("click", async () => {
    state.updateStatus = "installing";
    state.updateError = "";
    render();
    try {
      await installAvailableUpdate();
    } catch (err) {
      state.updateStatus = "error";
      state.updateError = err instanceof Error ? err.message : String(err);
      render();
    }
  });

  document.querySelectorAll<HTMLElement>("[data-drag-region]").forEach((element) => {
    element.addEventListener("mousedown", (event) => {
      if (event.button !== 0) return;
      void getCurrentWindow().startDragging();
    });
  });
}

function updateStatusLabel(): string {
  if (state.updateStatus === "checking") return t("updateChecking");
  if (state.updateStatus === "installing") return t("installingUpdate");
  if (state.updateStatus === "current") return t("updateCurrent");
  if (state.updateStatus === "error") return `${t("updateCheckFailed")}: ${state.updateError || t("unknown")}`;
  if (state.updateResult?.available) {
    return currentLanguage() === "en"
      ? `${t("updateAvailable")} v${state.updateResult.version}; current version is v${state.updateResult.currentVersion}.`
      : `${t("updateAvailable")} v${state.updateResult.version}，当前版本 v${state.updateResult.currentVersion}。`;
  }
  return t("updateHint");
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
    return `<div class="empty-history">${t("historyLoadFailed")}</div>`;
  }
  if (state.history.length === 0) {
    return `<div class="empty-history">${t("noHistory")}</div>`;
  }
  return state.history
    .slice(0, 3)
    .map(
      (item) => `
        <article class="history-item">
          <time>${formatTime(item.time)}</time>
          <strong>${escapeHtml(item.project || item.agent || t("unknownProject"))}</strong>
          <span>${item.event === "start" ? t("start") : t("stop")} · ${escapeHtml(item.message || item.agent || t("noMessage"))}</span>
        </article>
      `,
    )
    .join("");
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--:--";
  return date.toLocaleTimeString(currentLanguage() === "en" ? "en-US" : "zh-CN", { hour: "2-digit", minute: "2-digit" });
}

function formatError(value: string): string {
  if (value === "Failed to fetch") return t("cannotConnect");
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
