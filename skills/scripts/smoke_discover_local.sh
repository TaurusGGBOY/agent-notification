#!/usr/bin/env bash
set -euo pipefail

AGENT_NOTIFICATION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SKILL_DIR="$AGENT_NOTIFICATION_DIR/../agent-notify-discovery-skill"

echo "=== Smoke Test: Local mDNS Discovery ==="
echo "Starting local server..."

# Start server in background
cd "$AGENT_NOTIFICATION_DIR/windows-server"
go run . &
SERVER_PID=$!

# Wait for server to be ready
echo "Waiting for server..."
for i in {1..30}; do
    if curl -s http://localhost:17891/health > /dev/null 2>&1; then
        echo "Server ready"
        break
    fi
    sleep 0.5
done

# Give mDNS time to advertise
sleep 2

echo ""
echo "Running discovery..."
RESULT=$(~/.claude/skills/agent-notify-discovery/.venv/bin/python ~/.claude/skills/agent-notify-discovery/scripts/discover.py --timeout 5 --json 2>&1)
echo "$RESULT"

echo ""
echo "Checking results..."
COUNT=$(echo "$RESULT" | ~/.claude/skills/agent-notify-discovery/.venv/bin/python -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")

kill $SERVER_PID 2>/dev/null || true

if [ "$COUNT" -gt 0 ]; then
    echo "✓ PASS: Discovered $COUNT service(s)"
    exit 0
else
    echo "✗ FAIL: No services discovered"
    exit 1
fi
