# Release Checklist

1. Merge the release-ready branch to `main`.
2. Confirm GitHub Actions secrets include:
   - `TAURI_SIGNING_PRIVATE_KEY` from `tauri-app/tauri-updater-private.key`
   - `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if the updater key has a password
3. Confirm `tauri-app/src-tauri/tauri.conf.json` contains the matching `TAURI_UPDATER_PUBLIC_KEY` value in `plugins.updater.pubkey`.
4. Create and push a version tag:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

5. Open GitHub Actions and wait for the `Release` workflow to pass.
6. If you need to rerun an existing tag, run the `Release` workflow manually with the same tag input.
7. Open the generated GitHub Release.
8. Confirm release assets include:
   - `AgentNotify_<version>_x64-setup.exe`
   - `AgentNotify_<version>_x64-setup.exe.sig`
   - `AgentNotify_<version>_x64_en-US.msi`
   - `AgentNotify.app.tar.gz`
   - `AgentNotify.app.tar.gz.sig`
   - `latest.json`
   - `agent-notify-server.exe`
   - `agent-notify-server-arm64.exe`
9. Download the Windows installer or setup executable.
10. Install on Windows.
11. Open AgentNotify from the Start Menu or desktop.
12. Confirm UI appears.
13. Open the About popover and click `检查更新`.
14. Confirm the update check reports either the current version or a newer signed update.
15. Confirm `http://127.0.0.1:17891/health` returns `{"status":"ok"}`.
16. Confirm `http://<windows-lan-ip>:17891/health` works from Mac.
17. Send a test notification from Mac:

    ```bash
    curl -fsS -X POST http://<windows-lan-ip>:17891/notify \
      -H 'Content-Type: application/json' \
      -d '{"agent":"codex","event":"stop","project":"release-smoke","message":"LAN smoke test","sourcePayload":{}}'
    ```

18. Confirm Windows shows the test notification.
19. For a built app exe smoke test on Windows, run:

    ```powershell
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-windows-release.ps1 `
      -ExePath .\tauri-app\src-tauri\target\release\agent-notify.exe
    ```
