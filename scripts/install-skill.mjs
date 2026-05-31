#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { existsSync, cpSync, mkdirSync, rmSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const skillName = "agent-notify-discovery";
const skillSource = join(repoRoot, "skills");
const skillTarget = join(homedir(), ".claude", "skills", skillName);

function run(command, args, options = {}) {
  const result = spawnSync(command, args, { stdio: "inherit", ...options });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed with exit code ${result.status}`);
  }
}

function findPython() {
  const candidates =
    process.platform === "win32"
      ? [
          ["py", ["-3"]],
          ["python", []],
          ["python3", []],
        ]
      : [
          ["python3", []],
          ["python", []],
        ];

  for (const [command, prefixArgs] of candidates) {
    const result = spawnSync(command, [...prefixArgs, "--version"], { stdio: "ignore" });
    if (result.status === 0) {
      return { command, prefixArgs };
    }
  }

  throw new Error("Python 3 is required to install the agent-notify-discovery skill.");
}

function venvPythonPath() {
  return process.platform === "win32"
    ? join(skillTarget, ".venv", "Scripts", "python.exe")
    : join(skillTarget, ".venv", "bin", "python");
}

if (!existsSync(skillSource)) {
  throw new Error(`Skill source not found at ${skillSource}`);
}

const python = findPython();

console.log(`Installing ${skillName} skill...`);
mkdirSync(dirname(skillTarget), { recursive: true });
rmSync(skillTarget, { recursive: true, force: true });
cpSync(skillSource, skillTarget, { recursive: true });

run(python.command, [...python.prefixArgs, "-m", "venv", join(skillTarget, ".venv")]);

const venvPython = venvPythonPath();
run(venvPython, ["-m", "pip", "install", "--upgrade", "pip"]);
run(venvPython, ["-m", "pip", "install", "zeroconf"]);

console.log(`Installed ${skillName} skill at ${skillTarget}`);

if (process.argv.includes("--test")) {
  run(venvPython, [join(skillTarget, "scripts", "discover.py"), "--json"]);
}
