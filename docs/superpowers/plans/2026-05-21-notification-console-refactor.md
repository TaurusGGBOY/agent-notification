# Notification Console Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the single-screen AgentNotify console redesign with LAN URL display, mDNS broadcast toggle, differentiated previews, and recent notification history.

**Architecture:** The Go server owns runtime service data: LAN manifest URL, mDNS broadcast state, and in-memory notification history. The Tauri frontend loads config, manifest, broadcast, and history, then renders a single no-scroll control surface. Existing toast delivery and settings persistence remain unchanged.

**Tech Stack:** Go net/http, zeroconf mDNS, Tauri v2, TypeScript, Vite, CSS custom properties.

---

### Task 1: Backend Runtime State

**Files:**
- Modify: `windows-server/handlers.go`
- Modify: `windows-server/mdns.go`
- Modify: `windows-server/main.go`

- [ ] Add LAN IPv4 detection and make `/manifest.url` prefer `http://<lan-ip>:17891`.
- [ ] Add `NotificationHistoryItem`, record recent 3 valid `/notify` requests, and expose `GET /history`.
- [ ] Add `BroadcastController` with `SetEnabled`, `Enabled`, and `BroadcastHandler`.
- [ ] Start broadcast enabled by default in `main.go`.
- [ ] Register `/history` and `/broadcast`.
- [ ] Run `cd windows-server && go test ./... -v`.

### Task 2: Frontend Data API

**Files:**
- Modify: `tauri-app/src/api.ts`
- Modify: `tauri-app/src/state.ts`
- Modify: `tauri-app/src/service.ts`

- [ ] Remove `custom-card` from the frontend selectable `NotificationStyle`.
- [ ] Add history and broadcast API functions.
- [ ] Extend app state for history, history errors, broadcast state, and broadcast errors.
- [ ] Load config/manifest first; load history/broadcast independently so failures do not blank the UI.
- [ ] Run `cd tauri-app && npm run build`.

### Task 3: Frontend UI Redesign

**Files:**
- Modify: `tauri-app/src/ui.ts`
- Modify: `tauri-app/src/styles.css`

- [ ] Remove command/search UI and command handler.
- [ ] Left panel shows service status, LAN URL, and version in the lower section.
- [ ] Remove event mode, event switch, repeated style labels, and `custom-card`.
- [ ] Add differentiated preview markup for clean, status, badge, and compact.
- [ ] Replace restart button with mDNS broadcast toggle.
- [ ] Add notification history module with latest 3 items.
- [ ] Keep the app single-screen with no internal scrollbar.
- [ ] Run `cd tauri-app && npm run build`.

### Task 4: Windows Build and Screenshot Verification

**Files:**
- Verify: Windows release build output.

- [ ] Sync changed files to `<user>@<host>`.
- [ ] Run `cd D:\project\agent-notification\windows-server; go test ./... -v`.
- [ ] Run `cd D:\project\agent-notification\tauri-app; npm run tauri:build`.
- [ ] Capture screenshot with `skills/windows-ui-screenshot/scripts/capture_windows_ui.py`.
- [ ] Verify screenshot against the spec checklist.

### Self-Review

- Covers all 10 requested UI changes.
- Keeps existing notification and settings flows compatible.
- Adds only the backend endpoints needed by the new UI.
- Keeps history in memory as specified.
