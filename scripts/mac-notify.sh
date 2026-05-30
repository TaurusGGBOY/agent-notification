#!/usr/bin/env bash
set -euo pipefail

title="${1:-Agent Notify}"
message="${2:-No message}"
sound="${3:-Glass}"

osascript - "$title" "$message" "$sound" <<'APPLESCRIPT'
on run argv
  display notification (item 2 of argv) with title (item 1 of argv) sound name (item 3 of argv)
end run
APPLESCRIPT
