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

test("toast preview puts event agent and directory in first-glance positions", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const styles = await readProjectFile("src/styles.css");

  assert.match(ui, /START · Claude/);
  assert.ok(ui.includes("DIR: /Users/me/project/agent-notification"));
  assert.doesNotMatch(ui, /Agent 已启动/);
  assert.match(styles, /\.toast-preview\s+\.state-tag\s*{/);
});

test("dashboard exposes a copyable skill install command", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const styles = await readProjectFile("src/styles.css");
  const rootPackage = JSON.parse(await readProjectFile("../package.json"));

  assert.match(ui, /SKILL_INSTALL_COMMAND = "npx github:TaurusGGBOY\/agent-notification"/);
  assert.match(ui, /安装 skill 命令/);
  assert.match(ui, /data-action="copy-skill-install-command"/);
  assert.match(ui, /navigator\.clipboard\.writeText\(SKILL_INSTALL_COMMAND\)/);
  assert.match(styles, /\.install-command-card\s*{/);
  assert.match(styles, /\.command-copy-row\s*{/);
  assert.deepEqual(rootPackage.bin, { "agent-notification": "scripts/install-skill.mjs" });
});

test("npx install entrypoint installs the bundled discovery skill", async () => {
  const installer = await readProjectFile("../scripts/install-skill.mjs");

  assert.match(installer, /agent-notify-discovery/);
  assert.match(installer, /\.claude/);
  assert.match(installer, /zeroconf/);
});

test("dashboard keeps history contained after adding the install command card", async () => {
  const styles = await readProjectFile("src/styles.css");

  assert.match(styles, /\.history-card\s*{[^}]*overflow:\s*hidden;/s);
  assert.match(styles, /\.history-list\s*{[^}]*overflow:\s*auto;/s);
});
