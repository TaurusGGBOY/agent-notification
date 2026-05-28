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

Tauri owns the tray and command window. The Go server remains the notification engine, is bundled as a sidecar, listens on `0.0.0.0:17891` for LAN agents, and is controlled locally by the Tauri UI through `127.0.0.1:17891`.
