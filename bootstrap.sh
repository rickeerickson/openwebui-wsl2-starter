#!/bin/bash
#
# Bootstrap the owui Python package and run setup.
#
# Ensures python3, creates a venv, installs owui in editable
# mode, then hands off to the owui CLI.
#
# Usage:
#   ./bootstrap.sh          # Run owui setup
#   ./bootstrap.sh --help   # Pass flags to owui setup

set -euo pipefail
trap 'echo "ERROR: ${BASH_SOURCE[0]}:${LINENO}: ${BASH_COMMAND}" >&2' ERR

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${SCRIPT_DIR}/.venv"

# Ensure python3 is available
if ! command -v python3 &>/dev/null; then
    echo "ERROR: python3 not found. Install Python 3.11+ first." >&2
    exit 1
fi

# Create venv if needed
if [[ ! -d "${VENV_DIR}" ]]; then
    echo "Creating Python venv at ${VENV_DIR}..."
    python3 -m venv "${VENV_DIR}"
fi

# Activate and install
# shellcheck source=/dev/null
source "${VENV_DIR}/bin/activate"
pip install --quiet --upgrade pip
pip install --quiet -e "${SCRIPT_DIR}"

# Hand off to owui CLI
owui setup "$@"
