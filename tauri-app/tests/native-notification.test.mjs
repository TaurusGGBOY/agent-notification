import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const readProjectFile = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("managed macOS sidecar notifications are shown by the app process", async () => {
  const api = await readProjectFile("src/api.ts");
  const main = await readProjectFile("src-tauri/src/main.rs");
  const capabilities = JSON.parse(await readProjectFile("src-tauri/capabilities/default.json"));
  const packageJson = JSON.parse(await readProjectFile("package.json"));
  const cargoToml = await readProjectFile("src-tauri/Cargo.toml");
  const service = await readProjectFile("src-tauri/src/service.rs");
  const nativeNotification = await readProjectFile("src-tauri/native/macos_notification.m");

  assert.doesNotMatch(api, /@tauri-apps\/plugin-notification/);
  assert.doesNotMatch(main, /tauri_plugin_notification::init\(\)/);
  assert.ok(!capabilities.permissions.includes("notification:default"));
  assert.ok(!packageJson.dependencies["@tauri-apps/plugin-notification"]);
  assert.doesNotMatch(cargoToml, /tauri-plugin-notification/);
  assert.match(service, /#\[cfg\(target_os = "macos"\)\][\s\S]*AGENT_NOTIFY_TAURI_STDOUT/);
  assert.match(service, /AGENT_NOTIFY_TAURI_NOTIFICATION/);
  assert.match(nativeNotification, /UNUserNotificationCenter/);
  assert.match(nativeNotification, /UNNotificationPresentationOptionBanner/);
  assert.doesNotMatch(nativeNotification, /NSUserNotification/);
});

test("macOS notification settings opens this app's notification detail page", async () => {
  const notificationSettings = await readProjectFile("src-tauri/src/notification_settings.rs");
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));

  assert.equal(config.identifier, "com.agentnotify.client");
  assert.match(notificationSettings, /com\.apple\.Notifications-Settings\.extension\?id=/);
  assert.match(notificationSettings, /com\.agentnotify\.client/);
  assert.match(notificationSettings, /com\.apple\.preference\.notifications\?id=/);
});
