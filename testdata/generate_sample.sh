#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_FILE="${SCRIPT_DIR}/sample.jpg"
SRC_FILE="${SCRIPT_DIR}/sample.jpg.b64"

if [[ ! -f "${SRC_FILE}" ]]; then
  echo "missing source base64: ${SRC_FILE}" >&2
  exit 1
fi

base64 -d "${SRC_FILE}" > "${OUT_FILE}"
