#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/.venv"
ENV_PATH="$SCRIPT_DIR/.env"
PYTHON_BIN="${PYTHON_BIN:-python3}"

if [[ ! -f "$ENV_PATH" ]]; then
  echo "[CC98] Missing $ENV_PATH . Copy .env.example to .env first." >&2
  exit 1
fi

if ! command -v "$PYTHON_BIN" >/dev/null 2>&1; then
  echo "[CC98] Python not found: $PYTHON_BIN" >&2
  exit 1
fi

if [[ ! -d "$VENV_DIR" ]]; then
  "$PYTHON_BIN" -m venv "$VENV_DIR"
fi

# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"

if ! python -c "import requests; from Crypto.Cipher import AES" >/dev/null 2>&1; then
  python -m pip install -r "$SCRIPT_DIR/requirements.txt"
fi

exec python "$SCRIPT_DIR/webvpn-fixed-api.py" --env "$ENV_PATH" "$@"
