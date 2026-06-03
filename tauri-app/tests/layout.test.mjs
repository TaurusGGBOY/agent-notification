import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const readProjectFile = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("Tauri shell fills the full webview viewport", async () => {
  const styles = await readProjectFile("src/styles.css");
  const main = await readProjectFile("src/main.ts");

  assert.match(styles, /\.app-shell\s*{[^}]*width:\s*100vw;[^}]*height:\s*100vh;/s);
  assert.doesNotMatch(`${styles}\n${main}`, /runtime-scale/);
});

test("main window uses the 1200x675 logical design size", async () => {
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));
  const main = await readProjectFile("src-tauri/src/main.rs");

  assert.equal(config.app.windows[0].width, 1200);
  assert.equal(config.app.windows[0].height, 675);
  assert.match(main, /width:\s*1200\.0/);
  assert.match(main, /height:\s*675\.0/);
});

test("notification surface does not expose style or preview controls", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const styles = await readProjectFile("src/styles.css");
  const commands = await readProjectFile("src/commands.ts");
  const api = await readProjectFile("src/api.ts");

  assert.doesNotMatch(ui, /通知样式|通知预览|data-style|previewMarkup|previewText/);
  assert.doesNotMatch(styles, /style-card|preview-panel|toast-preview|runtime-scale/);
  assert.doesNotMatch(commands, /styleAliases|styleMatch|labelForStyle|未知样式/);
  assert.match(api, /export type NotificationStyle = "clean";/);
});

test("dashboard does not expose a manual refresh action", async () => {
  const ui = await readProjectFile("src/ui.ts");

  assert.doesNotMatch(ui, /data-action="refresh"/);
  assert.doesNotMatch(ui, /refresh:\s*"刷新"/);
  assert.doesNotMatch(ui, /refresh:\s*"Refresh"/);
});

test("Windows sidecar build uses GUI subsystem", async () => {
  const prepareSidecar = await readProjectFile("scripts/prepare-sidecar.mjs");
  const releaseWorkflow = await readProjectFile("../.github/workflows/release.yml");

  assert.match(prepareSidecar, /-ldflags/);
  assert.match(prepareSidecar, /-H=windowsgui/);
  assert.match(releaseWorkflow, /go build -ldflags "-H=windowsgui" -o agent-notify-server\.exe/);
  assert.match(releaseWorkflow, /go build -ldflags "-H=windowsgui" -o agent-notify-server-arm64\.exe/);
});

test("Windows startup removes legacy standalone server launchers before managing sidecar", async () => {
  const service = await readProjectFile("src-tauri/src/service.rs");

  assert.match(service, /cleanup_windows_legacy_server_autostart\(\)/);
  assert.match(service, /stop_windows_standalone_server_processes\(\)/);
  assert.match(service, /AgentNotifyServer/);
  assert.match(service, /taskkill/);
  assert.match(service, /CREATE_NO_WINDOW/);
});

test("Windows NSIS installer stops running app and sidecar before replacing files", async () => {
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));
  const hooksPath = config.bundle.windows.nsis.installerHooks;
  const hooks = await readProjectFile(`src-tauri/${hooksPath}`);

  assert.equal(hooksPath, "./windows/installer-hooks.nsh");
  assert.match(hooks, /NSIS_HOOK_PREINSTALL/);
  assert.match(hooks, /NSIS_HOOK_PREUNINSTALL/);
  assert.match(hooks, /taskkill/);
  assert.match(hooks, /AgentNotify\.exe/);
  assert.match(hooks, /agent-notify-server\*\.exe/);
});

test("dashboard exposes a copyable skill install command", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const styles = await readProjectFile("src/styles.css");
  const rootPackage = JSON.parse(await readProjectFile("../package.json"));

  assert.match(ui, /SKILL_INSTALL_COMMAND = "npx skills add TaurusGGBOY\/agent-notification"/);
  assert.match(ui, /安装 skill 命令/);
  assert.match(ui, /data-action="copy-skill-install-command"/);
  assert.match(ui, /navigator\.clipboard\.writeText\(SKILL_INSTALL_COMMAND\)/);
  assert.match(styles, /\.install-command-card\s*{/);
  assert.match(styles, /\.command-copy-row\s*{/);
  assert.deepEqual(rootPackage.bin, { "agent-notification": "scripts/install-skill.mjs" });
});

test("topbar exposes about details on hover and keyboard focus", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const styles = await readProjectFile("src/styles.css");

  assert.match(ui, /data-action="about"/);
  assert.match(ui, /关于 AgentNotify/);
  assert.match(ui, /本机 Agent 通知服务/);
  assert.match(ui, /<dt>\$\{t\("version"\)\}<\/dt><dd>v\$\{escapeHtml\(version\)\}<\/dd>/);
  assert.match(styles, /\.about-control\s*{/);
  assert.match(styles, /\.about-popover\s*{[^}]*position:\s*absolute;/s);
  assert.match(styles, /\.about-control:is\(:hover,\s*:focus-within\)\s+\.about-popover\s*{/);
});

test("dashboard exposes a one-click zh/en language toggle", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const api = await readProjectFile("src/api.ts");

  assert.match(api, /export type AppLanguage = "zh" \| "en";/);
  assert.match(ui, /data-action="language"/);
  assert.match(ui, /saveConfig\(\{ \.\.\.state\.config, language: nextLanguage \}\)/);
  assert.match(ui, /t\("notificationConsole"\)/);
});

test("sidebar exposes a platform startup toggle wired through Tauri commands", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const api = await readProjectFile("src/api.ts");
  const service = await readProjectFile("src/service.ts");
  const state = await readProjectFile("src/state.ts");
  const main = await readProjectFile("src-tauri/src/main.rs");

  assert.match(api, /export interface StartupStatus/);
  assert.match(api, /invoke<StartupStatus>\("startup_status"\)/);
  assert.match(api, /invoke<StartupStatus>\("set_startup_enabled", \{ enabled \}\)/);
  assert.match(service, /getStartupStatus/);
  assert.match(state, /startup: StartupStatus \| null/);
  assert.match(ui, /startupAtLogin/);
  assert.match(ui, /data-action="startup"/);
  assert.match(ui, /setStartupEnabled\(next\)/);
  assert.match(main, /mod startup;/);
  assert.match(main, /startup::startup_status/);
  assert.match(main, /startup::set_startup_enabled/);
});

test("npx install entrypoint installs the bundled discovery skill", async () => {
  const installer = await readProjectFile("../scripts/install-skill.mjs");

  assert.match(installer, /agent-notify-discovery/);
  assert.match(installer, /\.claude/);
  assert.match(installer, /\.openclaw/);
  assert.match(installer, /zeroconf/);
});

test("dashboard keeps history contained after adding the install command card", async () => {
  const styles = await readProjectFile("src/styles.css");

  assert.match(styles, /\.history-card\s*{[^}]*overflow:\s*hidden;/s);
  assert.match(styles, /\.history-list\s*{[^}]*overflow:\s*auto;/s);
});

test("dashboard keeps notification history fresh while the app is open", async () => {
  const main = await readProjectFile("src/main.ts");
  const service = await readProjectFile("src/service.ts");

  assert.match(service, /export function startAutoRefresh/);
  assert.match(service, /window\.setInterval\(\s*\(\)\s*=>\s*\{/);
  assert.match(service, /void refreshState\(\)\.then\(render\)/);
  assert.match(main, /startAutoRefresh\(render\)/);
});
