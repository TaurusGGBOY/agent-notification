@echo off
setlocal enabledelayedexpansion

set "SERVICE_NAME=AgentNotifyServer"
set "EXE_PATH=%~dp0agent-notify-server.exe"
set "LOG_FILE=%TEMP%\AgentNotifyServer.log"

echo Starting Agent Notify Server...
echo Starting Agent Notify Server at %DATE% %TIME% >> "%LOG_FILE%"

if not exist "%EXE_PATH%" (
    echo Error: agent-notify-server.exe not found at %EXE_PATH%
    exit /b 1
)

start "" "%EXE_PATH%"

echo Server started. PID: !errorlevel!
echo Server started at %DATE% %TIME% >> "%LOG_FILE%"
exit /b 0
