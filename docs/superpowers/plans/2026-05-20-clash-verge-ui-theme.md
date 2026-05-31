# Clash Verge UI Theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the AgentNotify Tauri client into a Clash Verge style Windows control-panel UI with persistent light/dark mode switching.

**Architecture:** Keep the current static TypeScript renderer and Go sidecar flow. Add a small frontend-only theme module that stores the selected theme in `localStorage`, applies `data-theme` to `document.documentElement`, and exposes a toggle used by `ui.ts`. Refactor `ui.ts` into a card dashboard layout and replace `styles.css` tokens/components with a Clash Verge inspired light-default design plus dark mode.

**Tech Stack:** Tauri v2, TypeScript, Vite, CSS custom properties, localStorage, existing Windows screenshot skill.

---

## File Structure

- Create: `tauri-app/src/theme.ts`
  - Owns theme type, localStorage key, initial theme detection, DOM application, and toggle helper.
- Modify: `tauri-app/src/main.ts`
  - Apply the saved theme before the first render.
- Modify: `tauri-app/src/ui.ts`
  - Replace the current dark agent UI with a Clash Verge style shell: left nav, top bar, dashboard cards, preview card, status/actions card, theme toggle.
- Modify: `tauri-app/src/styles.css`
  - Replace visual tokens and layout styles with light-default and `[data-theme="dark"]` variants.
- Verify only: `tauri-app/src-tauri/tauri.conf.json`
  - Confirm window remains `960x540` and visible.
- Verify only: `skills/windows-ui-screenshot/scripts/capture_windows_ui.py`
  - Use existing screenshot workflow after build.

---

### Task 1: Add Persistent Theme State

**Files:**
- Create: `tauri-app/src/theme.ts`
- Modify: `tauri-app/src/main.ts`

- [ ] **Step 1: Create the theme helper**

Create `tauri-app/src/theme.ts` with this content:

```ts
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
```

- [ ] **Step 2: Apply theme before first render**

Modify `tauri-app/src/main.ts` to import and apply the theme before `refreshState()`:

```ts
import "./styles.css";
import { refreshState } from "./service";
import { applyTheme, getInitialTheme } from "./theme";
import { render } from "./ui";

async function boot() {
  applyTheme(getInitialTheme());
  await refreshState();
  render();
}

boot().catch((err) => {
  const app = document.querySelector("#app") as HTMLElement;
  if (app) {
    app.innerHTML = `<pre>${String(err)}</pre>`;
  }
});
```

- [ ] **Step 3: Run local frontend build**

Run:

```bash
cd tauri-app
npm run build
```

Expected:

```text
✓ built
```

- [ ] **Step 4: Commit**

```bash
git add tauri-app/src/theme.ts tauri-app/src/main.ts
git commit -m "feat: add persistent app theme"
```

---

### Task 2: Refactor UI Markup to Clash Verge Dashboard

**Files:**
- Modify: `tauri-app/src/ui.ts`

- [ ] **Step 1: Replace imports and add theme state**

At the top of `tauri-app/src/ui.ts`, add theme imports and state:

```ts
import { restartService, saveConfig, sendTestNotification, type EventName, type NotificationStyle } from "./api";
import { runCommand } from "./commands";
import { refreshState } from "./service";
import { state } from "./state";
import { applyTheme, getInitialTheme, toggleTheme, type AppTheme } from "./theme";

const notificationStyles: NotificationStyle[] = ["clean", "status-color", "agent-badge", "compact", "custom-card"];
let commandMessage = "";
let currentTheme: AppTheme = getInitialTheme();
```

Remove the old `const styles: NotificationStyle[] = ...` line.

- [ ] **Step 2: Replace `render()` shell markup**

Replace the full `render()` function in `tauri-app/src/ui.ts` with:

```ts
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

        <nav class="side-nav" aria-label="主导航">
          ${navItem("⌂", "首页", true)}
          ${navItem("◈", "通知")}
          ${navItem("▣", "预览")}
          ${navItem("⚙", "设置")}
        </nav>

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
            <h1>首页</h1>
            <p>管理本机 Agent 通知服务、样式和事件开关</p>
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
            <div class="progress-row">
              <span>${isPaused ? "0" : "100"}%</span>
              <div class="progress-track"><i style="width: ${isPaused ? "0" : "100"}%"></i></div>
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
```

- [ ] **Step 3: Add `navItem()` helper**

Add this helper below `bindEvents()` or near other render helpers:

```ts
function navItem(icon: string, label: string, active = false): string {
  return `
    <button class="nav-item ${active ? "active" : ""}" type="button">
      <span>${icon}</span>
      <strong>${label}</strong>
    </button>
  `;
}
```

- [ ] **Step 4: Update theme button event binding**

Inside `bindEvents()`, after the refresh handler, add:

```ts
  document.querySelector<HTMLButtonElement>('[data-action="theme"]')?.addEventListener("click", () => {
    currentTheme = toggleTheme(currentTheme);
    applyTheme(currentTheme);
    render();
  });
```

- [ ] **Step 5: Run local frontend build**

Run:

```bash
cd tauri-app
npm run build
```

Expected:

```text
✓ built
```

- [ ] **Step 6: Commit**

```bash
git add tauri-app/src/ui.ts
git commit -m "feat: rebuild client dashboard layout"
```

---

### Task 3: Replace CSS With Clash Verge Visual System

**Files:**
- Modify: `tauri-app/src/styles.css`

- [ ] **Step 1: Replace `styles.css`**

Replace the full file with:

```css
:root {
  color-scheme: light;
  --bg: #eef1f5;
  --surface: #f7f8fa;
  --sidebar: #f4f6f9;
  --card: #ffffff;
  --card-soft: #eef6ff;
  --text: #111827;
  --muted: #6b7280;
  --subtle: #8a94a6;
  --border: #e1e5eb;
  --border-strong: #d0d6df;
  --primary: #0f7cff;
  --primary-hover: #006fee;
  --primary-soft: #e7f1ff;
  --success: #00a65a;
  --success-soft: #e3f7ec;
  --warning: #ff7a45;
  --warning-soft: #fff0e8;
  --shadow: 0 10px 26px rgb(15 23 42 / 8%);
  --radius: 10px;
  font-family: "Segoe UI", system-ui, sans-serif;
  background: var(--bg);
  color: var(--text);
}

:root[data-theme="dark"] {
  color-scheme: dark;
  --bg: #10151d;
  --surface: #151b24;
  --sidebar: #121821;
  --card: #1b2330;
  --card-soft: #142235;
  --text: #f4f7fb;
  --muted: #a8b1c2;
  --subtle: #7f8aa0;
  --border: #2a3443;
  --border-strong: #3a4658;
  --primary: #4d9cff;
  --primary-hover: #78b4ff;
  --primary-soft: #18375c;
  --success: #45d483;
  --success-soft: #173b2a;
  --warning: #ff9b6b;
  --warning-soft: #3a241b;
  --shadow: 0 12px 30px rgb(0 0 0 / 22%);
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--bg);
}

button,
input {
  font: inherit;
}

button {
  cursor: pointer;
}

.app-shell {
  display: grid;
  grid-template-columns: 246px minmax(0, 1fr);
  width: 100vw;
  height: 100vh;
  background: var(--surface);
  color: var(--text);
}

.sidebar {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 22px 12px 14px;
  background: var(--sidebar);
  border-right: 1px solid var(--border);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px 26px;
}

.brand-icon,
.card-icon,
.toast-logo {
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  font-weight: 800;
}

.brand-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, #0f7cff, #62b4ff);
  color: #fff;
}

.brand strong {
  display: block;
  font-size: 20px;
  line-height: 1.1;
}

.brand span {
  display: block;
  margin-top: 4px;
  color: var(--muted);
  font-size: 12px;
}

.side-nav {
  display: grid;
  gap: 8px;
}

.nav-item {
  display: grid;
  grid-template-columns: 40px 1fr;
  align-items: center;
  min-height: 60px;
  border: 0;
  border-radius: 8px;
  padding: 0 12px;
  background: transparent;
  color: var(--text);
  text-align: left;
}

.nav-item span {
  font-size: 22px;
}

.nav-item strong {
  font-size: 17px;
  letter-spacing: 0;
}

.nav-item.active {
  background: var(--primary-soft);
  color: var(--text);
}

.traffic-card {
  margin-top: auto;
  padding: 12px 10px;
  border-top: 1px solid var(--border);
}

.sparkline {
  position: relative;
  height: 42px;
  overflow: hidden;
}

.sparkline span {
  position: absolute;
  inset: 10px 0 0;
  border-top: 3px solid var(--primary);
  border-radius: 50%;
  transform: skewX(-18deg);
}

.sparkline span + span {
  inset: 18px 0 0;
  border-color: var(--warning);
  opacity: .75;
}

.traffic-card dl {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px 12px;
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.traffic-card dd {
  margin: 0;
  color: var(--text);
  font-weight: 700;
}

.main-surface {
  min-width: 0;
  overflow: auto;
  padding: 26px 24px 24px;
}

.topbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.topbar h1 {
  margin: 0;
  font-size: 28px;
  line-height: 1.15;
}

.topbar p {
  margin: 6px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.top-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.command-box {
  display: flex;
  align-items: center;
  width: 270px;
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
}

.search-mark {
  color: var(--muted);
  margin-right: 8px;
}

.command-box input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font-size: 13px;
}

.icon-button {
  width: 38px;
  height: 38px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--text);
  font-size: 17px;
}

.service-pill {
  display: inline-flex;
  align-items: center;
  height: 38px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--card);
  color: var(--muted);
  font-size: 13px;
  white-space: nowrap;
}

.service-pill.is-running {
  border-color: color-mix(in srgb, var(--success) 35%, transparent);
  background: var(--success-soft);
  color: var(--success);
  font-weight: 700;
}

.command-message,
.notice {
  margin-bottom: 14px;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 13px;
}

.command-message {
  border: 1px solid color-mix(in srgb, var(--primary) 32%, transparent);
  background: var(--primary-soft);
  color: var(--text);
}

.notice {
  border: 1px solid #ffc6c6;
  background: #fff1f1;
  color: #9b1c1c;
}

:root[data-theme="dark"] .notice {
  border-color: #733;
  background: #351b1b;
  color: #ffb4b4;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 14px;
}

.card {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--card);
  box-shadow: var(--shadow);
  padding: 20px;
}

.wide-card,
.event-card,
.actions-card {
  box-shadow: none;
}

.card-head {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 18px;
}

.card-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  font-size: 22px;
}

.card-icon.blue {
  background: var(--primary-soft);
  color: var(--primary);
}

.card-icon.green {
  background: var(--success-soft);
  color: var(--success);
}

.card-icon.orange {
  background: var(--warning-soft);
  color: var(--warning);
}

.card h2 {
  margin: 0;
  font-size: 20px;
  line-height: 1.2;
}

.card p {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.meta-list {
  display: grid;
  gap: 12px;
  color: var(--muted);
  font-size: 13px;
}

.meta-list div,
.select-like {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.meta-list strong,
.select-like strong {
  min-width: 0;
  overflow: hidden;
  color: var(--text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-row {
  margin-top: 16px;
  color: var(--muted);
  font-size: 13px;
}

.progress-track {
  height: 9px;
  margin-top: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--primary-soft);
}

.progress-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--primary);
}

.select-like {
  min-height: 52px;
  margin-top: 10px;
  padding: 0 14px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--card);
}

.select-like span {
  color: var(--muted);
  font-size: 13px;
}

.segmented {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.segment {
  min-height: 50px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: color-mix(in srgb, var(--border) 45%, transparent);
  color: var(--text);
  font-weight: 700;
}

.segment.active {
  border-color: var(--primary);
  background: var(--primary);
  color: #fff;
}

.toast-preview {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 100px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card-soft);
}

.toast-logo {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  background: var(--success);
  color: #fff;
}

.toast-preview strong,
.toast-preview span {
  display: block;
}

.toast-preview span {
  margin-top: 3px;
  color: var(--muted);
  font-size: 13px;
}

.toast-preview p {
  margin-top: 10px;
}

.toggle-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.event-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 54px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 14px;
  background: var(--card-soft);
  color: var(--text);
}

.event-toggle strong {
  color: var(--muted);
}

.event-toggle.enabled strong {
  color: var(--success);
}

.button-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.button-row button {
  min-height: 42px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: color-mix(in srgb, var(--border) 45%, transparent);
  color: var(--text);
  font-weight: 700;
}

.button-row button.primary {
  border-color: var(--primary);
  background: var(--primary);
  color: #fff;
}

@media (max-width: 820px) {
  .app-shell {
    grid-template-columns: 82px minmax(0, 1fr);
  }

  .brand div:not(.brand-icon),
  .nav-item strong,
  .traffic-card dl {
    display: none;
  }

  .nav-item {
    grid-template-columns: 1fr;
    justify-items: center;
    padding: 0;
  }

  .topbar {
    flex-direction: column;
  }

  .top-actions {
    width: 100%;
  }

  .command-box {
    flex: 1;
  }

  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd tauri-app
npm run build
```

Expected:

```text
✓ built
```

- [ ] **Step 3: Commit**

```bash
git add tauri-app/src/styles.css
git commit -m "style: apply clash verge inspired theme"
```

---

### Task 4: Check Window Ratio and Local Behavior

**Files:**
- Verify: `tauri-app/src-tauri/tauri.conf.json`

- [ ] **Step 1: Confirm window config remains 16:9**

Run:

```bash
grep -n '"width"\\|"height"\\|"visible"' tauri-app/src-tauri/tauri.conf.json
```

Expected:

```text
"width": 960,
"height": 540,
"visible": true
```

- [ ] **Step 2: Run TypeScript build**

Run:

```bash
cd tauri-app
npm run build
```

Expected:

```text
✓ built
```

- [ ] **Step 3: Commit**

If no file changed in this task, do not commit. If window config had to be restored, run:

```bash
git add tauri-app/src-tauri/tauri.conf.json tauri-app/src-tauri/src/main.rs
git commit -m "fix: keep client window at 16 by 9"
```

---

### Task 5: Build and Verify on Windows

**Files:**
- Verify build output under `tauri-app/src-tauri/target/release/`
- Verify screenshot output under `/tmp/agentnotify-clash-verge-ui.png`

- [ ] **Step 1: Sync changed frontend files to Windows**

Run from repo root:

```bash
rsync -av tauri-app/src/main.ts tauri-app/src/theme.ts tauri-app/src/ui.ts tauri-app/src/styles.css <user>@<host>:'D:/project/agent-notification/tauri-app/src/'
```

Expected:

```text
sent
```

- [ ] **Step 2: Build Windows release**

Run:

```bash
ssh <user>@<host> "powershell -NoProfile -Command \"cd D:\\project\\agent-notification\\tauri-app; npm run tauri:build\""
```

Expected:

```text
Built application at: D:\project\agent-notification\tauri-app\src-tauri\target\release\agent-notify.exe
Finished 2 bundles
```

- [ ] **Step 3: Stop old Windows processes and clear app cache**

Run:

```bash
ENC=$(python3 - <<'PY'
import base64
script = r'''
Get-Process agent-notify,agent-notify-server -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item "$env:LOCALAPPDATA\com.agentnotify.client" -Recurse -Force -ErrorAction SilentlyContinue
'''
print(base64.b64encode(script.encode('utf-16le')).decode())
PY
)
ssh <user>@<host> "powershell -NoProfile -EncodedCommand $ENC"
```

Expected: command exits with code `0`.

- [ ] **Step 4: Capture Windows screenshot**

Run:

```bash
python3 skills/windows-ui-screenshot/scripts/capture_windows_ui.py \
  --host <user>@<host> \
  --exe 'D:\project\agent-notification\tauri-app\src-tauri\target\release\agent-notify.exe' \
  --process agent-notify \
  --out /tmp/agentnotify-clash-verge-ui.png
```

Expected:

```text
screenshot=/tmp/agentnotify-clash-verge-ui.png
```

- [ ] **Step 5: Inspect screenshot**

Open `/tmp/agentnotify-clash-verge-ui.png` and verify:

- Window is 16:9.
- UI is mostly light by default.
- Left nav uses large icon + Chinese label rows.
- Main content uses white cards on gray background.
- Blue is the primary action/selected color.
- Theme toggle exists in top-right action area.
- Clicking theme toggle changes to dark mode and survives app reload.
- No text overlaps at 960x540.
- Buttons and cards remain readable in dark mode.

- [ ] **Step 6: Commit verification-ready implementation**

```bash
git status --short
git add tauri-app/src/theme.ts tauri-app/src/main.ts tauri-app/src/ui.ts tauri-app/src/styles.css
git commit -m "feat: add clash verge style client ui"
```

---

## Self-Review

- Spec coverage: The plan covers Clash Verge style refactor, normal/light mode, dark mode, persistent theme switch, 16:9 preservation, Windows build, and screenshot verification.
- Placeholder scan: No `TBD`, `TODO`, or undefined future work remains.
- Type consistency: Theme type is `AppTheme`; helpers are `getInitialTheme`, `applyTheme`, and `toggleTheme`; UI imports match the new file.
- Scope: This plan intentionally avoids backend changes. The existing notification flow, config save flow, sidecar startup, CORS behavior, and toast rendering remain unchanged.
