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