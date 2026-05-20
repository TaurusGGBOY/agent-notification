---
name: windows-ui-screenshot
description: Use when diagnosing or accepting a Windows desktop UI remotely over SSH, especially when a GUI must launch in the logged-in console session and screenshots must be pulled back automatically.
---

# Windows UI Screenshot

Automate Windows GUI acceptance from macOS/Linux without manual clicks.

## When To Use

- Need to verify a Windows desktop app actually opens.
- SSH can reach Windows, but direct `CopyFromScreen` from SSH fails.
- Need screenshot evidence for Tauri, Win32, tray, toast, or desktop UI debugging.

## Workflow

1. Confirm an interactive console session exists:

   ```bash
   ssh Administrator@192.168.31.167 "query user"
   ```

2. Run the bundled script:

   ```bash
   python3 skills/windows-ui-screenshot/scripts/capture_windows_ui.py \
     --host Administrator@192.168.31.167 \
     --exe 'D:\project\agent-notification\tauri-app\src-tauri\target\release\agent-notify.exe' \
     --process agent-notify \
     --out /tmp/agentnotify-ui.png
   ```

3. Inspect the returned PNG with `view_image`.

## Why Scheduled Task

`ssh powershell CopyFromScreen(...)` often fails with an invalid handle because SSH runs outside the interactive desktop. This skill creates a temporary Task Scheduler job with `/IT`, so capture runs in the logged-in console session.

## Notes

- Requires an active logged-in Windows user session.
- Script writes remote temp files under `C:\Users\Administrator\AppData\Local\Temp`.
- The task is named `AgentNotifyUIVerify` by default.
- If the screenshot shows UI but `Failed to fetch`, diagnose app/server networking separately.

