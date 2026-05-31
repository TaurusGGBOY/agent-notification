import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const readProjectFile = (path) => readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("test notification uses the Tauri native notification plugin", async () => {
  const api = await readProjectFile("src/api.ts");
  const main = await readProjectFile("src-tauri/src/main.rs");
  const capabilities = JSON.parse(await readProjectFile("src-tauri/capabilities/default.json"));
  const packageJson = JSON.parse(await readProjectFile("package.json"));
  const cargoToml = await readProjectFile("src-tauri/Cargo.toml");

  assert.match(api, /@tauri-apps\/plugin-notification/);
  assert.match(api, /isPermissionGranted/);
  assert.match(api, /requestPermission/);
  assert.match(api, /sendNotification/);
  assert.match(api, /sendNativeTestNotification/);
  assert.match(api, /const windowsNotifications = await getWindowsNotificationStatus\(\);/);
  assert.match(api, /if \(!windowsNotifications\.supported\) \{[\s\S]*await sendNativeTestNotification\(event\);[\s\S]*\}/);
  assert.match(main, /tauri_plugin_notification::init\(\)/);
  assert.ok(capabilities.permissions.includes("notification:default"));
  assert.ok(packageJson.dependencies["@tauri-apps/plugin-notification"]);
  assert.match(cargoToml, /tauri-plugin-notification/);
});
