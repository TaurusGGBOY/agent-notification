import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const readProjectFile = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("Tauri updater is configured for signed GitHub release manifests", async () => {
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));
  const cargo = await readProjectFile("src-tauri/Cargo.toml");
  const main = await readProjectFile("src-tauri/src/main.rs");
  const permissions = JSON.parse(await readProjectFile("src-tauri/capabilities/default.json"));

  assert.match(cargo, /tauri-plugin-updater/);
  assert.match(main, /\.plugin\(tauri_plugin_updater::Builder::new\(\)\.build\(\)\)/);
  assert.deepEqual(config.plugins.updater.endpoints, [
    "https://github.com/TaurusGGBOY/agent-notification/releases/latest/download/latest.json",
  ]);
  assert.match(config.plugins.updater.pubkey, /^dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6/);
  assert.equal(config.plugins.updater.windows.installMode, "passive");
  assert.equal(config.bundle.createUpdaterArtifacts, true);
  assert.ok(permissions.permissions.includes("updater:default"));
  assert.ok(permissions.permissions.includes("process:allow-restart"));
});

test("UI exposes a manual update check with status feedback", async () => {
  const ui = await readProjectFile("src/ui.ts");
  const api = await readProjectFile("src/api.ts");

  assert.match(api, /export type UpdateCheckResult/);
  assert.match(api, /@tauri-apps\/plugin-updater/);
  assert.match(api, /@tauri-apps\/plugin-process/);
  assert.match(api, /downloadAndInstall/);
  assert.match(api, /relaunch/);
  assert.match(ui, /data-action="check-update"/);
  assert.match(ui, /检查更新/);
  assert.match(ui, /state\.updateStatus/);
  assert.match(ui, /安装并重启/);
});

test("release workflow signs updater artifacts and publishes latest manifest", async () => {
  const workflow = await readProjectFile("../.github/workflows/release.yml");
  const checklist = await readProjectFile("../docs/release-checklist.md");

  assert.match(workflow, /TAURI_SIGNING_PRIVATE_KEY/);
  assert.match(workflow, /TAURI_SIGNING_PRIVATE_KEY_PASSWORD/);
  assert.match(workflow, /latest\.json/);
  assert.match(workflow, /\.sig/);
  assert.match(workflow, /TAURI_UPDATER_MACOS_PLATFORMS\.txt/);
  assert.match(checklist, /latest\.json/);
  assert.match(checklist, /TAURI_UPDATER_PUBLIC_KEY/);
});
