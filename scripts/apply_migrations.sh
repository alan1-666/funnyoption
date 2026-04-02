#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DSN="${FUNNYOPTION_POSTGRES_DSN:-}"

if [[ -z "${DSN}" ]]; then
  echo "FUNNYOPTION_POSTGRES_DSN is required"
  exit 1
fi

for migration in $(find "${ROOT_DIR}/migrations" -maxdepth 1 -type f -name '*.sql' | sort); do
  echo "applying ${migration}"
  psql "${DSN}" -v ON_ERROR_STOP=1 -f "${migration}"
done

echo "migrations applied successfully"
