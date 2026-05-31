import { execFileSync } from "node:child_process";
import { mkdirSync, copyFileSync, chmodSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";

const repoRoot = resolve("..");
const serverDir = join(repoRoot, "windows-server");
const binDir = resolve("src-tauri", "binaries");
mkdirSync(binDir, { recursive: true });

const rustVersion = execFileSync("rustc", ["-vV"], { encoding: "utf8" });
const targetTriple = rustVersion
  .split("\n")
  .find((line) => line.startsWith("host:"))
  ?.replace("host:", "")
  .trim();
if (!targetTriple) {
  throw new Error("could not read host target triple from rustc -vV");
}

const isWindows = process.platform === "win32";
const ext = isWindows ? ".exe" : "";
const outName = `agent-notify-server-${targetTriple}${ext}`;
const outPath = join(binDir, outName);
const tempName = `agent-notify-server-build${ext}`;
const tempPath = join(serverDir, tempName);

const env = { ...process.env };
if (targetTriple.includes("windows")) {
  env.GOOS = "windows";
  env.GOARCH = targetTriple.includes("aarch64") ? "arm64" : "amd64";
} else if (targetTriple.includes("apple-darwin")) {
  env.GOOS = "darwin";
  env.GOARCH = targetTriple.includes("aarch64") ? "arm64" : "amd64";
} else {
  env.GOOS = "linux";
  env.GOARCH = targetTriple.includes("aarch64") ? "arm64" : "amd64";
}

const buildArgs = ["build"];
if (targetTriple.includes("windows")) {
  buildArgs.push("-ldflags", "-H=windowsgui");
}
buildArgs.push("-o", tempPath);

execFileSync("go", buildArgs, {
  cwd: serverDir,
  env,
  stdio: "inherit",
});

if (!existsSync(tempPath)) {
  throw new Error(`expected Go build output at ${tempPath}`);
}

copyFileSync(tempPath, outPath);
chmodSync(outPath, 0o755);
console.log(`Prepared sidecar ${outPath}`);
