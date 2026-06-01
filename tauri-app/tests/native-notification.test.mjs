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

test("macOS notification delegate is registered before startup notifications can fire", async () => {
  const main = await readProjectFile("src-tauri/src/main.rs");
  const nativeBindings = await readProjectFile("src-tauri/src/native_notification.rs");
  const nativeNotification = await readProjectFile("src-tauri/native/macos_notification.m");

  const setupIndex = main.indexOf(".setup(|app| {");
  const delegateIndex = main.indexOf("native_notification::configure_delegate_early();");
  const sidecarIndex = main.indexOf("service::ensure_sidecar(app.handle())");

  assert.notEqual(setupIndex, -1);
  assert.notEqual(delegateIndex, -1);
  assert.notEqual(sidecarIndex, -1);
  assert.ok(delegateIndex > setupIndex);
  assert.ok(delegateIndex < sidecarIndex);

  assert.match(nativeBindings, /fn agentnotify_configure_notification_center_early\(\);/);
  assert.match(nativeBindings, /pub fn configure_delegate_early\(\)/);
  assert.match(nativeNotification, /void agentnotify_configure_notification_center_early\(void\)/);
  assert.match(
    nativeNotification,
    /static void agentnotify_configure_notification_center\(void\) \{[\s\S]*agentnotify_configure_notification_center_early\(\);[\s\S]*\}/,
  );
});

test("macOS notification delivery does not force the app into the foreground", async () => {
  const nativeNotification = await readProjectFile("src-tauri/native/macos_notification.m");

  assert.doesNotMatch(nativeNotification, /activateIgnoringOtherApps/);
  assert.doesNotMatch(nativeNotification, /activated app active/);
  assert.doesNotMatch(nativeNotification, /UNTimeIntervalNotificationTrigger/);
  assert.match(nativeNotification, /content:content\s+trigger:nil/);
});

test("test notification is marked so it always exercises native delivery", async () => {
  const api = await readProjectFile("src/api.ts");

  assert.match(api, /sourcePayload:\s*\{\s*agentNotifyTest:\s*true\s*\}/);
});

test("macOS build defaults to ad-hoc signing so notifications use the app bundle identity", async () => {
  const packageJson = JSON.parse(await readProjectFile("package.json"));
  const tauriBuild = await readProjectFile("scripts/tauri-build.mjs");

  assert.equal(packageJson.scripts["tauri:build"], "npm run prepare:sidecar && node scripts/tauri-build.mjs");
  assert.match(tauriBuild, /process\.platform === "darwin"/);
  assert.match(tauriBuild, /env\.MACOS_RELEASE_MODE !== "signed"/);
  assert.match(tauriBuild, /env\.APPLE_SIGNING_IDENTITY = "-"/);
});

test("macOS notification settings opens this app's notification detail page", async () => {
  const notificationSettings = await readProjectFile("src-tauri/src/notification_settings.rs");
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));

  assert.equal(config.identifier, "com.agentnotify.client");
  assert.match(notificationSettings, /com\.apple\.Notifications-Settings\.extension\?id=/);
  assert.match(notificationSettings, /com\.agentnotify\.client/);
  assert.match(notificationSettings, /com\.apple\.preference\.notifications\?id=/);
});
