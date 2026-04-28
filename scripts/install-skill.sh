#!/usr/bin/env bash
# Install agent-notify-discovery skill to ~/.claude/skills/

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILL_SOURCE="$REPO_ROOT/skills/agent-notify-discovery"
SKILL_TARGET="$HOME/.claude/skills/agent-notify-discovery"

if [ ! -d "$SKILL_SOURCE" ]; then
    echo "Error: skill source not found at $SKILL_SOURCE" >&2
    exit 1
fi

echo "Installing agent-notify-discovery skill..."

# Create ~/.claude/skills/ if not exists
mkdir -p "$HOME/.claude/skills"

# Remove existing if present
rm -rf "$SKILL_TARGET"

# Symlink (comment out below line and uncomment copy if you prefer copy)
ln -s "$SKILL_SOURCE" "$SKILL_TARGET"

echo "✓ Skill installed at $SKILL_TARGET"

# Optionally run discovery test
if [ "$1" == "--test" ]; then
    echo "Running discovery test..."
    python3 "$SKILL_TARGET/scripts/discover.py" --json
fi
