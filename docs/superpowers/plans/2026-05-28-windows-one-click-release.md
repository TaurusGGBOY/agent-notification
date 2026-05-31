# Windows One-Click Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make AgentNotify shippable to Windows users as a one-click desktop app that opens a UI, starts the notification server on the LAN, and is released automatically from `v*` tags.

**Architecture:** The Tauri app remains the user-facing executable and bundles the Go server as a sidecar. The sidecar must listen on `0.0.0.0:17891` so Mac/Codex/Claude machines on the LAN can reach it, while the Tauri UI continues using `127.0.0.1:17891` for local control. GitHub Actions builds Windows artifacts on `windows-latest`, creates a GitHub Release on tag push, and a Windows smoke path verifies the generated `.exe` opens the UI and starts the server.

**Tech Stack:** Tauri v2, Rust 1.84+, Go, Node 22, GitHub Actions, PowerShell, Windows runner/Windows dev host.

---

## File Structure

- Modify: `tauri-app/src-tauri/src/service.rs`
  - Add explicit constants/helpers for control address and sidecar listen address.
  - Use `0.0.0.0:17891` only for the sidecar process environment.
  - Keep health checks on `127.0.0.1:17891`.
- Modify: `tauri-app/README.md`
  - Document one-click behavior and LAN listener.
- Modify: `README.md`
  - Document Windows user install flow and tag release flow.
- Create: `.github/workflows/release.yml`
  - Build Windows installer/app artifacts on tag push.
  - Upload Tauri bundles and Go server debug binaries to GitHub Release.
- Create: `scripts/verify-windows-release.ps1`
  - Run on Windows with a built `agent-notify.exe`.
  - Start the app, wait for UI process and `/health`, verify `/manifest` exposes a LAN URL, take process evidence, then stop it.
- Create: `docs/release-checklist.md`
  - Human release steps: tag, Actions, download installer, smoke test on Windows.

## Known Baseline

- Existing worktree: `/Users/<username>/project/agent-notification/.worktrees/windows-one-click-release`
- Branch: `feature/windows-one-click-release`
- Baseline Go tests passed on Mac: `66 passed`
- Baseline frontend build passed on Mac.
- Baseline Tauri Rust tests could not run locally because `cargo`/`rustc` are not installed on this Mac. Rust/Tauri verification must run on Windows CI or a Windows dev host.

## Task 1: Make Sidecar Listen On LAN While UI Uses Localhost

**Files:**
- Modify: `tauri-app/src-tauri/src/service.rs`
- Test: `tauri-app/src-tauri/src/service.rs`

- [ ] **Step 1: Write failing Rust tests for address helpers**

Add this test module at the end of `tauri-app/src-tauri/src/service.rs`:

```rust
#[cfg(test)]
mod tests {
    use super::{control_addr, sidecar_listen_addr};

    #[test]
    fn sidecar_listens_on_all_interfaces_for_lan_agents() {
        assert_eq!(sidecar_listen_addr(), "0.0.0.0:17891");
    }

    #[test]
    fn tauri_controls_server_through_loopback() {
        assert_eq!(control_addr(), "127.0.0.1:17891");
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run on a machine with Rust installed:

```bash
cd tauri-app/src-tauri
cargo test sidecar_listens_on_all_interfaces_for_lan_agents tauri_controls_server_through_loopback -- --nocapture
```

Expected: fail with unresolved imports `control_addr` and `sidecar_listen_addr`.

- [ ] **Step 3: Add address helper functions**

Add near the top of `tauri-app/src-tauri/src/service.rs`:

```rust
const CONTROL_ADDR: &str = "127.0.0.1:17891";
const SIDECAR_LISTEN_ADDR: &str = "0.0.0.0:17891";

pub fn control_addr() -> &'static str {
    CONTROL_ADDR
}

pub fn sidecar_listen_addr() -> &'static str {
    SIDECAR_LISTEN_ADDR
}
```

- [ ] **Step 4: Use helpers in service lifecycle**

Change:

```rust
let client = match Client::new("127.0.0.1:17891", Duration::from_millis(600)) {
```

to:

```rust
let client = match Client::new(control_addr(), Duration::from_millis(600)) {
```

Change:

```rust
.env("AGENT_NOTIFY_HTTP_ADDR", "127.0.0.1:17891")
```

to:

```rust
.env("AGENT_NOTIFY_HTTP_ADDR", sidecar_listen_addr())
```

- [ ] **Step 5: Run Rust tests**

```bash
cd tauri-app/src-tauri
cargo test --all -- --nocapture
```

Expected: all Rust tests pass, including the two new address tests.

- [ ] **Step 6: Commit**

```bash
git add tauri-app/src-tauri/src/service.rs
git commit -m "fix: expose bundled server on lan"
```

## Task 2: Document One-Click Windows Behavior

**Files:**
- Modify: `README.md`
- Modify: `tauri-app/README.md`

- [ ] **Step 1: Write doc regression test**

Create a shell check command and run it before editing docs:

```bash
python3 - <<'PY'
from pathlib import Path
readme = Path("README.md").read_text()
tauri = Path("tauri-app/README.md").read_text()
assert "one-click Windows app" in readme
assert "0.0.0.0:17891" in tauri
PY
```

Expected: fail with `AssertionError`.

- [ ] **Step 2: Update root README**

Add this section after Quick Start:

```markdown
## Windows One-Click App

Windows users should install and start the Tauri app. The app opens the AgentNotify UI, starts the bundled Go server sidecar, listens on `0.0.0.0:17891` for LAN agent notifications, and advertises itself with mDNS.

After opening the app, copy the LAN URL shown in the sidebar into the Claude/Codex setup skill, or let the skill discover the server automatically.
```

- [ ] **Step 3: Update Tauri README**

Replace the Architecture paragraph:

```markdown
Tauri owns the tray and command window. The Go server remains the notification engine and listens on `127.0.0.1:17891`.
```

with:

```markdown
Tauri owns the tray and command window. The Go server remains the notification engine, is bundled as a sidecar, listens on `0.0.0.0:17891` for LAN agents, and is controlled locally by the Tauri UI through `127.0.0.1:17891`.
```

- [ ] **Step 4: Run doc regression test**

```bash
python3 - <<'PY'
from pathlib import Path
readme = Path("README.md").read_text()
tauri = Path("tauri-app/README.md").read_text()
assert "one-click Windows app" in readme
assert "0.0.0.0:17891" in tauri
PY
```

Expected: no output, exit 0.

- [ ] **Step 5: Commit**

```bash
git add README.md tauri-app/README.md
git commit -m "docs: describe windows one-click app"
```

## Task 3: Add Windows Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write failing workflow existence check**

```bash
python3 - <<'PY'
from pathlib import Path
text = Path(".github/workflows/release.yml").read_text()
assert "tags:" in text
assert "v*" in text
assert "windows-latest" in text
assert "npm run tauri:build" in text
assert "softprops/action-gh-release" in text
PY
```

Expected: fail with `FileNotFoundError`.

- [ ] **Step 2: Create release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"
  workflow_dispatch:
    inputs:
      tag:
        description: "Existing tag to release, for manual reruns"
        required: true
        type: string

permissions:
  contents: write

concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false

jobs:
  windows:
    name: Build Windows release
    runs-on: windows-latest
    defaults:
      run:
        shell: pwsh

    steps:
      - name: Checkout tag
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.inputs.tag || github.ref }}

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: windows-server/go.mod
          cache-dependency-path: windows-server/go.sum

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: tauri-app/package-lock.json

      - name: Set up Rust
        uses: dtolnay/rust-toolchain@stable
        with:
          toolchain: stable

      - name: Cache Rust
        uses: swatinem/rust-cache@v2
        with:
          workspaces: tauri-app/src-tauri

      - name: Install frontend dependencies
        working-directory: tauri-app
        run: npm ci

      - name: Run Go tests
        working-directory: windows-server
        run: go test ./... -v

      - name: Build Tauri Windows app
        working-directory: tauri-app
        run: npm run tauri:build

      - name: Build standalone server debug binaries
        working-directory: windows-server
        run: |
          $env:GOOS = "windows"
          $env:GOARCH = "amd64"
          go build -o agent-notify-server.exe
          $env:GOARCH = "arm64"
          go build -o agent-notify-server-arm64.exe

      - name: Collect release assets
        run: |
          New-Item -ItemType Directory -Force dist-release | Out-Null
          Get-ChildItem tauri-app/src-tauri/target/release/bundle -Recurse -File |
            Where-Object { $_.Extension -in ".exe", ".msi" } |
            Copy-Item -Destination dist-release
          Copy-Item windows-server/agent-notify-server.exe dist-release/
          Copy-Item windows-server/agent-notify-server-arm64.exe dist-release/
          Get-ChildItem dist-release | Format-Table Name,Length

      - name: Upload workflow artifacts
        uses: actions/upload-artifact@v4
        with:
          name: agentnotify-windows
          path: dist-release/*
          if-no-files-found: error

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ github.event.inputs.tag || github.ref_name }}
          name: AgentNotify ${{ github.event.inputs.tag || github.ref_name }}
          generate_release_notes: true
          files: dist-release/*
```

- [ ] **Step 3: Run workflow static check**

```bash
python3 - <<'PY'
from pathlib import Path
text = Path(".github/workflows/release.yml").read_text()
assert "tags:" in text
assert "\"v*\"" in text
assert "windows-latest" in text
assert "npm run tauri:build" in text
assert "softprops/action-gh-release@v2" in text
assert "dist-release/*" in text
PY
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add windows release workflow"
```

## Task 4: Add Windows Smoke Verification Script

**Files:**
- Create: `scripts/verify-windows-release.ps1`

- [ ] **Step 1: Write failing script existence check**

```bash
python3 - <<'PY'
from pathlib import Path
text = Path("scripts/verify-windows-release.ps1").read_text()
assert "Start-Process" in text
assert "http://127.0.0.1:17891/health" in text
assert "http://127.0.0.1:17891/manifest" in text
assert "Stop-Process" in text
PY
```

Expected: fail with `FileNotFoundError`.

- [ ] **Step 2: Create verification script**

Create `scripts/verify-windows-release.ps1`:

```powershell
param(
  [Parameter(Mandatory = $true)]
  [string]$ExePath,
  [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ExePath)) {
  throw "Executable not found: $ExePath"
}

$process = Start-Process -FilePath $ExePath -PassThru
try {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $health = $null
  while ((Get-Date) -lt $deadline) {
    try {
      $health = Invoke-RestMethod -Uri "http://127.0.0.1:17891/health" -TimeoutSec 2
      if ($health.status -eq "ok") {
        break
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  }

  if (-not $health -or $health.status -ne "ok") {
    throw "AgentNotify server did not become healthy"
  }

  $manifest = Invoke-RestMethod -Uri "http://127.0.0.1:17891/manifest" -TimeoutSec 5
  if (-not $manifest.url -or $manifest.url -notmatch "^http://.+:17891$") {
    throw "Manifest URL is invalid: $($manifest.url)"
  }
  if ($manifest.url -match "127\.0\.0\.1|localhost") {
    throw "Manifest URL must be LAN reachable, got: $($manifest.url)"
  }

  $agentProcess = Get-Process -Id $process.Id -ErrorAction Stop
  if (-not $agentProcess.MainWindowHandle) {
    throw "AgentNotify process started but no main window handle was detected"
  }

  Write-Host "AgentNotify UI process started: pid=$($process.Id)"
  Write-Host "Server health: $($health.status)"
  Write-Host "Manifest URL: $($manifest.url)"
}
finally {
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Get-Process agent-notify-server -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
```

- [ ] **Step 3: Run static script check**

```bash
python3 - <<'PY'
from pathlib import Path
text = Path("scripts/verify-windows-release.ps1").read_text()
assert "Start-Process" in text
assert "http://127.0.0.1:17891/health" in text
assert "http://127.0.0.1:17891/manifest" in text
assert "Stop-Process" in text
PY
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add scripts/verify-windows-release.ps1
git commit -m "test: add windows release smoke check"
```

## Task 5: Build And Smoke Test On Windows

**Files:**
- No source changes expected.

- [ ] **Step 1: Push branch to GitHub**

```bash
git push -u origin feature/windows-one-click-release
```

Expected: branch pushed.

- [ ] **Step 2: Run CI or Windows build locally/remotely**

On a Windows machine with Go, Node, Rust, and WebView2 build prerequisites:

```powershell
cd agent-notification\tauri-app
npm ci
npm run tauri:build
```

Expected: Tauri prints a built executable path under:

```text
agent-notification\tauri-app\src-tauri\target\release\agent-notify.exe
```

Expected bundles under:

```text
agent-notification\tauri-app\src-tauri\target\release\bundle
```

- [ ] **Step 3: Verify `.exe` starts UI and server**

On Windows:

```powershell
cd agent-notification
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-windows-release.ps1 `
  -ExePath .\tauri-app\src-tauri\target\release\agent-notify.exe
```

Expected output includes:

```text
AgentNotify UI process started
Server health: ok
Manifest URL: http://<lan-ip>:17891
```

- [ ] **Step 4: Verify LAN reachability from Mac**

From Mac, replacing `<lan-ip>` with the manifest URL host:

```bash
curl -fsS http://<lan-ip>:17891/health
curl -fsS http://<lan-ip>:17891/manifest
curl -fsS -X POST http://<lan-ip>:17891/notify \
  -H 'Content-Type: application/json' \
  -d '{"agent":"codex","event":"stop","project":"release-smoke","message":"LAN smoke test","sourcePayload":{}}'
```

Expected:

```text
{"status":"ok","version":"1.0.0"}
```

Windows should show a notification for `release-smoke`.

## Task 6: Tag Release Smoke

**Files:**
- Modify: `docs/release-checklist.md`

- [ ] **Step 1: Create release checklist**

Create `docs/release-checklist.md`:

```markdown
# Release Checklist

1. Merge release-ready branch to `main`.
2. Create a version tag:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
3. Open GitHub Actions and wait for the `Release` workflow to pass.
4. Open the generated GitHub Release.
5. Download the Windows installer or setup executable.
6. Install on Windows.
7. Open AgentNotify from Start Menu or desktop.
8. Confirm UI appears.
9. Confirm `http://127.0.0.1:17891/health` returns `{"status":"ok"}`.
10. Confirm `http://<windows-lan-ip>:17891/health` works from Mac.
11. Send a test notification from Mac with `scripts/send.py` or `curl`.
```

- [ ] **Step 2: Run checklist static check**

```bash
python3 - <<'PY'
from pathlib import Path
text = Path("docs/release-checklist.md").read_text()
assert "git tag v0.1.0" in text
assert "Release" in text
assert "http://<windows-lan-ip>:17891/health" in text
assert "Confirm UI appears" in text
PY
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add docs/release-checklist.md
git commit -m "docs: add release checklist"
```

## Final Verification

Run in this worktree:

```bash
go test ./... -v
```

Run in `tauri-app`:

```bash
npm ci
npm run build
```

Run on Windows CI or Windows dev host:

```powershell
cd tauri-app
npm ci
npm run tauri:build
cd ..
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-windows-release.ps1 `
  -ExePath .\tauri-app\src-tauri\target\release\agent-notify.exe
```

Run release workflow test by pushing a throwaway prerelease tag after branch is merged:

```bash
git tag v0.1.0-test.1
git push origin v0.1.0-test.1
```

Expected:
- GitHub Release exists for the tag.
- Release assets include at least one Tauri Windows installer/setup executable.
- Release assets include `agent-notify-server.exe` and `agent-notify-server-arm64.exe`.
- Downloaded `agent-notify.exe` or installed app opens UI.
- Local health endpoint works.
- LAN health endpoint works from Mac.
- Mac can send `/notify` and Windows receives a toast.

## Self-Review

- Scope coverage: fixes LAN sidecar bug, adds release workflow, adds executable smoke script, documents tag release.
- Placeholder scan: no unresolved placeholder markers remain.
- Type consistency: `control_addr()` and `sidecar_listen_addr()` are defined before tests use them; PowerShell script parameter is consistently `ExePath`.
