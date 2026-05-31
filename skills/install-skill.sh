#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIR="${HOME}/.claude/skills/agent-notify-discovery"

mkdir -p "${HOME}/.claude/skills"
rm -rf "${DEST_DIR}"
mkdir -p "${DEST_DIR}"

# Copy skill content, excluding install scripts
cp -R "${SRC_DIR}/SKILL.md" "${DEST_DIR}/"
cp -R "${SRC_DIR}/references" "${DEST_DIR}/"
cp -R "${SRC_DIR}/scripts" "${DEST_DIR}/"
cp -R "${SRC_DIR}/tests" "${DEST_DIR}/"

python3 -m venv "${DEST_DIR}/.venv"
"${DEST_DIR}/.venv/bin/python" -m pip install --upgrade pip
"${DEST_DIR}/.venv/bin/python" -m pip install zeroconf

echo "Installed agent-notify-discovery skill at ${DEST_DIR}"