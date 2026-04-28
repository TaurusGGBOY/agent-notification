@echo off
setlocal enabledelayedexpansion

set "TASK_NAME=AgentNotifyServer"
set "EXE_PATH=%~dp0agent-notify-server.exe"
set "LOG_FILE=%TEMP%\AgentNotifyServerInstall.log"

echo Installing Agent Notify Server as startup task...

if not exist "%EXE_PATH%" (
    echo Error: agent-notify-server.exe not found at %EXE_PATH%
    exit /b 1
)

schtasks /create /tn "%TASK_NAME%" /tr "\"%EXE_PATH%\"" /sc onlogon /rl limited /f
if !errorlevel! equ 0 (
    echo Task '%TASK_NAME%' created successfully
    echo [%DATE% %TIME%] Task installed >> "%LOG_FILE%"
    echo.
    echo Agent Notify Server will start automatically at logon.
    exit /b 0
) else (
    echo Error: Failed to create scheduled task
    exit /b 1
)
