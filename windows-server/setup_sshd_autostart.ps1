# Auto-start setup for sshd service
$ErrorActionPreference = "Stop"

Write-Host "Setting up sshd service auto-start..."

# Check if OpenSSH service exists
$service = Get-Service -Name sshd -ErrorAction SilentlyContinue
if (-not $service) {
    Write-Host "sshd service not found. Make sure OpenSSH is installed."
    Write-Host "Download from: https://github.com/PowerShell/Win32-OpenSSH"
    exit 1
}

# Set startup type to Automatic
Set-Service -Name sshd -StartupType Automatic
Write-Host "Set sshd startup type to: Automatic"

# Start service if not running
if ($service.Status -ne "Running") {
    Start-Service sshd
    Write-Host "Started sshd service"
}

# Verify
$service = Get-Service -Name sshd
Write-Host ""
Write-Host "sshd service status:"
Write-Host "  Name: $($service.Name)"
Write-Host "  DisplayName: $($service.DisplayName)"
Write-Host "  Status: $($service.Status)"
Write-Host "  StartType: $($service.StartType)"
Write-Host ""
Write-Host "Done! sshd will auto-start on boot."