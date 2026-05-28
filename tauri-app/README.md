# AgentNotify Tauri Client

Tauri v2 desktop client for AgentNotify.

## Windows One-Click App

Windows users should install and start the Tauri app from the release package. The app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agents, and advertises itself with mDNS.

After opening the app, copy the LAN URL shown in the sidebar into the Claude/Codex setup skill, or let the skill discover the server automatically. The Tauri UI controls the server locally through `127.0.0.1:17891`.

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

Tauri owns the tray and command window. The Go server remains the notification engine, is bundled as a sidecar, listens on `0.0.0.0:17891` for LAN agents, and is controlled locally by the Tauri UI through `127.0.0.1:17891`.
