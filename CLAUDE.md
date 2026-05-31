# Agent Notes

## DMG Packaging Worktree

- Use `/Users/gaoguobin/project/agent-notification/.worktrees/package-dmg-20260531` for local macOS DMG packaging experiments.
- Keep the repository root worktree on `main` for sync/inspection only. Do not create build artifacts in the root worktree.
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
