import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const readProjectFile = (path) =>
  readFile(new URL(`../${path}`, import.meta.url), "utf8");

test("Tauri tray uses Windows and macOS icons", async () => {
  const config = JSON.parse(await readProjectFile("src-tauri/tauri.conf.json"));
  const tray = await readProjectFile("src-tauri/src/tray.rs");
  const icons = config.bundle.icon;

  assert.deepEqual(
    ["icons/icon.icns", "icons/icon.ico", "icons/icon.png"].every((icon) =>
      icons.includes(icon),
    ),
    true,
  );
  await access(new URL("../src-tauri/icons/icon.icns", import.meta.url));

  assert.match(
    tray,
    /\.icon\([\s\S]*app\.default_window_icon\(\)[\s\S]*\.cloned\(\)[\s\S]*\.expect/s,
  );
  assert.match(tray, /\.icon_as_template\(\s*cfg!\(target_os = "macos"\)\s*\)/);
  assert.match(tray, /\.show_menu_on_left_click\(false\)/);
});

test("close hides the main window and only tray quit exits", async () => {
  const main = await readProjectFile("src-tauri/src/main.rs");
  const tray = await readProjectFile("src-tauri/src/tray.rs");

  assert.match(main, /WindowEvent::CloseRequested\s*\{\s*api,\s*\.\.\s*\}/);
  assert.match(main, /api\.prevent_close\(\);[\s\S]*window\.hide\(\)/);

  assert.match(main, /RunEvent::ExitRequested\s*\{\s*api,\s*\.\.\s*\}/);
  assert.match(main, /consume_tray_exit_request\(\)/);
  assert.match(main, /api\.prevent_exit\(\)/);
  assert.match(main, /\.enable_macos_default_menu\(false\)/);
  assert.match(main, /ActivationPolicy::Regular/);
  assert.match(main, /if let Err\(err\) = service::ensure_sidecar\(app\.handle\(\)\)/);
  assert.doesNotMatch(main, /service::ensure_sidecar\(app\.handle\(\)\?;/);

  assert.match(tray, /"quit"\s*=>\s*\{[\s\S]*request_tray_exit\(app\);[\s\S]*app\.exit\(0\)/);
});
