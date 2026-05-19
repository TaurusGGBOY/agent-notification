# Tauri System Tray Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Tauri v2 desktop client that bundles the existing Go notification server as a sidecar, adds a Windows/macOS system tray, and replaces the old Go-served settings page with a compact 2026-style command client.

**Architecture:** Tauri owns desktop app lifecycle, tray UI, command window, command palette, and sidecar lifecycle. The existing Go server remains the notification engine and is bundled as a Tauri sidecar using `bundle.externalBin`. Tauri communicates with the Go server through `http://127.0.0.1:17891` and starts the sidecar only when no compatible server is already healthy.

**Tech Stack:** Tauri v2, Rust, `tauri-plugin-shell`, TypeScript, Vite, HTML/CSS, Go sidecar, existing AgentNotify HTTP API.

---

## References

- Tauri v2 sidecar docs: https://v2.tauri.app/develop/sidecar/
- Tauri v2 tray docs: https://v2.tauri.app/learn/system-tray/
- Tauri v2 config docs: https://v2.tauri.app/develop/configuration-files/

Key constraints from docs:
- `bundle.externalBin` paths are relative to `src-tauri/tauri.conf.json`.
- Sidecar binaries need a `-$TARGET_TRIPLE` suffix.
- Rust sidecar startup uses `tauri_plugin_shell::ShellExt` and `app.shell().sidecar("name")`.
- Tauri tray requires the `tray-icon` feature.

## File Structure

```
agent-notification/
  docs/superpowers/specs/2026-05-19-tauri-system-tray-design.md
    # Update to final decisions: sidecar, migrated command UI, auto theme, 2026 Windows client.

  tauri-app/
    package.json
    index.html
    tsconfig.json
    vite.config.ts
    src/
      main.ts
      api.ts
      state.ts
      service.ts
      ui.ts
      commands.ts
      styles.css
    src-tauri/
      Cargo.toml
      build.rs
      tauri.conf.json
      capabilities/default.json
      icons/icon.ico
      icons/icon.png
      binaries/.gitkeep
      src/main.rs
      src/service.rs
      src/tray.rs

  windows-server/
    main.go
      # Add optional env override support if needed; keep HTTP API unchanged.
    settings.go
      # Keep old /settings for backward compatibility during migration.
    Makefile
      # Add sidecar build target if useful.
```

---

## Task 1: Align The Existing Spec With Final Scope

**Files:**
- Modify: `docs/superpowers/specs/2026-05-19-tauri-system-tray-design.md`

- [ ] **Step 1: Replace outdated architecture text**

Replace the current spec with:

````markdown
# Tauri System Tray Client Design

## Overview

Build a Tauri v2 desktop client for AgentNotify. The client bundles the existing Go HTTP server as a sidecar, owns the system tray, and presents a compact 2026-style Windows command client for agent notifications.

This is not a traditional settings window. The main window is a tray-first command center: search and commands are the primary interaction, while notification settings remain visible for quick direct control.

## Decisions

- Use Tauri v2.
- Bundle the Go server as a Tauri sidecar.
- Keep the Go server as the notification engine.
- Migrate the UI into Tauri instead of embedding `/settings`.
- Preserve the Go `/settings` page for backward compatibility during migration.
- Redefine the UI as a compact command client, not a settings panel.
- Use auto theme: light and dark follow the OS theme.
- Use Windows 11 / Fluent 2 inspired surfaces, density, and motion.
- Add deterministic command palette actions in MVP.
- Keep AI intent parsing as a future extension point, not MVP logic.
- Do not rewrite the Go server in Rust.
- Do not add authentication in this phase.

## Architecture

```text
Tauri App
  - System tray
  - Compact command window
  - Command palette
  - Context/status panel
  - Sidecar lifecycle manager
  - Local HTTP client
       |
       | http://127.0.0.1:17891
       v
Go AgentNotify Server Sidecar
  - /health
  - /manifest
  - /notify
  - /config
  - /settings
  - Windows toast + custom-card PNG generation
  - mDNS/DNS-SD advertisement
```

## UI Model

AgentNotify is a tray-first Windows command client for agent notifications. The main window is a compact control surface, roughly 460px wide, with:

- A command/search bar as the first interaction.
- A minimal navigation rail.
- A central area for notification style, event toggles, and live card preview.
- A context panel for server state, last event, and quick service actions.
- Auto theme based on OS preference.

The UI should feel closer to Cursor, modern Windows utilities, and Fluent 2 command surfaces than to a web dashboard. It should avoid oversized cards, heavy gradients, decorative blobs, emoji-heavy labels, and traditional menu bars.

## Command Bar

The command bar accepts deterministic commands in MVP:

- `test` / `send test`
- `pause`
- `resume`
- `start off`
- `start on`
- `stop off`
- `stop on`
- `style clean`
- `style card`
- `restart`

The command bar can later evolve into AI intent parsing without changing the surrounding UI architecture.

## Tray Menu

- Open AgentNotify
- Send Test Notification
- Pause Notifications
- Resume Notifications
- Restart Service
- Quit

Left-click tray opens or focuses the command window. Closing the window hides it; quitting from the tray exits the app.

## Sidecar Lifecycle

On startup, Tauri checks `http://127.0.0.1:17891/health`. If a compatible server is already running, Tauri reuses it. Otherwise, Tauri starts the bundled Go sidecar. On quit, Tauri stops only the sidecar process it started.

## Build Output

The user installs one Tauri app bundle. Internally, the bundle includes the Go server sidecar binary. The app experience should feel like one product even though the Go server remains a separate sidecar process.

## Constraints

- Windows is the primary target.
- macOS is secondary and may use toast stubs.
- HTTP port remains `17891`.
- mDNS/DNS-SD remains the discovery protocol.
- UDP broadcast is not part of this design.
- The old Go `/settings` page remains available during transition, but the Tauri UI is the preferred client.
````

- [ ] **Step 2: Self-review the spec**

Run:

```bash
rg -n "UDP|broadcast|single executable|Go Service \\(no change\\)|clean/status-color/agent-badge/compact\\)" docs/superpowers/specs/2026-05-19-tauri-system-tray-design.md
```

Expected: no matches.

- [ ] **Step 3: Commit spec alignment**

```bash
git add docs/superpowers/specs/2026-05-19-tauri-system-tray-design.md
git commit -m "docs: align tauri tray client design"
```

---

## Task 2: Scaffold Tauri v2 App

**Files:**
- Create: `tauri-app/package.json`
- Create: `tauri-app/index.html`
- Create: `tauri-app/tsconfig.json`
- Create: `tauri-app/vite.config.ts`
- Create: `tauri-app/src/main.ts`
- Create: `tauri-app/src/styles.css`
- Create: `tauri-app/src-tauri/Cargo.toml`
- Create: `tauri-app/src-tauri/build.rs`
- Create: `tauri-app/src-tauri/tauri.conf.json`
- Create: `tauri-app/src-tauri/capabilities/default.json`
- Create: `tauri-app/src-tauri/src/main.rs`
- Create: `tauri-app/src-tauri/binaries/.gitkeep`

- [ ] **Step 1: Create frontend package files**

Create `tauri-app/package.json`:

```json
{
  "name": "agent-notify-tauri",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite --host 127.0.0.1",
    "build": "tsc && vite build",
    "tauri": "tauri",
    "tauri:dev": "tauri dev",
    "tauri:build": "tauri build"
  },
  "dependencies": {
    "@tauri-apps/api": "^2.0.0"
  },
  "devDependencies": {
    "@tauri-apps/cli": "^2.0.0",
    "typescript": "^5.6.0",
    "vite": "^6.0.0"
  }
}
```

Create `tauri-app/index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>AgentNotify</title>
  </head>
  <body>
    <main id="app"></main>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

Create `tauri-app/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "strict": true
  },
  "include": ["src"]
}
```

Create `tauri-app/vite.config.ts`:

```ts
import { defineConfig } from "vite";

export default defineConfig({
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
  },
  envPrefix: ["VITE_", "TAURI_"],
  build: {
    target: "es2022",
    minify: false,
  },
});
```

- [ ] **Step 2: Create placeholder frontend**

Create `tauri-app/src/main.ts`:

```ts
import "./styles.css";

const app = document.querySelector<HTMLMainElement>("#app");

if (!app) {
  throw new Error("missing #app root");
}

app.innerHTML = `
  <section class="shell">
    <header class="topbar">
      <div>
        <h1>AgentNotify</h1>
        <p>localhost:17891</p>
      </div>
      <span class="status">Starting</span>
    </header>
    <section class="panel">
      <p>Tauri client shell ready.</p>
    </section>
  </section>
`;
```

Create `tauri-app/src/styles.css`:

```css
:root {
  color-scheme: light dark;
  font-family: "Segoe UI", system-ui, sans-serif;
  background: #f6f7f9;
  color: #1f2328;
}

@media (prefers-color-scheme: dark) {
  :root {
    background: #0f141b;
    color: #f8fafc;
  }
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-width: 420px;
  min-height: 540px;
}

.shell {
  padding: 18px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

h1 {
  margin: 0;
  font-size: 17px;
}

p {
  margin: 0;
  color: #667085;
  font-size: 12px;
}

.status {
  border: 1px solid #bde8ce;
  border-radius: 999px;
  color: #0f7a43;
  background: #e8f7ef;
  font-size: 12px;
  padding: 5px 9px;
}

.panel {
  border: 1px solid color-mix(in srgb, currentColor 14%, transparent);
  border-radius: 12px;
  padding: 14px;
  background: color-mix(in srgb, Canvas 92%, currentColor 3%);
}
```

- [ ] **Step 3: Create Rust/Tauri config**

Create `tauri-app/src-tauri/Cargo.toml`:

```toml
[package]
name = "agent-notify"
version = "0.1.0"
description = "AgentNotify desktop client"
authors = ["AgentNotify"]
edition = "2021"
rust-version = "1.84"

[build-dependencies]
tauri-build = { version = "2.0.0" }

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
tauri = { version = "2.0.0", features = ["tray-icon"] }
tauri-plugin-shell = "2.0.0"
```

Create `tauri-app/src-tauri/build.rs`:

```rust
fn main() {
    tauri_build::build()
}
```

Create `tauri-app/src-tauri/tauri.conf.json`:

```json
{
  "$schema": "https://schema.tauri.app/config/2",
  "productName": "AgentNotify",
  "version": "0.1.0",
  "identifier": "com.agentnotify.client",
  "build": {
    "beforeDevCommand": "npm run dev",
    "beforeBuildCommand": "npm run build",
    "devUrl": "http://127.0.0.1:1420",
    "frontendDist": "../dist"
  },
  "app": {
    "windows": [
      {
        "label": "main",
        "title": "AgentNotify",
        "width": 460,
        "height": 560,
        "resizable": false,
        "fullscreen": false,
        "visible": false
      }
    ]
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "externalBin": ["binaries/agent-notify-server"],
    "icon": ["icons/icon.png", "icons/icon.ico"]
  }
}
```

Create `tauri-app/src-tauri/capabilities/default.json`:

```json
{
  "$schema": "../gen/schemas/desktop-schema.json",
  "identifier": "default",
  "description": "Main window capability",
  "windows": ["main"],
  "permissions": [
    "core:default",
    {
      "identifier": "shell:allow-spawn",
      "allow": [
        {
          "name": "binaries/agent-notify-server",
          "sidecar": true
        }
      ]
    }
  ]
}
```

Create `tauri-app/src-tauri/src/main.rs`:

```rust
fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!())
        .expect("error while running AgentNotify");
}
```

Create `tauri-app/src-tauri/binaries/.gitkeep` as an empty file.

- [ ] **Step 4: Install dependencies and verify scaffold**

Run:

```bash
cd tauri-app
npm install
npm run build
cd src-tauri
cargo check
```

Expected: npm install succeeds, frontend build succeeds, Rust compiles.

- [ ] **Step 5: Commit scaffold**

```bash
git add tauri-app
git commit -m "feat: scaffold tauri client"
```

---

## Task 3: Prepare Go Server Sidecar Binary

**Files:**
- Create: `tauri-app/scripts/prepare-sidecar.mjs`
- Modify: `tauri-app/package.json`
- Modify: `.gitignore`

- [ ] **Step 1: Add failing verification script expectation**

Create `tauri-app/scripts/prepare-sidecar.mjs`:

```js
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

execFileSync("go", ["build", "-o", tempPath], {
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
```

- [ ] **Step 2: Add package scripts**

Modify `tauri-app/package.json` scripts:

```json
{
  "scripts": {
    "dev": "vite --host 127.0.0.1",
    "build": "tsc && vite build",
    "prepare:sidecar": "node scripts/prepare-sidecar.mjs",
    "tauri": "tauri",
    "tauri:dev": "npm run prepare:sidecar && tauri dev",
    "tauri:build": "npm run prepare:sidecar && tauri build"
  }
}
```

- [ ] **Step 3: Ignore generated sidecars**

Add to `.gitignore`:

```gitignore
tauri-app/src-tauri/binaries/agent-notify-server-*
```

- [ ] **Step 4: Run sidecar preparation**

Run:

```bash
cd tauri-app
npm run prepare:sidecar
ls -l src-tauri/binaries
```

Expected: one binary named like `agent-notify-server-aarch64-apple-darwin` on Apple Silicon macOS or `agent-notify-server-x86_64-pc-windows-msvc.exe` on Windows.

- [ ] **Step 5: Commit sidecar build script**

```bash
git add .gitignore tauri-app/package.json tauri-app/scripts/prepare-sidecar.mjs tauri-app/src-tauri/binaries/.gitkeep
git commit -m "feat: prepare go server sidecar"
```

---

## Task 4: Manage Sidecar Lifecycle In Rust

**Files:**
- Create: `tauri-app/src-tauri/src/service.rs`
- Modify: `tauri-app/src-tauri/src/main.rs`

- [ ] **Step 1: Add service module**

Create `tauri-app/src-tauri/src/service.rs`:

```rust
use std::sync::Mutex;
use std::time::Duration;

use tauri::{AppHandle, Manager, State};
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

pub struct ServiceState {
    child: Mutex<Option<CommandChild>>,
}

impl ServiceState {
    pub fn new() -> Self {
        Self {
            child: Mutex::new(None),
        }
    }
}

#[derive(serde::Serialize)]
pub struct ServiceStatus {
    pub healthy: bool,
    pub managed_by_tauri: bool,
}

pub fn is_server_healthy() -> bool {
    let client = match tiny_http_check::Client::new("127.0.0.1:17891", Duration::from_millis(600)) {
        Ok(client) => client,
        Err(_) => return false,
    };
    client.get("/health").is_ok()
}

pub fn ensure_sidecar(app: &AppHandle) -> Result<(), String> {
    if is_server_healthy() {
        return Ok(());
    }

    let state = app.state::<ServiceState>();
    let mut guard = state.child.lock().map_err(|_| "service mutex poisoned")?;
    if guard.is_some() {
        return Ok(());
    }

    let command = app
        .shell()
        .sidecar("agent-notify-server")
        .map_err(|err| format!("failed to create sidecar command: {err}"))?;

    let (mut rx, child) = command
        .env("AGENT_NOTIFY_HTTP_ADDR", "127.0.0.1:17891")
        .spawn()
        .map_err(|err| format!("failed to spawn sidecar: {err}"))?;

    let app_for_events = app.clone();
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
                    let line = String::from_utf8_lossy(&bytes).to_string();
                    let _ = app_for_events.emit("agentnotify://server-stdout", line);
                }
                CommandEvent::Stderr(bytes) => {
                    let line = String::from_utf8_lossy(&bytes).to_string();
                    let _ = app_for_events.emit("agentnotify://server-stderr", line);
                }
                _ => {}
            }
        }
    });

    *guard = Some(child);
    Ok(())
}

pub fn stop_sidecar(state: &State<ServiceState>) {
    if let Ok(mut guard) = state.child.lock() {
        if let Some(child) = guard.as_mut() {
            let _ = child.kill();
        }
        *guard = None;
    }
}

#[tauri::command]
pub fn service_status(state: State<ServiceState>) -> ServiceStatus {
    ServiceStatus {
        healthy: is_server_healthy(),
        managed_by_tauri: state.child.lock().map(|child| child.is_some()).unwrap_or(false),
    }
}

mod tiny_http_check {
    use std::io::{Read, Write};
    use std::net::TcpStream;
    use std::time::Duration;

    pub struct Client {
        addr: String,
        timeout: Duration,
    }

    impl Client {
        pub fn new(addr: &str, timeout: Duration) -> Result<Self, String> {
            Ok(Self {
                addr: addr.to_string(),
                timeout,
            })
        }

        pub fn get(&self, path: &str) -> Result<(), String> {
            let mut stream = TcpStream::connect(&self.addr).map_err(|err| err.to_string())?;
            stream
                .set_read_timeout(Some(self.timeout))
                .map_err(|err| err.to_string())?;
            stream
                .set_write_timeout(Some(self.timeout))
                .map_err(|err| err.to_string())?;
            let req = format!(
                "GET {path} HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n"
            );
            stream.write_all(req.as_bytes()).map_err(|err| err.to_string())?;
            let mut response = String::new();
            stream.read_to_string(&mut response).map_err(|err| err.to_string())?;
            if response.starts_with("HTTP/1.1 200") || response.starts_with("HTTP/1.0 200") {
                Ok(())
            } else {
                Err(response.lines().next().unwrap_or("empty response").to_string())
            }
        }
    }
}
```

- [ ] **Step 2: Wire service into Tauri**

Replace `tauri-app/src-tauri/src/main.rs`:

```rust
mod service;

use tauri::Manager;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(service::ServiceState::new())
        .invoke_handler(tauri::generate_handler![service::service_status])
        .setup(|app| {
            service::ensure_sidecar(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                if window.label() == "main" {
                    api.prevent_close();
                    let _ = window.hide();
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("error while building AgentNotify")
        .run(|app, event| {
            if let tauri::RunEvent::ExitRequested { .. } = event {
                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
            }
        });
}
```

- [ ] **Step 3: Verify Rust compile**

Run:

```bash
cd tauri-app/src-tauri
cargo check
```

Expected: compiles.

- [ ] **Step 4: Verify sidecar starts in dev**

Run:

```bash
cd tauri-app
npm run tauri:dev
```

Expected:
- Tauri app launches.
- `curl http://127.0.0.1:17891/health` returns `{"status":"ok","version":"1.0.0"}`.
- Closing the window hides it instead of quitting.

- [ ] **Step 5: Commit service lifecycle**

```bash
git add tauri-app/src-tauri/src/main.rs tauri-app/src-tauri/src/service.rs
git commit -m "feat: manage go sidecar lifecycle"
```

---

## Task 5: Add System Tray

**Files:**
- Create: `tauri-app/src-tauri/src/tray.rs`
- Modify: `tauri-app/src-tauri/src/main.rs`

- [ ] **Step 1: Add tray module**

Create `tauri-app/src-tauri/src/tray.rs`:

```rust
use tauri::menu::{Menu, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager};

use crate::service;

pub fn build_tray(app: &AppHandle) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, "open", "Open AgentNotify", true, None::<&str>)?;
    let test = MenuItem::with_id(app, "test", "Send Test Notification", true, None::<&str>)?;
    let pause = MenuItem::with_id(app, "pause", "Pause Notifications", true, None::<&str>)?;
    let resume = MenuItem::with_id(app, "resume", "Resume Notifications", true, None::<&str>)?;
    let restart = MenuItem::with_id(app, "restart", "Restart Service", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
    let separator = PredefinedMenuItem::separator(app)?;

    let menu = Menu::with_items(app, &[&open, &test, &pause, &resume, &restart, &separator, &quit])?;

    TrayIconBuilder::with_id("agentnotify")
        .tooltip("AgentNotify")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                show_main_window(tray.app_handle());
            }
        })
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_main_window(app),
            "test" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = send_test_notification().await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "pause" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = set_events_enabled(false).await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "resume" => {
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    let _ = set_events_enabled(true).await;
                    let _ = app.emit("agentnotify://refresh", ());
                });
            }
            "restart" => {
                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
                let _ = service::ensure_sidecar(app);
            }
            "quit" => app.exit(0),
            _ => {}
        })
        .build(app)?;

    Ok(())
}

fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

async fn send_test_notification() -> Result<(), String> {
    post_json(
        "/notify",
        r#"{"agent":"tauri","event":"start","project":"AgentNotify","message":"Test notification from tray","sourcePayload":{}}"#,
    )
    .await
}

async fn set_events_enabled(enabled: bool) -> Result<(), String> {
    let events = if enabled { r#"["start","stop"]"# } else { "[]" };
    let body = format!(
        r#"{{"notificationStyle":"custom-card","enabledEvents":{events},"futureOverrides":{{}}}}"#
    );
    post_json("/settings", &body).await
}

async fn post_json(path: &str, body: &str) -> Result<(), String> {
    let addr = "127.0.0.1:17891";
    let request = format!(
        "POST {path} HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
        body.len(),
        body
    );
    tokio_like_tcp(addr, &request).await
}

async fn tokio_like_tcp(addr: &str, request: &str) -> Result<(), String> {
    use std::io::{Read, Write};
    use std::net::TcpStream;

    let addr = addr.to_string();
    let request = request.to_string();
    tauri::async_runtime::spawn_blocking(move || {
        let mut stream = TcpStream::connect(addr).map_err(|err| err.to_string())?;
        stream.write_all(request.as_bytes()).map_err(|err| err.to_string())?;
        let mut response = String::new();
        stream.read_to_string(&mut response).map_err(|err| err.to_string())?;
        if response.starts_with("HTTP/1.1 204")
            || response.starts_with("HTTP/1.0 204")
            || response.starts_with("HTTP/1.1 200")
            || response.starts_with("HTTP/1.0 200")
        {
            Ok(())
        } else {
            Err(response.lines().next().unwrap_or("empty response").to_string())
        }
    })
    .await
    .map_err(|err| err.to_string())?
}
```

- [ ] **Step 2: Wire tray into main**

Modify `tauri-app/src-tauri/src/main.rs`:

```rust
mod service;
mod tray;

use tauri::Manager;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(service::ServiceState::new())
        .invoke_handler(tauri::generate_handler![service::service_status])
        .setup(|app| {
            service::ensure_sidecar(app.handle())?;
            tray::build_tray(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                if window.label() == "main" {
                    api.prevent_close();
                    let _ = window.hide();
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("error while building AgentNotify")
        .run(|app, event| {
            if let tauri::RunEvent::ExitRequested { .. } = event {
                let state = app.state::<service::ServiceState>();
                service::stop_sidecar(&state);
            }
        });
}
```

- [ ] **Step 3: Verify tray**

Run:

```bash
cd tauri-app
npm run tauri:dev
```

Expected:
- Tray icon appears.
- Left click opens/focuses window.
- Right click opens menu.
- Send Test Notification triggers `/notify`.
- Quit exits app and sidecar it started.

- [ ] **Step 4: Commit tray**

```bash
git add tauri-app/src-tauri/src/main.rs tauri-app/src-tauri/src/tray.rs
git commit -m "feat: add tauri system tray"
```

---

## Task 6: Build Local HTTP API Client

**Files:**
- Create: `tauri-app/src/api.ts`
- Create: `tauri-app/src/state.ts`
- Create: `tauri-app/src/service.ts`
- Modify: `tauri-app/src/main.ts`

- [ ] **Step 1: Add typed API client**

Create `tauri-app/src/api.ts`:

```ts
const BASE_URL = "http://127.0.0.1:17891";

export type NotificationStyle = "clean" | "status-color" | "agent-badge" | "compact" | "custom-card";
export type EventName = "start" | "stop";

export interface AgentConfig {
  notificationStyle: NotificationStyle;
  enabledEvents: EventName[];
  futureOverrides: Record<string, string>;
  _path?: string;
}

export interface Manifest {
  name: string;
  version: string;
  url: string;
  hostname: string;
  protocol: string;
  serviceType: string;
  supportedEvents: EventName[];
  supportedStyles: NotificationStyle[];
}

export async function getConfig(): Promise<AgentConfig> {
  return getJson<AgentConfig>("/config");
}

export async function getManifest(): Promise<Manifest> {
  return getJson<Manifest>("/manifest");
}

export async function saveConfig(config: AgentConfig): Promise<void> {
  const res = await fetch(`${BASE_URL}/settings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  if (!res.ok) {
    throw new Error(`save config failed: ${res.status}`);
  }
}

export async function sendTestNotification(event: EventName = "start"): Promise<void> {
  const res = await fetch(`${BASE_URL}/notify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      agent: "tauri",
      event,
      project: "AgentNotify",
      message: "Test notification from AgentNotify",
      timestamp: new Date().toISOString(),
      sourcePayload: {},
    }),
  });
  if (!res.ok) {
    throw new Error(`send test notification failed: ${res.status}`);
  }
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`);
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return (await res.json()) as T;
}
```

Create `tauri-app/src/state.ts`:

```ts
import type { AgentConfig, Manifest } from "./api";

export interface AppState {
  loading: boolean;
  error: string;
  config: AgentConfig | null;
  manifest: Manifest | null;
  serviceHealthy: boolean;
}

export const state: AppState = {
  loading: true,
  error: "",
  config: null,
  manifest: null,
  serviceHealthy: false,
};
```

Create `tauri-app/src/service.ts`:

```ts
import { getConfig, getManifest } from "./api";
import { state } from "./state";

export async function refreshState(): Promise<void> {
  state.loading = true;
  state.error = "";
  try {
    const [config, manifest] = await Promise.all([getConfig(), getManifest()]);
    state.config = config;
    state.manifest = manifest;
    state.serviceHealthy = true;
  } catch (err) {
    state.error = err instanceof Error ? err.message : String(err);
    state.serviceHealthy = false;
  } finally {
    state.loading = false;
  }
}
```

- [ ] **Step 2: Wire state into main**

Replace `tauri-app/src/main.ts`:

```ts
import "./styles.css";
import { refreshState } from "./service";
import { render } from "./ui";

async function boot() {
  await refreshState();
  render();
}

boot().catch((err) => {
  const app = document.querySelector<HTMLMainElement>("#app");
  if (app) {
    app.innerHTML = `<pre>${String(err)}</pre>`;
  }
});
```

- [ ] **Step 3: Add temporary UI renderer**

Create `tauri-app/src/ui.ts`:

```ts
import { state } from "./state";

export function render(): void {
  const app = document.querySelector<HTMLMainElement>("#app");
  if (!app) {
    throw new Error("missing #app root");
  }

  app.innerHTML = `
    <section class="shell">
      <header class="topbar">
        <div>
          <h1>AgentNotify</h1>
          <p>${state.manifest?.url ?? "localhost:17891"}</p>
        </div>
        <span class="status">${state.serviceHealthy ? "Running" : "Offline"}</span>
      </header>
      <section class="panel">
        <p>Style: ${state.config?.notificationStyle ?? "unknown"}</p>
      </section>
    </section>
  `;
}
```

- [ ] **Step 4: Verify frontend API client**

Run:

```bash
cd tauri-app
npm run build
```

Expected: TypeScript passes.

- [ ] **Step 5: Commit API client**

```bash
git add tauri-app/src/api.ts tauri-app/src/state.ts tauri-app/src/service.ts tauri-app/src/ui.ts tauri-app/src/main.ts
git commit -m "feat: add tauri http api client"
```

---

## Task 7: Build 2026 Command-First Auto-Theme UI

**Files:**
- Create: `tauri-app/src/commands.ts`
- Modify: `tauri-app/src/ui.ts`
- Modify: `tauri-app/src/styles.css`
- Modify: `tauri-app/src/service.ts`

- [ ] **Step 1: Add deterministic command parser**

Create `tauri-app/src/commands.ts`:

```ts
import {
  saveConfig,
  sendTestNotification,
  type AgentConfig,
  type EventName,
  type NotificationStyle,
} from "./api";

export interface CommandResult {
  message: string;
  config?: AgentConfig;
}

const styleAliases: Record<string, NotificationStyle> = {
  clean: "clean",
  status: "status-color",
  badge: "agent-badge",
  compact: "compact",
  card: "custom-card",
  custom: "custom-card",
};

export async function runCommand(input: string, config: AgentConfig | null): Promise<CommandResult> {
  const command = input.trim().toLowerCase();
  if (!command) return { message: "Type a command" };

  if (command === "test" || command === "send test") {
    await sendTestNotification("start");
    return { message: "Sent test notification" };
  }

  if (!config) return { message: "Config is not loaded yet" };

  if (command === "pause") {
    const next = { ...config, enabledEvents: [] as EventName[] };
    await saveConfig(next);
    return { message: "Paused notifications", config: next };
  }

  if (command === "resume") {
    const next = { ...config, enabledEvents: ["start", "stop"] as EventName[] };
    await saveConfig(next);
    return { message: "Resumed notifications", config: next };
  }

  const eventMatch = command.match(/^(start|stop)\s+(on|off)$/);
  if (eventMatch) {
    const event = eventMatch[1] as EventName;
    const enabled = eventMatch[2] === "on";
    const events = new Set(config.enabledEvents);
    if (enabled) events.add(event);
    else events.delete(event);
    const next = { ...config, enabledEvents: [...events] as EventName[] };
    await saveConfig(next);
    return { message: `${event} events ${enabled ? "enabled" : "disabled"}`, config: next };
  }

  const styleMatch = command.match(/^style\s+([a-z-]+)$/);
  if (styleMatch) {
    const style = styleAliases[styleMatch[1]];
    if (!style) return { message: `Unknown style: ${styleMatch[1]}` };
    const next = { ...config, notificationStyle: style };
    await saveConfig(next);
    return { message: `Switched style to ${style}`, config: next };
  }

  return { message: `Unknown command: ${input}` };
}
```

- [ ] **Step 2: Replace UI renderer with command client**

Replace `tauri-app/src/ui.ts`:

```ts
import { saveConfig, sendTestNotification, type EventName, type NotificationStyle } from "./api";
import { runCommand } from "./commands";
import { refreshState } from "./service";
import { state } from "./state";

const styles: NotificationStyle[] = ["clean", "status-color", "agent-badge", "compact", "custom-card"];
let commandMessage = "";

export function render(): void {
  const app = document.querySelector<HTMLMainElement>("#app");
  if (!app) {
    throw new Error("missing #app root");
  }

  const config = state.config;
  const currentStyle = config?.notificationStyle ?? "custom-card";
  const enabledEvents = config?.enabledEvents ?? ["start", "stop"];
  const isPaused = enabledEvents.length === 0;

  app.innerHTML = `
    <section class="shell">
      <header class="window-strip">
        <span></span><span></span><span></span>
        <strong>AgentNotify</strong>
      </header>

      <section class="command-row">
        <form class="command-box" data-command-form>
          <span class="search-mark">⌕</span>
          <input name="command" autocomplete="off" placeholder="Ask or search actions..." />
        </form>
        <span class="status ${state.serviceHealthy ? "is-running" : "is-offline"}">
          ${state.serviceHealthy ? "Running" : "Offline"}
        </span>
      </section>

      ${commandMessage ? `<section class="command-message">${escapeHtml(commandMessage)}</section>` : ""}
      ${state.error ? `<section class="notice">${escapeHtml(state.error)}</section>` : ""}

      <section class="workspace">
        <nav class="nav-rail" aria-label="Sections">
          <button class="nav-item active" title="Notifications">●</button>
          <button class="nav-item" title="Preview">▣</button>
          <button class="nav-item" title="Settings">⚙</button>
        </nav>

        <div>
          <section class="panel">
            <div class="section-label">Notification style</div>
            <div class="segmented">
              ${styles
                .map(
                  (style) => `
                    <button class="segment ${style === currentStyle ? "active" : ""}" data-style="${style}">
                      ${labelForStyle(style)}
                    </button>
                  `,
                )
                .join("")}
            </div>
          </section>

          <section class="preview-card">
            <div class="preview-top">
              <div class="avatar">C</div>
              <div class="preview-copy">
                <strong>${currentStyle === "custom-card" ? "Custom card preview" : "Native toast preview"}</strong>
                <span>agent-notification</span>
              </div>
            </div>
            <p>${previewText(currentStyle)}</p>
          </section>

          <section class="toggle-grid">
            ${eventToggle("start", enabledEvents.includes("start"))}
            ${eventToggle("stop", enabledEvents.includes("stop"))}
          </section>
        </div>

        <aside class="context-panel">
          <div class="section-label">Context</div>
          <dl>
            <dt>Mode</dt>
            <dd>${isPaused ? "Paused" : "Active"}</dd>
            <dt>Server</dt>
            <dd>${state.manifest?.url ?? "127.0.0.1:17891"}</dd>
            <dt>Version</dt>
            <dd>${state.manifest?.version ?? "unknown"}</dd>
          </dl>
          <button class="primary block" data-action="test">Test</button>
          <button class="block" data-action="refresh">Refresh</button>
        </aside>
      </section>
    </section>
  `;

  bindEvents();
}

function bindEvents(): void {
  document.querySelector<HTMLFormElement>("[data-command-form]")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const input = new FormData(form).get("command")?.toString() ?? "";
    const result = await runCommand(input, state.config);
    commandMessage = result.message;
    if (result.config) state.config = result.config;
    await refreshState();
    form.reset();
    render();
  });

  document.querySelectorAll<HTMLButtonElement>("[data-style]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.config) return;
      const style = button.dataset.style as NotificationStyle;
      state.config = { ...state.config, notificationStyle: style };
      render();
      await saveConfig(state.config);
      await refreshState();
      render();
    });
  });

  document.querySelectorAll<HTMLButtonElement>("[data-event]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.config) return;
      const event = button.dataset.event as EventName;
      const enabled = new Set(state.config.enabledEvents);
      if (enabled.has(event)) enabled.delete(event);
      else enabled.add(event);
      state.config = { ...state.config, enabledEvents: [...enabled] as EventName[] };
      render();
      await saveConfig(state.config);
      await refreshState();
      render();
    });
  });

  document.querySelector<HTMLButtonElement>('[data-action="test"]')?.addEventListener("click", async () => {
    await sendTestNotification("start");
  });

  document.querySelector<HTMLButtonElement>('[data-action="refresh"]')?.addEventListener("click", async () => {
    await refreshState();
    render();
  });
}

function eventToggle(event: EventName, enabled: boolean): string {
  return `
    <button class="event-toggle ${enabled ? "enabled" : ""}" data-event="${event}">
      <span>${event === "start" ? "Start events" : "Stop events"}</span>
      <strong>${enabled ? "On" : "Off"}</strong>
    </button>
  `;
}

function labelForStyle(style: NotificationStyle): string {
  const labels: Record<NotificationStyle, string> = {
    clean: "Clean",
    "status-color": "Status",
    "agent-badge": "Badge",
    compact: "Compact",
    "custom-card": "Card",
  };
  return labels[style];
}

function previewText(style: NotificationStyle): string {
  if (style === "custom-card") return "Generated PNG hero card inside a native Windows toast.";
  if (style === "compact") return "One-line native toast for high-frequency events.";
  return "Native toast layout managed by the Go notification engine.";
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => {
    const map: Record<string, string> = {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#039;",
    };
    return map[char];
  });
}
```

- [ ] **Step 3: Replace styles with auto-theme command UI**

Replace `tauri-app/src/styles.css`:

```css
:root {
  color-scheme: light dark;
  --bg: #f6f7f9;
  --panel: #ffffff;
  --panel-2: #f1f3f6;
  --text: #1f2328;
  --muted: #667085;
  --border: #e3e6eb;
  --accent: #2563eb;
  --success-bg: #e8f7ef;
  --success-text: #0f7a43;
  --success-border: #bde8ce;
  --preview: #111827;
  --preview-muted: #cbd5e1;
  --rail: #ffffff;
  font-family: "Segoe UI", system-ui, sans-serif;
  background: var(--bg);
  color: var(--text);
}

@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0f141b;
    --panel: #151c25;
    --panel-2: #101722;
    --text: #f8fafc;
    --muted: #94a3b8;
    --border: #263241;
    --accent: #7aa2ff;
    --success-bg: #123524;
    --success-text: #7ee0a1;
    --success-border: #245c3b;
    --preview: #111827;
    --preview-muted: #cbd5e1;
    --rail: #111821;
  }
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-width: 460px;
  min-height: 560px;
  background: var(--bg);
}

button {
  font: inherit;
}

.shell {
  padding: 14px;
}

.window-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  margin-bottom: 10px;
}

.window-strip span {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}

.window-strip span:nth-child(1) { background: #ef4444; }
.window-strip span:nth-child(2) { background: #f59e0b; }
.window-strip span:nth-child(3) { background: #22c55e; }

.window-strip strong {
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  margin-left: auto;
}

.command-row {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}

.command-box {
  align-items: center;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex: 1;
  height: 42px;
  padding: 0 12px;
}

.search-mark {
  color: var(--muted);
  font-size: 15px;
  margin-right: 8px;
}

.command-box input {
  background: transparent;
  border: 0;
  color: var(--text);
  flex: 1;
  font: inherit;
  font-size: 13px;
  outline: none;
}

.command-box input::placeholder {
  color: var(--muted);
}

.status {
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 12px;
  padding: 5px 9px;
}

.status.is-running {
  border-color: var(--success-border);
  background: var(--success-bg);
  color: var(--success-text);
}

.notice {
  margin-bottom: 12px;
  border: 1px solid #f0c6c6;
  border-radius: 10px;
  background: #fff1f1;
  color: #8a1f1f;
  padding: 10px;
  font-size: 12px;
}

.command-message {
  background: color-mix(in srgb, var(--accent) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 28%, transparent);
  border-radius: 10px;
  color: var(--text);
  font-size: 12px;
  margin-bottom: 10px;
  padding: 9px 10px;
}

.workspace {
  display: grid;
  grid-template-columns: 52px 1fr 118px;
  gap: 10px;
}

.nav-rail {
  align-content: start;
  background: var(--rail);
  border: 1px solid var(--border);
  border-radius: 14px;
  display: grid;
  gap: 8px;
  padding: 8px;
}

.nav-item {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: 10px;
  color: var(--muted);
  cursor: pointer;
  display: flex;
  height: 36px;
  justify-content: center;
}

.nav-item.active {
  background: var(--panel-2);
  color: var(--text);
}

.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px;
  margin-bottom: 12px;
}

.section-label {
  color: var(--muted);
  font-size: 12px;
  margin-bottom: 8px;
}

.segmented {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 4px;
  background: var(--panel-2);
  border-radius: 9px;
  padding: 4px;
}

.segment {
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 11px;
  min-height: 30px;
}

.segment.active {
  background: var(--panel);
  color: var(--text);
  box-shadow: 0 1px 2px rgb(0 0 0 / 8%);
}

.preview-card {
  background: var(--preview);
  border: 1px solid rgb(255 255 255 / 10%);
  border-radius: 13px;
  color: white;
  padding: 14px;
  margin-bottom: 12px;
  box-shadow: 0 14px 32px rgb(0 0 0 / 18%);
}

.preview-top {
  align-items: center;
  display: flex;
  gap: 12px;
}

.avatar {
  align-items: center;
  background: #22c55e;
  border-radius: 12px;
  color: #062b16;
  display: flex;
  font-weight: 800;
  height: 42px;
  justify-content: center;
  width: 42px;
}

.preview-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.preview-copy strong {
  font-size: 15px;
}

.preview-copy span,
.preview-card p {
  color: var(--preview-muted);
  font-size: 12px;
}

.preview-card p {
  margin-top: 12px;
}

.toggle-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}

.event-toggle {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  color: var(--text);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  min-height: 42px;
  padding: 10px;
}

.event-toggle span,
.event-toggle strong {
  font-size: 13px;
}

.event-toggle.enabled strong {
  color: var(--success-text);
}

.context-panel {
  background: var(--rail);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 10px;
}

.context-panel dl {
  margin: 0 0 12px;
}

.context-panel dt {
  color: var(--muted);
  font-size: 11px;
  margin-top: 10px;
}

.context-panel dd {
  color: var(--text);
  font-size: 12px;
  margin: 3px 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.actions {
  display: flex;
  gap: 8px;
}

.actions button {
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--panel);
  color: var(--text);
  cursor: pointer;
  min-height: 34px;
  padding: 0 12px;
}

.actions button.primary {
  background: var(--accent);
  border-color: var(--accent);
  color: white;
}

.block {
  width: 100%;
  margin-top: 8px;
}
```

- [ ] **Step 4: Verify UI build**

Run:

```bash
cd tauri-app
npm run build
```

Expected: TypeScript and Vite build pass.

- [ ] **Step 5: Verify UI manually**

Run:

```bash
cd tauri-app
npm run tauri:dev
```

Expected:
- Window follows OS light/dark theme.
- Command bar executes `test`, `pause`, `resume`, `start off`, `start on`, `stop off`, `stop on`, `style clean`, and `style card`.
- Style selector saves immediately.
- Event toggles save immediately.
- Test button sends toast.
- Refresh reloads Go config.

- [ ] **Step 6: Commit UI**

```bash
git add tauri-app/src/commands.ts tauri-app/src/ui.ts tauri-app/src/styles.css tauri-app/src/service.ts
git commit -m "feat: add command-first tauri client ui"
```

---

## Task 8: Add Service Error States And Recovery Actions

**Files:**
- Modify: `tauri-app/src-tauri/src/service.rs`
- Modify: `tauri-app/src-tauri/src/main.rs`
- Modify: `tauri-app/src/ui.ts`
- Modify: `tauri-app/src/api.ts`
- Modify: `tauri-app/src/commands.ts`

- [ ] **Step 1: Add Rust commands to restart service**

Modify `tauri-app/src-tauri/src/service.rs` by adding:

```rust
#[tauri::command]
pub fn restart_service(app: AppHandle) -> Result<(), String> {
    let state = app.state::<ServiceState>();
    stop_sidecar(&state);
    ensure_sidecar(&app)
}
```

Modify `tauri-app/src-tauri/src/main.rs` invoke handler:

```rust
.invoke_handler(tauri::generate_handler![
    service::service_status,
    service::restart_service
])
```

- [ ] **Step 2: Add frontend command call**

Modify `tauri-app/src/api.ts`:

```ts
import { invoke } from "@tauri-apps/api/core";

export async function restartService(): Promise<void> {
  await invoke("restart_service");
}
```

Modify `tauri-app/src/ui.ts` imports:

```ts
import { restartService, saveConfig, sendTestNotification, type EventName, type NotificationStyle } from "./api";
```

Modify `tauri-app/src/commands.ts` imports:

```ts
import {
  restartService,
  saveConfig,
  sendTestNotification,
  type AgentConfig,
  type EventName,
  type NotificationStyle,
} from "./api";
```

Add this branch before the config-loaded guard:

```ts
if (command === "restart") {
  await restartService();
  return { message: "Restarted service" };
}
```

Update `bindEvents()`:

```ts
document.querySelector<HTMLButtonElement>('[data-action="restart"]')?.addEventListener("click", async () => {
  await restartService();
  await refreshState();
  render();
});
```

Update context action markup:

```html
<button class="primary block" data-action="test">Test</button>
<button class="block" data-action="restart">Restart</button>
<button class="block" data-action="refresh">Refresh</button>
```

- [ ] **Step 3: Verify restart action**

Run:

```bash
cd tauri-app
npm run build
cd src-tauri
cargo check
```

Expected: both pass.

Manual:

```bash
cd tauri-app
npm run tauri:dev
```

Click `Restart`. Expected: service remains healthy after restart.

Type `restart` in the command bar. Expected: service remains healthy after restart and the command message says `Restarted service`.

- [ ] **Step 4: Commit recovery actions**

```bash
git add tauri-app/src-tauri/src/service.rs tauri-app/src-tauri/src/main.rs tauri-app/src/api.ts tauri-app/src/ui.ts tauri-app/src/commands.ts
git commit -m "feat: add tauri service recovery actions"
```

---

## Task 9: Build And Package

**Files:**
- Modify: `README.md`
- Modify: `windows-server/README.md`
- Create: `tauri-app/README.md`

- [ ] **Step 1: Add Tauri README**

Create `tauri-app/README.md`:

```markdown
# AgentNotify Tauri Client

Tauri v2 desktop client for AgentNotify.

## Development

```bash
npm install
npm run tauri:dev
```

`npm run tauri:dev` builds a Go sidecar for the current Rust target triple and starts the Tauri app.

## Build

```bash
npm run tauri:build
```

The final app bundle includes the Go server sidecar.

## Architecture

Tauri owns the tray and command window. The Go server remains the notification engine and listens on `127.0.0.1:17891`.
```

- [ ] **Step 2: Update root README**

Add a section to `README.md`:

```markdown
## Desktop Client

The Tauri client lives in `tauri-app/`. It bundles the Go server as a sidecar, adds a system tray, and provides the command-first notification client UI.

```bash
cd tauri-app
npm install
npm run tauri:dev
```
```

- [ ] **Step 3: Build package**

Run:

```bash
cd tauri-app
npm run tauri:build
```

Expected:
- Frontend build passes.
- Rust build passes.
- Tauri bundle is created.
- Bundle contains sidecar binary.

- [ ] **Step 4: Commit docs**

```bash
git add README.md windows-server/README.md tauri-app/README.md
git commit -m "docs: add tauri client usage"
```

---

## Task 10: Windows Manual Verification

**Files:**
- No source changes expected.

- [ ] **Step 1: Build Windows Tauri bundle**

On Windows:

```powershell
cd D:\project\agent-notification\tauri-app
npm install
npm run tauri:build
```

Expected: Windows installer/app bundle produced under `src-tauri\target\release\bundle`.

- [ ] **Step 2: Launch app**

Run the built app.

Expected:
- No console window remains visible.
- Tray icon appears.
- Main command window is hidden at launch.
- Left-click tray opens command window.
- Right-click tray opens menu.

- [ ] **Step 3: Verify sidecar**

Run:

```powershell
curl http://127.0.0.1:17891/health
curl http://127.0.0.1:17891/manifest
```

Expected:
- `/health` returns ok.
- `/manifest` includes `custom-card`.

- [ ] **Step 4: Verify command UI**

In the Tauri window:
- Select `Clean`.
- Click `Test`.
- Select `Card`.
- Click `Test`.
- Toggle Start off.
- Send a start event from curl.
- Toggle Start on.
- Send a start event from curl again.

Commands:

```powershell
curl -Method POST http://127.0.0.1:17891/notify `
  -ContentType "application/json" `
  -Body '{"agent":"manual","event":"start","project":"AgentNotify","message":"manual test","sourcePayload":{}}'
```

Expected:
- Style changes apply without restarting.
- Event toggles apply without restarting.
- Custom card PNG updates under `%LOCALAPPDATA%\AgentNotify\toast-card.png`.

- [ ] **Step 5: Verify quit behavior**

Right-click tray, choose `Quit`.

Expected:
- Tauri exits.
- Go sidecar process launched by Tauri exits.
- If an external Go server was already running before Tauri launch, Tauri does not kill it.

---

## Verification Summary

Run before claiming complete:

```bash
cd windows-server
go test ./... -v

cd ../tauri-app
npm install
npm run build
npm run prepare:sidecar
cd src-tauri
cargo check
```

Manual:

```bash
cd tauri-app
npm run tauri:dev
```

Windows final:

```powershell
cd D:\project\agent-notification\tauri-app
npm run tauri:build
```

Success means:
- Tauri app bundles and starts the Go sidecar.
- Tray menu works.
- Command-first auto-theme UI controls Go config.
- Notification test works.
- Existing Go API behavior remains compatible.
