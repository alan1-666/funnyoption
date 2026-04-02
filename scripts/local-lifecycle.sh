#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "${ROOT_DIR}/.env.local" ]]; then
  set -a
  source "${ROOT_DIR}/.env.local"
  set +a
fi

if [[ -f "${ROOT_DIR}/.run/dev/local-chain.env" ]]; then
  set -a
  source "${ROOT_DIR}/.run/dev/local-chain.env"
  set +a
fi

cd "${ROOT_DIR}"
exec go run ./cmd/local-lifecycle "$@"
