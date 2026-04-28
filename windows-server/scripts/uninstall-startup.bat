@echo off
setlocal enabledelayedexpansion

set "TASK_NAME=AgentNotifyServer"
set "LOG_FILE=%TEMP%\AgentNotifyServerInstall.log"

echo Uninstalling Agent Notify Server startup task...

schtasks /delete /tn "%TASK_NAME%" /f
if !errorlevel! equ 0 (
    echo Task '%TASK_NAME%' removed successfully
    echo [%DATE% %TIME%] Task uninstalled >> "%LOG_FILE%"
    exit /b 0
) else (
    echo Error: Task not found or already removed
    exit /b 1
)
