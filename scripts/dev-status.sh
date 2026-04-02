#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="${ROOT_DIR}/.run/dev"
LOG_DIR="${ROOT_DIR}/.logs/dev"
MANAGED_KAFKA_FILE="${RUN_DIR}/managed-kafka"

if [[ ! -d "${RUN_DIR}" ]]; then
  echo "no pid directory yet: ${RUN_DIR}"
  exit 0
fi

for pid_file in "${RUN_DIR}"/*.pid; do
  [[ -e "${pid_file}" ]] || continue
  name="$(basename "${pid_file}" .pid)"
  pid="$(cat "${pid_file}" 2>/dev/null || true)"
  log_file="${LOG_DIR}/${name}.log"

  if [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1; then
    echo "[up]   ${name} pid=${pid} log=${log_file}"
  else
    echo "[down] ${name} pid=${pid:-unknown} log=${log_file}"
  fi
done

if [[ -f "${MANAGED_KAFKA_FILE}" ]]; then
  echo "[info] kafka managed mode=$(cat "${MANAGED_KAFKA_FILE}")"
fi
