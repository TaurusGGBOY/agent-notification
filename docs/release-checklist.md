# Release Checklist

1. Merge the release-ready branch to `main`.
2. Create and push a version tag:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. Open GitHub Actions and wait for the `Release` workflow to pass.
4. If you need to rerun an existing tag, run the `Release` workflow manually with the same tag input.
5. Open the generated GitHub Release.
6. Confirm release assets include:
   - `AgentNotify_<version>_x64-setup.exe`
   - `AgentNotify_<version>_x64_en-US.msi`
   - `agent-notify-server.exe`
   - `agent-notify-server-arm64.exe`
7. Download the Windows installer or setup executable.
8. Install on Windows.
9. Open AgentNotify from the Start Menu or desktop.
10. Confirm UI appears.
11. Confirm `http://127.0.0.1:17891/health` returns `{"status":"ok"}`.
12. Confirm `http://<windows-lan-ip>:17891/health` works from Mac.
13. Send a test notification from Mac:

    ```bash
    curl -fsS -X POST http://<windows-lan-ip>:17891/notify \
      -H 'Content-Type: application/json' \
      -d '{"agent":"codex","event":"stop","project":"release-smoke","message":"LAN smoke test","sourcePayload":{}}'
    ```

14. Confirm Windows shows the test notification.
15. For a built app exe smoke test on Windows, run:

    ```powershell
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-windows-release.ps1 `
      -ExePath .\tauri-app\src-tauri\target\release\agent-notify.exe
    ```
