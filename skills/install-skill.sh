#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIRS=(
  "${HOME}/.claude/skills/agent-notify-discovery"
  "${HOME}/.openclaw/skills/agent-notify-discovery"
)

for DEST_DIR in "${DEST_DIRS[@]}"; do
  mkdir -p "$(dirname "${DEST_DIR}")"
  rm -rf "${DEST_DIR}"
  mkdir -p "${DEST_DIR}"

  # Copy skill content, excluding install scripts
  cp -R "${SRC_DIR}/SKILL.md" "${DEST_DIR}/"
  cp -R "${SRC_DIR}/references" "${DEST_DIR}/"
  cp -R "${SRC_DIR}/scripts" "${DEST_DIR}/"
  cp -R "${SRC_DIR}/tests" "${DEST_DIR}/"

  echo "Installed agent-notify-discovery skill at ${DEST_DIR}"
done
