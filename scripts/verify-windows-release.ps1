param(
  [Parameter(Mandatory = $true)]
  [string]$ExePath,
  [int]$TimeoutSeconds = 45,
  [string]$ScreenshotPath = ""
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public static class AgentNotifyNativeWindow {
  [StructLayout(LayoutKind.Sequential)]
  public struct RECT {
    public int Left;
    public int Top;
    public int Right;
    public int Bottom;
  }

  [DllImport("user32.dll")]
  public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);

  [DllImport("user32.dll")]
  public static extern bool SetForegroundWindow(IntPtr hWnd);

  [DllImport("user32.dll")]
  public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@

function Stop-AgentNotifyProcesses {
  param(
    [System.Diagnostics.Process]$AppProcess,
    [int[]]$ServerProcessIds = @()
  )

  if ($AppProcess -and -not $AppProcess.HasExited) {
    Stop-Process -Id $AppProcess.Id -Force -ErrorAction SilentlyContinue
  }

  foreach ($serverProcessId in $ServerProcessIds) {
    Stop-Process -Id $serverProcessId -Force -ErrorAction SilentlyContinue
  }
}

function Get-AgentNotifyWindow {
  param(
    [int]$ProcessId
  )

  $process = Get-Process -Id $ProcessId -ErrorAction Stop
  if ($process.MainWindowHandle -ne 0) {
    return $process
  }
  return $null
}

function Get-AgentNotifyListener {
  if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) {
    return $null
  }

  Get-NetTCPConnection -LocalPort 17891 -State Listen -ErrorAction SilentlyContinue |
    Select-Object -First 1
}

function Get-AgentNotifyServerProcessInfo {
  param(
    [int]$ProcessId = 0
  )

  $processes = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -like "agent-notify-server*.exe" }
  if ($ProcessId -gt 0) {
    return $processes | Where-Object { [int]$_.ProcessId -eq $ProcessId }
  }
  return $processes
}

function Test-LooksRendered {
  param(
    [System.Drawing.Bitmap]$Bitmap
  )

  $colors = @{}
  $minBrightness = 255
  $maxBrightness = 0
  $left = [Math]::Max([int]($Bitmap.Width * 0.04), 1)
  $right = [Math]::Min([int]($Bitmap.Width * 0.96), $Bitmap.Width - 1)
  $top = [Math]::Min([Math]::Max([int]($Bitmap.Height * 0.12), 72), $Bitmap.Height - 2)
  $bottom = [Math]::Min([int]($Bitmap.Height * 0.96), $Bitmap.Height - 1)
  $stepX = [Math]::Max([int](($right - $left) / 24), 1)
  $stepY = [Math]::Max([int](($bottom - $top) / 24), 1)

  for ($x = $left; $x -lt $right; $x += $stepX) {
    for ($y = $top; $y -lt $bottom; $y += $stepY) {
      $pixel = $Bitmap.GetPixel($x, $y)
      $colors[$pixel.ToArgb()] = $true
      $brightness = [int](($pixel.R + $pixel.G + $pixel.B) / 3)
      if ($brightness -lt $minBrightness) { $minBrightness = $brightness }
      if ($brightness -gt $maxBrightness) { $maxBrightness = $brightness }
    }
  }

  return ($colors.Count -ge 8 -and ($maxBrightness - $minBrightness) -ge 20)
}

function Save-AgentNotifyScreenshot {
  param(
    [System.Diagnostics.Process]$WindowProcess,
    [string]$Path
  )

  $hwnd = [IntPtr]$WindowProcess.MainWindowHandle
  [AgentNotifyNativeWindow]::ShowWindow($hwnd, 5) | Out-Null
  [AgentNotifyNativeWindow]::SetForegroundWindow($hwnd) | Out-Null
  Start-Sleep -Milliseconds 750

  $rect = New-Object AgentNotifyNativeWindow+RECT
  if (-not [AgentNotifyNativeWindow]::GetWindowRect($hwnd, [ref]$rect)) {
    throw "Could not read AgentNotify window bounds"
  }

  $width = $rect.Right - $rect.Left
  $height = $rect.Bottom - $rect.Top
  if ($width -lt 320 -or $height -lt 240) {
    throw "AgentNotify window is too small to be usable: ${width}x${height}"
  }

  $bitmap = New-Object System.Drawing.Bitmap $width, $height
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  try {
    $graphics.CopyFromScreen($rect.Left, $rect.Top, 0, 0, $bitmap.Size)
    $bitmap.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
    if (-not (Test-LooksRendered -Bitmap $bitmap)) {
      throw "AgentNotify window capture appears blank or unrendered: $Path"
    }
  }
  finally {
    $graphics.Dispose()
    $bitmap.Dispose()
  }
}

if ($TimeoutSeconds -lt 5) {
  throw "TimeoutSeconds must be at least 5"
}

if (-not (Test-Path -LiteralPath $ExePath)) {
  throw "Executable not found: $ExePath"
}

$resolvedExe = (Resolve-Path -LiteralPath $ExePath).Path
$screenshotResolvedPath = $ScreenshotPath
if ([string]::IsNullOrWhiteSpace($screenshotResolvedPath)) {
  $screenshotResolvedPath = Join-Path $env:TEMP "agentnotify-smoke-window.png"
}
$screenshotDir = Split-Path -Parent $screenshotResolvedPath
if ($screenshotDir) {
  New-Item -ItemType Directory -Force $screenshotDir | Out-Null
}

$existing = @(
  Get-Process -Name "agent-notify" -ErrorAction SilentlyContinue
  Get-Process -Name "AgentNotify" -ErrorAction SilentlyContinue
  Get-Process -Name "agent-notify-server*" -ErrorAction SilentlyContinue
)
if ($existing.Count -gt 0) {
  $names = ($existing | ForEach-Object { "$($_.ProcessName):$($_.Id)" }) -join ", "
  throw "AgentNotify is already running; stop existing processes before smoke test: $names"
}

$process = $null
$serverProcessIds = @()
try {
  $process = Start-Process -FilePath $resolvedExe -WorkingDirectory (Split-Path -Parent $resolvedExe) -PassThru
  try {
    $null = $process.WaitForInputIdle([Math]::Min($TimeoutSeconds * 1000, 15000))
  } catch {
    # Some GUI process types do not support WaitForInputIdle; polling below is authoritative.
  }

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  $health = $null
  $manifest = $null
  $window = $null
  $listener = $null

  while ((Get-Date) -lt $deadline) {
    if ($process.HasExited) {
      throw "AgentNotify process exited early with code $($process.ExitCode)"
    }

    if (-not $window) {
      $window = Get-AgentNotifyWindow -ProcessId $process.Id
    }

    if (-not $health) {
      try {
        $health = Invoke-RestMethod -Uri "http://127.0.0.1:17891/health" -TimeoutSec 2
        if ($health.status -ne "ok") {
          $health = $null
        }
      } catch {
        $health = $null
      }
    }

    if ($health -and -not $manifest) {
      try {
        $manifest = Invoke-RestMethod -Uri "http://127.0.0.1:17891/manifest" -TimeoutSec 2
      } catch {
        $manifest = $null
      }
    }

    if (-not $listener) {
      $candidateListener = Get-AgentNotifyListener
      if ($candidateListener) {
        $candidateProcessInfo = Get-AgentNotifyServerProcessInfo -ProcessId ([int]$candidateListener.OwningProcess) | Select-Object -First 1
        if ($candidateProcessInfo) {
          $listener = $candidateListener
          $serverProcessIds = @([int]$candidateListener.OwningProcess)
        }
      }
    }

    if ($window -and $health -and $manifest -and $listener) {
      break
    }

    Start-Sleep -Milliseconds 500
  }

  if (-not $window) {
    throw "AgentNotify process started but no main window handle was detected"
  }
  Save-AgentNotifyScreenshot -WindowProcess $window -Path $screenshotResolvedPath

  if (-not $health -or $health.status -ne "ok") {
    throw "AgentNotify server did not become healthy"
  }

  if (-not $manifest -or -not $manifest.url) {
    throw "AgentNotify manifest did not include a url"
  }

  $manifestUri = $null
  if (-not [Uri]::TryCreate($manifest.url, [UriKind]::Absolute, [ref]$manifestUri)) {
    throw "Manifest URL is invalid: $($manifest.url)"
  }
  if ($manifestUri.Scheme -ne "http" -or $manifestUri.Port -ne 17891) {
    throw "Manifest URL must be http on port 17891, got: $($manifest.url)"
  }
  if ($manifestUri.Host -match "^(127\.|0\.0\.0\.0$|localhost$|::1$)") {
    throw "Manifest URL must be LAN reachable, got: $($manifest.url)"
  }

  $lanHealthUri = "$($manifest.url.TrimEnd('/'))/health"
  $lanHealth = Invoke-RestMethod -Uri $lanHealthUri -TimeoutSec 5
  if (-not $lanHealth -or $lanHealth.status -ne "ok") {
    throw "Manifest LAN URL did not return healthy status: $lanHealthUri"
  }

  if (-not $listener) {
    throw "No TCP listener on port 17891 was detected"
  }
  if ($listener.LocalAddress -eq "127.0.0.1") {
    throw "Server is listening only on loopback: $($listener.LocalAddress):$($listener.LocalPort)"
  }

  $serverProcessInfo = @(Get-AgentNotifyServerProcessInfo -ProcessId ([int]$listener.OwningProcess))
  $serverProcessIds = @($serverProcessInfo | ForEach-Object { [int]$_.ProcessId })
  $serverProcesses = @()
  if ($serverProcessIds.Count -gt 0) {
    $serverProcesses = @(Get-Process -Id $serverProcessIds -ErrorAction SilentlyContinue)
  }
  if (-not $serverProcesses) {
    throw "No bundled agent-notify-server sidecar process was detected"
  }
  if ($serverProcessIds -notcontains [int]$listener.OwningProcess) {
    throw "Port 17891 listener pid=$($listener.OwningProcess) is not an agent-notify-server sidecar process"
  }

  $listenerProcessInfo = $serverProcessInfo | Where-Object { [int]$_.ProcessId -eq [int]$listener.OwningProcess } | Select-Object -First 1
  if (-not $listenerProcessInfo) {
    throw "Could not resolve listener process metadata for pid=$($listener.OwningProcess)"
  }
  if ([int]$listenerProcessInfo.ParentProcessId -ne [int]$process.Id) {
    throw "Server listener is not a child of AgentNotify app: pid=$($listener.OwningProcess), parent=$($listenerProcessInfo.ParentProcessId), app=$($process.Id)"
  }

  Write-Host "AgentNotify UI process: pid=$($process.Id), windowHandle=$($window.MainWindowHandle)"
  Write-Host "AgentNotify UI screenshot: $screenshotResolvedPath"
  Write-Host "AgentNotify server processes: $(($serverProcesses | ForEach-Object { "$($_.ProcessName):$($_.Id)" }) -join ', ')"
  Write-Host "Server listener: $($listener.LocalAddress):$($listener.LocalPort), pid=$($listener.OwningProcess)"
  Write-Host "Server health: $($health.status)"
  Write-Host "Manifest URL: $($manifest.url)"
  Write-Host "Manifest LAN health: $($lanHealth.status)"
}
finally {
  Stop-AgentNotifyProcesses -AppProcess $process -ServerProcessIds $serverProcessIds
}
