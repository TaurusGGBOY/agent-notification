$ErrorActionPreference = "Stop"

$src = Split-Path -Parent $MyInvocation.MyCommand.Path
$dest = Join-Path $HOME ".claude\skills\agent-notify-discovery"

New-Item -ItemType Directory -Force (Join-Path $HOME ".claude\skills") | Out-Null
if (Test-Path $dest) {
  Remove-Item -Recurse -Force $dest
}
Copy-Item -Recurse $src $dest

py -3 -m venv (Join-Path $dest ".venv")
& (Join-Path $dest ".venv\Scripts\python.exe") -m pip install --upgrade pip
& (Join-Path $dest ".venv\Scripts\python.exe") -m pip install zeroconf

Write-Host "Installed agent-notify-discovery skill at $dest"