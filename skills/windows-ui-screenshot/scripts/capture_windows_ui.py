#!/usr/bin/env python3
"""Capture a Windows desktop UI screenshot through an interactive task."""

import argparse
import base64
import os
import subprocess
import tempfile
import textwrap
import time


TASK_NAME = "AgentNotifyUIVerify"
REMOTE_TEMP = r"C:\Users\Administrator\AppData\Local\Temp"


def run(args, check=False):
    result = subprocess.run(args, capture_output=True, text=True, errors="replace")
    if check and result.returncode != 0:
        raise RuntimeError(
            f"command failed: {' '.join(args)}\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def ssh(host, command, check=False):
    return run(["ssh", host, command], check=check)


def ps(host, command, check=False):
    encoded = base64.b64encode(command.encode("utf-16le")).decode("ascii")
    return ssh(host, f"powershell -NoProfile -EncodedCommand {encoded}", check=check)


def write_capture_script(host, exe_path, process_name):
    remote_script = rf"{REMOTE_TEMP}\agentnotify-ui-capture.ps1"
    remote_png = rf"{REMOTE_TEMP}\agentnotify-ui-capture.png"
    remote_info = rf"{REMOTE_TEMP}\agentnotify-ui-capture.info"
    remote_done = rf"{REMOTE_TEMP}\agentnotify-ui-capture.done"

    script = textwrap.dedent(
        rf"""
        $ErrorActionPreference = "Stop"
        $exe = "{exe_path}"
        $processName = "{process_name}"
        $png = "{remote_png}"
        $info = "{remote_info}"
        $done = "{remote_done}"

        Remove-Item $png, $info, $done -Force -ErrorAction SilentlyContinue
        Get-Process $processName -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

        Add-Type @"
        using System;
        using System.Runtime.InteropServices;
        public class NativeWin {{
          [DllImport("user32.dll")]
          public static extern bool SetForegroundWindow(IntPtr hWnd);
          [DllImport("user32.dll")]
          public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
        }}
"@

        $proc = Start-Process -FilePath $exe -PassThru
        $hwnd = 0
        for ($i = 0; $i -lt 20; $i++) {{
          Start-Sleep -Milliseconds 500
          $p = Get-Process $processName -ErrorAction SilentlyContinue | Where-Object {{ $_.MainWindowHandle -ne 0 }} | Select-Object -First 1
          if ($null -ne $p) {{
            $hwnd = $p.MainWindowHandle
            [NativeWin]::ShowWindow($hwnd, 9) | Out-Null
            [NativeWin]::SetForegroundWindow($hwnd) | Out-Null
            break
          }}
        }}

        Start-Sleep -Seconds 5
        Add-Type -AssemblyName System.Windows.Forms
        Add-Type -AssemblyName System.Drawing
        $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
        $bitmap = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height)
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        try {{
          $graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
          $bitmap.Save($png, [System.Drawing.Imaging.ImageFormat]::Png)
          "hwnd=$hwnd width=$($bounds.Width) height=$($bounds.Height)" | Set-Content -Encoding UTF8 $info
        }} finally {{
          $graphics.Dispose()
          $bitmap.Dispose()
          "done" | Set-Content -Encoding UTF8 $done
        }}
        """
    ).strip()

    with tempfile.NamedTemporaryFile("w", suffix=".ps1", delete=False, encoding="utf-8") as f:
        f.write(script)
        local_script = f.name
    try:
        run(["scp", local_script, f"{host}:{remote_script}"], check=True)
    finally:
        os.unlink(local_script)
    return remote_script, remote_png, remote_info, remote_done


def scp_path(windows_path):
    if windows_path.startswith(r"C:\Users\Administrator"):
        suffix = windows_path[len(r"C:\Users\Administrator") :].replace("\\", "/")
        return f"/Users/Administrator{suffix}"
    return windows_path.replace("\\", "/")


def schedule_and_run(host, remote_script):
    ps(host, f'schtasks /Delete /TN {TASK_NAME} /F 2>$null', check=False)
    create = textwrap.dedent(
        rf"""
        $trigger = 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "{remote_script}"'
        & schtasks /Create /TN {TASK_NAME} /TR $trigger /SC ONCE /ST 23:59 /RL HIGHEST /IT /F
        exit $LASTEXITCODE
        """
    )
    ps(host, create, check=True)
    ps(host, f'schtasks /Run /TN {TASK_NAME}', check=True)


def wait_for_done(host, remote_done, timeout=30):
    for _ in range(timeout):
        result = ps(host, f'Test-Path "{remote_done}"')
        if "True" in result.stdout:
            return True
        time.sleep(1)
    return False


def main():
    parser = argparse.ArgumentParser(description="Capture Windows UI screenshot")
    parser.add_argument("--host", required=True, help="SSH destination")
    parser.add_argument("--exe", required=True, help="Path to .exe on Windows")
    parser.add_argument("--process", required=True, help="Process name without .exe")
    parser.add_argument("--out", required=True, help="Local output path")
    args = parser.parse_args()

    remote_script, remote_png, remote_info, remote_done = write_capture_script(
        args.host, args.exe, args.process
    )
    schedule_and_run(args.host, remote_script)
    if not wait_for_done(args.host, remote_done):
        raise SystemExit("ERROR: timed out waiting for interactive screenshot task")

    os.makedirs(os.path.dirname(args.out) or ".", exist_ok=True)
    run(["scp", f"{args.host}:{scp_path(remote_png)}", args.out], check=True)
    info = ps(args.host, f'Get-Content "{remote_info}" -Raw')
    print(f"screenshot={args.out}")
    print(info.stdout.strip())

    ps(
        args.host,
        f'Remove-Item "{remote_script}","{remote_png}","{remote_info}","{remote_done}" -Force -ErrorAction SilentlyContinue; schtasks /Delete /TN {TASK_NAME} /F 2>$null',
        check=False,
    )


if __name__ == "__main__":
    main()
