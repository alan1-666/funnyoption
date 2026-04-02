#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FUNNYOPTION_ENV_FILE:-${ROOT_DIR}/.env.local}"
SQL_FILE="${ROOT_DIR}/docs/sql/local_stale_freeze_audit.sql"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  source "${ENV_FILE}"
  set +a
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "missing required command: psql"
  exit 1
fi

if [[ ! -f "${SQL_FILE}" ]]; then
  echo "audit SQL not found: ${SQL_FILE}"
  exit 1
fi

: "${FUNNYOPTION_POSTGRES_DSN:?FUNNYOPTION_POSTGRES_DSN is required (set it directly or via .env.local)}"

psql "${FUNNYOPTION_POSTGRES_DSN}" -v ON_ERROR_STOP=1 -f "${SQL_FILE}"
