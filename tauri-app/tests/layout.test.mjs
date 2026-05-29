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
