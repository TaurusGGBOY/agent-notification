#!/usr/bin/env bash
# Install agent-notify-discovery skill to Claude Code and OpenClaw skill paths.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILL_SOURCE="$REPO_ROOT/skills"
SKILL_TARGETS=(
    "$HOME/.claude/skills/agent-notify-discovery"
    "$HOME/.openclaw/skills/agent-notify-discovery"
)

if [ ! -d "$SKILL_SOURCE" ]; then
    echo "Error: skill source not found at $SKILL_SOURCE" >&2
    exit 1
fi

echo "Installing agent-notify-discovery skill..."

for SKILL_TARGET in "${SKILL_TARGETS[@]}"; do
    mkdir -p "$(dirname "$SKILL_TARGET")"
    rm -rf "$SKILL_TARGET"
    cp -R "$SKILL_SOURCE" "$SKILL_TARGET"
    python3 -m venv "$SKILL_TARGET/.venv"
    "$SKILL_TARGET/.venv/bin/python" -m pip install --upgrade pip
    "$SKILL_TARGET/.venv/bin/python" -m pip install zeroconf
    echo "✓ Skill installed at $SKILL_TARGET"
done

# Optionally run discovery test
if [ "${1:-}" == "--test" ]; then
    echo "Running discovery test..."
    for SKILL_TARGET in "${SKILL_TARGETS[@]}"; do
        "$SKILL_TARGET/.venv/bin/python" "$SKILL_TARGET/scripts/discover.py" --json
    done
fi
