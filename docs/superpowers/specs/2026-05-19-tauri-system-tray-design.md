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
