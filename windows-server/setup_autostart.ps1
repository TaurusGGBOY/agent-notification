# Auto-start setup for Agent Notify Server
$ErrorActionPreference = "Stop"

$exePath = "D:\project\agent-notification\windows-server\agent-notify-server.exe"
$workDir = "D:\project\agent-notification\windows-server"
$taskName = "AgentNotifyServer"

Write-Host "Setting up Agent Notify Server auto-start..."

# Check if already exists
$existing = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "Task already exists. Removing old task..."
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
}

# Create action
$action = New-ScheduledTaskAction -Execute $exePath -WorkingDirectory $workDir

# Create trigger (at startup)
$trigger = New-ScheduledTaskTrigger -AtStartup

# Create settings
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

# Register task
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Description "Agent Notify Server for Windows notifications"

Write-Host "Done! Agent Notify Server will auto-start on boot."
Write-Host ""
Write-Host "Current status:"
Get-ScheduledTask -TaskName $taskName | Select-Object TaskName, State