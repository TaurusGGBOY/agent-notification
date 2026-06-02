# Agent Notes

## Development Workflow

- Keep normal code and documentation work in a feature worktree, not the repository root worktree.
- Run the relevant automated tests before committing.
- Before committing, decide whether the change can be verified with a screenshot. UI changes, notification behavior, installer/client startup behavior, and other visible workflows should be screenshot-verified when a desktop session is available.
- If screenshot verification is possible, capture and inspect the screenshot before committing. Do not commit until the screenshot confirms the visible behavior is correct.
- Review the exact staged or to-be-committed diff before committing. Fix any issues found during this review first.
- Push the feature branch and open or update the PR only after tests, screenshot verification when applicable, and pre-commit review pass.
- After opening or updating the PR, review the PR diff and CI/check results again. Apply fixes in the same feature branch and repeat until there are no blocking issues.

## Local macOS DMG and Toast Verification

- Use `/Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531` for local macOS DMG packaging experiments.
- Keep the repository root worktree on `main` for sync/inspection only. Do not create build artifacts in the root worktree.
- For code fixes, run verification from the active feature worktree for that task. Treat the paths below as templates and replace the worktree path with the active feature worktree path before running commands.
- Before every local macOS app/toast verification, force-stop any existing `AgentNotify` client and `agent-notify-server`, then confirm port `17891` is free before launching the build under test:

```bash
pkill -f '/AgentNotify.app/Contents/MacOS/agent-notify-server' 2>/dev/null || true
pkill -f '/AgentNotify.app/Contents/MacOS/agent-notify' 2>/dev/null || true
pkill -f 'agent-notify-server' 2>/dev/null || true
pkill -f 'AgentNotify' 2>/dev/null || true
sleep 2
ps -axo pid,ppid,comm,args | rg -i 'AgentNotify|agent-notify' || true
lsof -nP -iTCP:17891 -sTCP:LISTEN || true
```

- For local toast verification against a Tauri-built `.app`, ad-hoc sign the app bundle before launch. An unsigned or partially signed bundle can make `UNUserNotificationCenter` fail authorization even when macOS notification settings are enabled:

```bash
APP="/Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531/tauri-app/src-tauri/target/release/bundle/macos/AgentNotify.app"
codesign --force --deep --sign - "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"
```

- Before packaging, update the packaging worktree to latest `origin/main`:

```bash
git -C /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531 fetch origin
git -C /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531 pull --ff-only
```

- Build from `tauri-app` inside that worktree:

```bash
cd /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531/tauri-app
npm ci
npm run tauri:build
```

- `npm run tauri:build` runs `npm run prepare:sidecar` first. That compiles the Go sidecar and may modify `windows-server/agent-notify-server-build`; restore that tracked build artifact before committing anything:

```bash
git -C /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531 restore windows-server/agent-notify-server-build
rm -f /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531/windows-server/agent-notify-server
```

- Expected Tauri output paths:
  - App bundle: `tauri-app/src-tauri/target/release/bundle/macos/AgentNotify.app`
  - DMG: `tauri-app/src-tauri/target/release/bundle/dmg/AgentNotify_1.0.1_aarch64.dmg`

- If Tauri's generated `bundle_dmg.sh` fails after producing `AgentNotify.app`, create a simple local trial DMG manually:

```bash
cd /Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531/tauri-app
codesign --force --deep --sign - src-tauri/target/release/bundle/macos/AgentNotify.app
codesign --verify --deep --strict --verbose=2 src-tauri/target/release/bundle/macos/AgentNotify.app

OUT="$PWD/src-tauri/target/release/bundle/dmg/AgentNotify_1.0.1_aarch64.dmg"
STAGE="$PWD/src-tauri/target/release/bundle/dmg/manual-stage"
rm -rf "$STAGE" "$OUT"
mkdir -p "$STAGE"
cp -R "$PWD/src-tauri/target/release/bundle/macos/AgentNotify.app" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
hdiutil create -volname "AgentNotify" -srcfolder "$STAGE" -ov -format UDZO "$OUT"
rm -rf "$STAGE"
hdiutil verify "$OUT"
cp -f "$OUT" /Users/gaoguobin/Desktop/AgentNotify_1.0.1_aarch64.dmg
```

- Local DMGs are ad-hoc signed and not notarized. They are for local trial only; first launch may require right-clicking the app and choosing Open.
