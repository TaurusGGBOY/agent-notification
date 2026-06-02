$ErrorActionPreference = "Stop"

$src = Split-Path -Parent $MyInvocation.MyCommand.Path
$dests = @(
  (Join-Path $HOME ".claude\skills\agent-notify-discovery"),
  (Join-Path $HOME ".openclaw\skills\agent-notify-discovery")
)

foreach ($dest in $dests) {
  New-Item -ItemType Directory -Force (Split-Path -Parent $dest) | Out-Null
  if (Test-Path $dest) {
    Remove-Item -Recurse -Force $dest
  }
  Copy-Item -Recurse $src $dest

  Write-Host "Installed agent-notify-discovery skill at $dest"
}
