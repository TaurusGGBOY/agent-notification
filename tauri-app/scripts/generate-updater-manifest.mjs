import { createHash } from "node:crypto";
import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const [distDir, tagName, repository = "TaurusGGBOY/agent-notification"] = process.argv.slice(2);

if (!distDir || !tagName) {
  throw new Error("usage: node scripts/generate-updater-manifest.mjs <dist-dir> <tag>");
}

const version = tagName.replace(/^v/, "");
const releaseBaseUrl = `https://github.com/${repository}/releases/download/${tagName}`;
const entries = await readdir(distDir);

const asset = (predicate) => entries.find(predicate);
const signatureFor = async (name) => {
  const sigName = `${name}.sig`;
  if (!entries.includes(sigName)) {
    throw new Error(`missing updater signature for ${name}`);
  }
  return (await readFile(path.join(distDir, sigName), "utf8")).trim();
};

const windowsAsset = asset((name) => /_x64-setup\.exe$/i.test(name));
const macosAsset = asset((name) => /\.app\.tar\.gz$/i.test(name));

if (!windowsAsset) {
  throw new Error("missing Windows NSIS updater asset ending with _x64-setup.exe");
}
if (!macosAsset) {
  throw new Error("missing macOS app tarball updater asset ending with .app.tar.gz");
}

const windowsSignature = await signatureFor(windowsAsset);
const macosSignature = await signatureFor(macosAsset);
const macosPlatforms = entries.includes("TAURI_UPDATER_MACOS_PLATFORMS.txt")
  ? (await readFile(path.join(distDir, "TAURI_UPDATER_MACOS_PLATFORMS.txt"), "utf8"))
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
  : ["darwin-aarch64", "darwin-x86_64"];
const notesAsset = asset((name) => /^SHA256SUMS\.txt$/i.test(name));
const notes = notesAsset
  ? await readFile(path.join(distDir, notesAsset), "utf8")
  : `AgentNotify ${tagName}`;

const manifest = {
  version,
  notes,
  pub_date: new Date().toISOString(),
  platforms: {
    "windows-x86_64": {
      signature: windowsSignature,
      url: `${releaseBaseUrl}/${encodeURIComponent(windowsAsset)}`,
    },
  },
};

for (const platform of macosPlatforms) {
  if (!["darwin-aarch64", "darwin-x86_64"].includes(platform)) {
    throw new Error(`unsupported macOS updater platform: ${platform}`);
  }
  manifest.platforms[platform] = {
    signature: macosSignature,
    url: `${releaseBaseUrl}/${encodeURIComponent(macosAsset)}`,
  };
}

await writeFile(
  path.join(distDir, "latest.json"),
  `${JSON.stringify(manifest, null, 2)}\n`,
);

const digest = createHash("sha256")
  .update(await readFile(path.join(distDir, "latest.json")))
  .digest("hex");
console.log(`generated latest.json for ${version} (${digest})`);
