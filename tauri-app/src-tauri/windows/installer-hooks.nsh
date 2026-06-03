!macro AGENTNOTIFY_STOP_RUNNING_PROCESSES
  DetailPrint "Stopping running AgentNotify processes before modifying installed files..."
  ExecWait 'taskkill /F /T /IM AgentNotify.exe' $0
  ExecWait 'taskkill /F /T /IM agent-notify-server*.exe' $0
!macroend

!macro NSIS_HOOK_PREINSTALL
  !insertmacro AGENTNOTIFY_STOP_RUNNING_PROCESSES
!macroend

!macro NSIS_HOOK_PREUNINSTALL
  !insertmacro AGENTNOTIFY_STOP_RUNNING_PROCESSES
!macroend
