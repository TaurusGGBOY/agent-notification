@echo off
setlocal enabledelayedexpansion

set "SERVICE_NAME=AgentNotifyServer"
set "LOG_FILE=%TEMP%\AgentNotifyServer.log"

echo Stopping Agent Notify Server...
call :StopProcess

timeout /t 2 /nobreak >nul

echo Starting Agent Notify Server...
call "%~dp0start.bat"

echo Server restarted at %DATE% %TIME% >> "%LOG_FILE%"
exit /b 0

:StopProcess
tasklist /FI "IMAGENAME eq agent-notify-server.exe" /FO CSV /NH | findstr /i "agent-notify-server.exe" >nul
if !errorlevel! equ 0 (
    taskkill /F /IM agent-notify-server.exe >nul 2>&1
    echo Stopped existing instance
)
exit /b 0
