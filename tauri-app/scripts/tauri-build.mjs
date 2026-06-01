import { execFileSync } from "node:child_process";

const env = { ...process.env };

if (
  process.platform === "darwin" &&
  !env.APPLE_SIGNING_IDENTITY &&
  env.MACOS_RELEASE_MODE !== "signed"
) {
  env.APPLE_SIGNING_IDENTITY = "-";
}

const npx = process.platform === "win32" ? "npx.cmd" : "npx";
execFileSync(npx, ["tauri", "build", ...process.argv.slice(2)], {
  env,
  stdio: "inherit",
});
