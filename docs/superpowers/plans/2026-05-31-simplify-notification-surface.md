# Simplify Notification Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reintroduce the useful parts of the reverted notification simplification without restoring the Mac runtime scaling regression.

**Architecture:** Keep notification styling as a server-side compatibility field but expose and persist only `clean`. Remove style and preview controls from both the Tauri shell and legacy Windows settings page, collapse toast XML rendering to one clean layout, hide Windows helper windows, and preserve the existing 1200x675 fixed shell.

**Tech Stack:** TypeScript renderer tests with `node:test`, Tauri Rust unit tests, Go HTTP server and toast unit tests.

---

### Task 1: Frontend Shell Simplification

**Files:**
- Modify: `tauri-app/tests/layout.test.mjs`
- Modify: `tauri-app/src/ui.ts`
- Modify: `tauri-app/src/styles.css`
- Modify: `tauri-app/src/api.ts`
- Modify: `tauri-app/src/commands.ts`

- [ ] **Step 1: Write failing tests**

Update the layout tests so the Tauri UI must not contain style selectors, preview panels, style command handling, or runtime scale logic:

```js
assert.doesNotMatch(ui, /通知样式|通知预览|data-style|previewMarkup|previewText/);
assert.doesNotMatch(styles, /style-card|preview-panel|toast-preview|runtime-scale/);
assert.doesNotMatch(commands, /styleAliases|style\s*\(/);
assert.match(api, /export type NotificationStyle = "clean";/);
```

- [ ] **Step 2: Run test to verify RED**

Run: `cd tauri-app && npm test`

Expected: FAIL because current `ui.ts`, `styles.css`, and `commands.ts` still expose style and preview behavior.

- [ ] **Step 3: Implement minimal frontend change**

Remove `saveConfig` and `NotificationStyle` from `ui.ts`, delete the style card, preview panel, style click binding, and helper functions. Change the topbar subtitle to omit "styles". Narrow `NotificationStyle` in `api.ts` to `"clean"`. Remove the `style ...` command path from `commands.ts`. Delete unused preview/style CSS selectors.

- [ ] **Step 4: Run test to verify GREEN**

Run: `cd tauri-app && npm test`

Expected: PASS, including the existing checks for `1200x675` and absence of `runtime-scale`.

### Task 2: Server Style Compatibility Collapse

**Files:**
- Modify: `windows-server/config.go`
- Modify: `windows-server/handlers.go`
- Modify: `windows-server/settings.go`
- Modify: `windows-server/toast_xml.go`
- Modify: `windows-server/coverage_test.go`
- Modify: `windows-server/discovery_test.go`
- Modify: `windows-server/windows_test.go`

- [ ] **Step 1: Write failing tests**

Update Go tests to require `supportedStyles()` and mDNS TXT to expose only `clean`, unsupported or old styles to normalize to `clean`, and `formatToastXML` to render the same clean XML for every style input.

- [ ] **Step 2: Run test to verify RED**

Run: `cd windows-server && go test ./...`

Expected: FAIL because the server currently exposes and renders multiple styles.

- [ ] **Step 3: Implement minimal server change**

Make `IsSupportedStyle` and legacy `validStyles` accept only `clean`; make `/manifest` supported styles return only `clean`; remove style cards and preview markup from the legacy settings page and save `notificationStyle: "clean"`; collapse `formatToastXML` to a single clean layout.

- [ ] **Step 4: Run test to verify GREEN**

Run: `cd windows-server && go test ./...`

Expected: PASS.

### Task 3: Windows Notification and Helper Window Behavior

**Files:**
- Modify: `tauri-app/scripts/prepare-sidecar.mjs`
- Modify: `tauri-app/src-tauri/src/notification_settings.rs`
- Modify: `tauri-app/src/ui.ts`
- Modify: `windows-server/toast_card.go`
- Modify: `windows-server/toast_windows.go`
- Modify: `windows-server/toast_windows_test.go`
- Modify: `windows-server/coverage_test.go`

- [ ] **Step 1: Write failing tests**

Add tests for global `ToastEnabled` parsing, default-on missing registry values, hidden PowerShell command flags, logo override XML, and `prepare-sidecar.mjs` using `-H=windowsgui` for Windows targets.

- [ ] **Step 2: Run test to verify RED**

Run: `cd tauri-app && npm test`, `cd tauri-app/src-tauri && cargo test`, and `cd windows-server && go test ./...`

Expected: FAIL because hidden window flags, global notification parsing, and logo override behavior are incomplete.

- [ ] **Step 3: Implement minimal behavior**

Add Windows `-ldflags -H=windowsgui` to sidecar builds, add global toast registry parsing and polling after opening Windows notification settings, generate a reusable app logo PNG, pass it as `appLogoOverride`, and send PowerShell through a hidden non-interactive command.

- [ ] **Step 4: Run test to verify GREEN**

Run the same three commands.

Expected: PASS.

### Task 4: Final Verification and PR

**Files:**
- Verify all modified files.

- [ ] **Step 1: Run full checks**

Run:

```bash
cd tauri-app && npm test
cd tauri-app && npm run build
cd tauri-app && npm run prepare:sidecar
cd tauri-app/src-tauri && cargo test
cd tauri-app/src-tauri && cargo build
cd windows-server && go test ./...
```

- [ ] **Step 2: Push and open PR**

Push `feature/simplify-notification-surface` and open a PR to `main`.

- [ ] **Step 3: Review and merge**

Inspect the PR diff, address blocking review/check failures, rebase on latest `main`, merge the PR, update local `main`, then remove the feature worktree and local feature branch.
