#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="${ROOT_DIR}/.run/dev"
KAFKA_COMPOSE_FILE="${ROOT_DIR}/deploy/kafka/docker-compose.yml"
MANAGED_KAFKA_FILE="${RUN_DIR}/managed-kafka"
SERVICE_PORTS=(8080 8081 9090 9091 9093 9094 9095 3000 3001)

if [[ ! -d "${RUN_DIR}" ]]; then
  echo "no running pid directory: ${RUN_DIR}"
  exit 0
fi

stop_pid_file() {
  local pid_file="$1"
  local name
  name="$(basename "${pid_file}" .pid)"
  local pid
  pid="$(cat "${pid_file}" 2>/dev/null || true)"
  if [[ -z "${pid}" ]]; then
    rm -f "${pid_file}"
    return 0
  fi

  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
    sleep 1
    if kill -0 "${pid}" >/dev/null 2>&1; then
      kill -9 "${pid}" >/dev/null 2>&1 || true
    fi
    echo "[stopped] ${name} (${pid})"
  else
    echo "[stale] ${name} (${pid})"
  fi

  rm -f "${pid_file}"
}

stop_port() {
  local port="$1"
  local pids
  pids="$(lsof -tiTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
  [[ -n "${pids}" ]] || return 0

  for pid in ${pids}; do
    kill "${pid}" >/dev/null 2>&1 || true
    sleep 1
    if kill -0 "${pid}" >/dev/null 2>&1; then
      kill -9 "${pid}" >/dev/null 2>&1 || true
    fi
    echo "[stopped] port ${port} pid ${pid}"
  done
}

for pid_file in "${RUN_DIR}"/*.pid; do
  [[ -e "${pid_file}" ]] || continue
  stop_pid_file "${pid_file}"
done

for port in "${SERVICE_PORTS[@]}"; do
  stop_port "${port}"
done

if [[ -f "${MANAGED_KAFKA_FILE}" ]]; then
  mode="$(cat "${MANAGED_KAFKA_FILE}" 2>/dev/null || true)"
  case "${mode}" in
    existing)
      docker stop kafka-broker zookeeper >/dev/null 2>&1 || true
      echo "[stopped] kafka-broker/zookeeper"
      ;;
    compose)
      docker compose -f "${KAFKA_COMPOSE_FILE}" down >/dev/null 2>&1 || true
      echo "[stopped] funnyoption local kafka"
      ;;
  esac
  rm -f "${MANAGED_KAFKA_FILE}"
fi

echo "funnyoption local dev stopped."
