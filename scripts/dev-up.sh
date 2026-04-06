#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="${ROOT_DIR}/web"
ADMIN_DIR="${ROOT_DIR}/admin"
ENV_FILE="${FUNNYOPTION_ENV_FILE:-${ROOT_DIR}/.env.local}"
RUN_DIR="${ROOT_DIR}/.run/dev"
LOG_DIR="${ROOT_DIR}/.logs/dev"
BIN_DIR="${RUN_DIR}/bin"
KAFKA_COMPOSE_FILE="${ROOT_DIR}/deploy/kafka/docker-compose.yml"
MANAGED_KAFKA_FILE="${RUN_DIR}/managed-kafka"
LOCAL_CHAIN_FRESH_FILE="${RUN_DIR}/local-chain-fresh-start"

mkdir -p "${RUN_DIR}" "${LOG_DIR}" "${BIN_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "env file not found: ${ENV_FILE}"
  exit 1
fi

set -a
source "${ENV_FILE}"
set +a

: "${FUNNYOPTION_LOCAL_CHAIN_MODE:=}"
: "${FUNNYOPTION_API_HTTP_ADDR:=:8080}"
: "${FUNNYOPTION_WS_HTTP_ADDR:=:8081}"
: "${FUNNYOPTION_MATCHING_GRPC_ADDR:=:9090}"
: "${FUNNYOPTION_ACCOUNT_GRPC_ADDR:=:9091}"
: "${FUNNYOPTION_LEDGER_GRPC_ADDR:=:9095}"
: "${FUNNYOPTION_SETTLEMENT_GRPC_ADDR:=:9093}"
: "${FUNNYOPTION_CHAIN_GRPC_ADDR:=:9094}"
: "${FUNNYOPTION_KAFKA_BROKERS:=127.0.0.1:9092}"
: "${FUNNYOPTION_POSTGRES_DSN:=postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable}"
: "${FUNNYOPTION_CHAIN_ID:=97}"
: "${FUNNYOPTION_FRONTEND_CHAIN_NAME:=BSC Testnet}"
: "${FUNNYOPTION_CHAIN_EXPLORER_URL:=https://testnet.bscscan.com}"
: "${FUNNYOPTION_CHAIN_RPC_URL:=https://data-seed-prebsc-1-s1.bnbchain.org:8545}"
: "${FUNNYOPTION_VAULT_ADDRESS:=}"
: "${FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS:=}"
: "${FUNNYOPTION_COLLATERAL_SYMBOL:=USDT}"
: "${FUNNYOPTION_COLLATERAL_DECIMALS:=6}"
: "${FUNNYOPTION_COLLATERAL_ACCOUNTING_DECIMALS:=2}"
: "${FUNNYOPTION_NATIVE_CURRENCY_NAME:=BNB}"
: "${FUNNYOPTION_NATIVE_CURRENCY_SYMBOL:=tBNB}"
: "${FUNNYOPTION_NATIVE_CURRENCY_DECIMALS:=18}"
: "${FUNNYOPTION_ADMIN_HTTP_ADDR:=:3001}"
: "${FUNNYOPTION_OPERATOR_WALLETS:=}"
: "${FUNNYOPTION_DEFAULT_OPERATOR_USER_ID:=1001}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1"
    exit 1
  fi
}

describe_port_listener() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true
}

port_is_listening() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
}

port_from_addr() {
  local value="$1"
  echo "${value##*:}"
}

http_url_from_addr() {
  local value="$1"
  local port
  port="$(port_from_addr "${value}")"
  echo "http://127.0.0.1:${port}"
}

is_running() {
  local pid_file="$1"
  [[ -f "${pid_file}" ]] || return 1
  local pid
  pid="$(cat "${pid_file}")"
  [[ -n "${pid}" ]] || return 1
  kill -0 "${pid}" >/dev/null 2>&1
}

ensure_service_ports_free() {
  local name="$1"
  shift
  local pid_file="${RUN_DIR}/${name}.pid"

  if is_running "${pid_file}"; then
    return 0
  fi

  local conflict=0
  local port
  for port in "$@"; do
    [[ -n "${port}" ]] || continue
    if port_is_listening "${port}"; then
      if (( conflict == 0 )); then
        echo "[fail] ${name} cannot start because a required port is already in use"
      fi
      conflict=1
      echo "port ${port} listener:"
      describe_port_listener "${port}"
    fi
  done

  if (( conflict != 0 )); then
    echo "tip: run ${ROOT_DIR}/scripts/dev-down.sh to stop tracked local services, or stop the conflicting listener manually."
    exit 1
  fi
}

wait_for_http() {
  local url="$1"
  local timeout="${2:-30}"
  local elapsed=0
  while (( elapsed < timeout )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  return 1
}

check_port() {
  local host="$1"
  local port="$2"
  nc -z "${host}" "${port}" >/dev/null 2>&1
}

wait_for_port() {
  local host="$1"
  local port="$2"
  local timeout="${3:-30}"
  local elapsed=0
  while (( elapsed < timeout )); do
    if check_port "${host}" "${port}"; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  return 1
}

docker_container_exists() {
  local name="$1"
  docker ps -a --format '{{.Names}}' | grep -qx "${name}"
}

ensure_kafka() {
  local broker="$1"
  local host="${broker%%:*}"
  local port="${broker##*:}"

  if check_port "${host}" "${port}"; then
    echo "kafka already reachable at ${broker}"
    rm -f "${MANAGED_KAFKA_FILE}"
    return 0
  fi

  if command -v docker >/dev/null 2>&1; then
    if docker_container_exists "zookeeper" && docker_container_exists "kafka-broker"; then
      echo "==> starting existing kafka containers (zookeeper, kafka-broker)"
      docker start zookeeper kafka-broker >/dev/null
      if wait_for_port "${host}" "${port}" 30; then
        echo "existing kafka containers are ready"
        echo "existing" >"${MANAGED_KAFKA_FILE}"
        return 0
      fi
    fi

    if [[ -f "${KAFKA_COMPOSE_FILE}" ]]; then
      echo "==> starting funnyoption local kafka"
      docker compose -f "${KAFKA_COMPOSE_FILE}" up -d >/dev/null
      if wait_for_port "${host}" "${port}" 45; then
        echo "compose" >"${MANAGED_KAFKA_FILE}"
        return 0
      fi
    fi
  fi

  echo "kafka is not reachable at ${broker}"
  return 1
}

write_web_env() {
  cat >"${WEB_DIR}/.env.local" <<EOF
NEXT_PUBLIC_API_BASE_URL=$(http_url_from_addr "${FUNNYOPTION_API_HTTP_ADDR}")
NEXT_PUBLIC_WS_BASE_URL=$(http_url_from_addr "${FUNNYOPTION_WS_HTTP_ADDR}")
NEXT_PUBLIC_DEFAULT_USER_ID=1001
NEXT_PUBLIC_ADMIN_BASE_URL=$(http_url_from_addr "${FUNNYOPTION_ADMIN_HTTP_ADDR}")

NEXT_PUBLIC_CHAIN_ID=${FUNNYOPTION_CHAIN_ID}
NEXT_PUBLIC_CHAIN_NAME=${FUNNYOPTION_FRONTEND_CHAIN_NAME}
NEXT_PUBLIC_VAULT_ADDRESS=${FUNNYOPTION_VAULT_ADDRESS}
NEXT_PUBLIC_COLLATERAL_TOKEN_ADDRESS=${FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS}
NEXT_PUBLIC_COLLATERAL_SYMBOL=${FUNNYOPTION_COLLATERAL_SYMBOL}
NEXT_PUBLIC_COLLATERAL_DECIMALS=${FUNNYOPTION_COLLATERAL_DECIMALS}
NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS=${FUNNYOPTION_COLLATERAL_ACCOUNTING_DECIMALS}
NEXT_PUBLIC_CHAIN_EXPLORER_URL=${FUNNYOPTION_CHAIN_EXPLORER_URL}
NEXT_PUBLIC_CHAIN_RPC_URL=${FUNNYOPTION_CHAIN_RPC_URL}
NEXT_PUBLIC_NATIVE_CURRENCY_NAME=${FUNNYOPTION_NATIVE_CURRENCY_NAME}
NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL=${FUNNYOPTION_NATIVE_CURRENCY_SYMBOL}
NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS=${FUNNYOPTION_NATIVE_CURRENCY_DECIMALS}
EOF
}

write_admin_env() {
  cat >"${ADMIN_DIR}/.env.local" <<EOF
NEXT_PUBLIC_API_BASE_URL=$(http_url_from_addr "${FUNNYOPTION_API_HTTP_ADDR}")
NEXT_PUBLIC_WS_BASE_URL=$(http_url_from_addr "${FUNNYOPTION_WS_HTTP_ADDR}")
NEXT_PUBLIC_DEFAULT_OPERATOR_USER_ID=${FUNNYOPTION_DEFAULT_OPERATOR_USER_ID}
NEXT_PUBLIC_OPERATOR_WALLETS=${FUNNYOPTION_OPERATOR_WALLETS}
NEXT_PUBLIC_PUBLIC_WEB_BASE_URL=http://127.0.0.1:3000
NEXT_PUBLIC_CHAIN_ID=${FUNNYOPTION_CHAIN_ID}
NEXT_PUBLIC_CHAIN_NAME=${FUNNYOPTION_FRONTEND_CHAIN_NAME}
NEXT_PUBLIC_CHAIN_RPC_URL=${FUNNYOPTION_CHAIN_RPC_URL}
NEXT_PUBLIC_CHAIN_EXPLORER_URL=${FUNNYOPTION_CHAIN_EXPLORER_URL}
NEXT_PUBLIC_COLLATERAL_SYMBOL=${FUNNYOPTION_COLLATERAL_SYMBOL}
NEXT_PUBLIC_COLLATERAL_ACCOUNTING_DECIMALS=${FUNNYOPTION_COLLATERAL_ACCOUNTING_DECIMALS}
NEXT_PUBLIC_NATIVE_CURRENCY_NAME=${FUNNYOPTION_NATIVE_CURRENCY_NAME}
NEXT_PUBLIC_NATIVE_CURRENCY_SYMBOL=${FUNNYOPTION_NATIVE_CURRENCY_SYMBOL}
NEXT_PUBLIC_NATIVE_CURRENCY_DECIMALS=${FUNNYOPTION_NATIVE_CURRENCY_DECIMALS}

FUNNYOPTION_DEFAULT_OPERATOR_USER_ID=${FUNNYOPTION_DEFAULT_OPERATOR_USER_ID}
FUNNYOPTION_OPERATOR_WALLETS=${FUNNYOPTION_OPERATOR_WALLETS}
EOF
}

start_service() {
  local name="$1"
  local cwd="$2"
  local command="$3"
  local pid_file="${RUN_DIR}/${name}.pid"
  local log_file="${LOG_DIR}/${name}.log"

  if is_running "${pid_file}"; then
    echo "[skip] ${name} already running (pid $(cat "${pid_file}"))"
    return 0
  fi

  rm -f "${pid_file}"

  (
    cd "${cwd}"
    nohup bash -lc "exec ${command}" >"${log_file}" 2>&1 &
    echo $! >"${pid_file}"
  )

  sleep 1
  if ! is_running "${pid_file}"; then
    echo "[fail] ${name} did not stay up. log: ${log_file}"
    tail -n 40 "${log_file}" || true
    rm -f "${pid_file}"
    exit 1
  fi

  echo "[ok] ${name} started (pid $(cat "${pid_file}"))"
}

build_go_service() {
  local name="$1"
  local pkg="$2"
  local output="${BIN_DIR}/${name}"
  (
    cd "${ROOT_DIR}"
    go build -o "${output}" "${pkg}"
  )
  echo "${output}"
}

start_go_service() {
  local name="$1"
  local pkg="$2"
  local binary
  binary="$(build_go_service "${name}" "${pkg}")"
  start_service "${name}" "${ROOT_DIR}" "${binary}"
}

require_command go
require_command npm
require_command curl
require_command psql
require_command nc
require_command lsof

reset_local_anvil_runtime_state() {
  [[ "${FUNNYOPTION_LOCAL_CHAIN_MODE}" == "anvil" ]] || return 0
  [[ -f "${LOCAL_CHAIN_FRESH_FILE}" ]] || return 0

  echo "==> resetting local anvil runtime state"
  psql "${FUNNYOPTION_POSTGRES_DSN}" <<SQL
DELETE FROM chain_listener_cursors
 WHERE chain_name = 'anvil'
   AND network_name = 'local'
   AND vault_address = '${FUNNYOPTION_VAULT_ADDRESS}';
DELETE FROM chain_transactions
 WHERE chain_name = 'anvil'
   AND network_name = 'local';
DELETE FROM chain_withdrawals
 WHERE chain_name = 'anvil'
   AND network_name = 'local';
DELETE FROM chain_deposits
 WHERE chain_name = 'anvil'
   AND network_name = 'local';
TRUNCATE rollup_accepted_withdrawals,
         rollup_accepted_payouts,
         rollup_accepted_positions,
         rollup_accepted_balances,
         rollup_accepted_batches,
         rollup_shadow_submissions,
         rollup_shadow_batches,
         rollup_shadow_journal_entries
RESTART IDENTITY;
SQL
  rm -f "${LOCAL_CHAIN_FRESH_FILE}"
}

if [[ "${FUNNYOPTION_LOCAL_CHAIN_MODE}" == "anvil" ]]; then
  require_command anvil
  require_command forge
  require_command cast
  echo "==> preparing persistent local anvil chain"
  FUNNYOPTION_ENV_FILE="${ENV_FILE}" "${ROOT_DIR}/scripts/local-chain-up.sh"
  if [[ ! -f "${RUN_DIR}/local-chain.env" ]]; then
    echo "local chain env was not generated"
    exit 1
  fi
  set -a
  source "${RUN_DIR}/local-chain.env"
  set +a
fi

echo "==> checking local postgres"
if ! psql "${FUNNYOPTION_POSTGRES_DSN}" -Atc "SELECT 1" >/dev/null; then
  echo "postgres is not reachable: ${FUNNYOPTION_POSTGRES_DSN}"
  exit 1
fi

reset_local_anvil_runtime_state

echo "==> checking kafka"
FIRST_BROKER="${FUNNYOPTION_KAFKA_BROKERS%%,*}"
KAFKA_HOST="${FIRST_BROKER%%:*}"
KAFKA_PORT="${FIRST_BROKER##*:}"
if ! ensure_kafka "${FIRST_BROKER}"; then
  exit 1
fi

echo "==> applying migrations"
FUNNYOPTION_POSTGRES_DSN="${FUNNYOPTION_POSTGRES_DSN}" "${ROOT_DIR}/scripts/apply_migrations.sh"

echo "==> writing frontend env"
write_web_env
write_admin_env

echo "==> checking service ports"
ensure_service_ports_free "account" "$(port_from_addr "${FUNNYOPTION_ACCOUNT_GRPC_ADDR}")"
ensure_service_ports_free "matching" "$(port_from_addr "${FUNNYOPTION_MATCHING_GRPC_ADDR}")"
ensure_service_ports_free "ledger" "$(port_from_addr "${FUNNYOPTION_LEDGER_GRPC_ADDR}")"
ensure_service_ports_free "settlement" "$(port_from_addr "${FUNNYOPTION_SETTLEMENT_GRPC_ADDR}")"
ensure_service_ports_free "chain" "$(port_from_addr "${FUNNYOPTION_CHAIN_GRPC_ADDR}")"
ensure_service_ports_free "api" "$(port_from_addr "${FUNNYOPTION_API_HTTP_ADDR}")"
ensure_service_ports_free "ws" "$(port_from_addr "${FUNNYOPTION_WS_HTTP_ADDR}")"
ensure_service_ports_free "web" "3000"
ensure_service_ports_free "admin" "$(port_from_addr "${FUNNYOPTION_ADMIN_HTTP_ADDR}")"

echo "==> starting backend services"
start_go_service "account" "./cmd/account"
start_go_service "matching" "./cmd/matching"
start_go_service "ledger" "./cmd/ledger"
start_go_service "settlement" "./cmd/settlement"
start_go_service "chain" "./cmd/chain"
start_go_service "api" "./cmd/api"
start_go_service "ws" "./cmd/ws"

echo "==> starting frontend"
start_service "web" "${WEB_DIR}" "npm run dev -- --hostname 127.0.0.1 --port 3000"

echo "==> starting dedicated admin service (Next.js)"
start_service "admin" "${ADMIN_DIR}" "npm run dev -- --hostname 127.0.0.1 --port $(port_from_addr "${FUNNYOPTION_ADMIN_HTTP_ADDR}")"

echo "==> waiting for http health checks"
if ! wait_for_http "$(http_url_from_addr "${FUNNYOPTION_API_HTTP_ADDR}")/healthz" 30; then
  echo "api health check failed"
  exit 1
fi
if ! wait_for_http "$(http_url_from_addr "${FUNNYOPTION_WS_HTTP_ADDR}")/healthz" 30; then
  echo "ws health check failed"
  exit 1
fi
if ! wait_for_http "http://127.0.0.1:3000" 60; then
  echo "frontend health check failed"
  exit 1
fi
if ! wait_for_http "$(http_url_from_addr "${FUNNYOPTION_ADMIN_HTTP_ADDR}")" 60; then
  echo "admin service health check failed"
  exit 1
fi

cat <<EOF

funnyoption local dev is up.

- frontend: http://127.0.0.1:3000
- admin:    $(http_url_from_addr "${FUNNYOPTION_ADMIN_HTTP_ADDR}")
- api: $(http_url_from_addr "${FUNNYOPTION_API_HTTP_ADDR}")
- ws:  $(http_url_from_addr "${FUNNYOPTION_WS_HTTP_ADDR}")
- logs: ${LOG_DIR}
- pids: ${RUN_DIR}
- admin runtime: dedicated Next.js operator service
- operator wallets: ${FUNNYOPTION_OPERATOR_WALLETS:-not configured}
- local chain mode: ${FUNNYOPTION_LOCAL_CHAIN_MODE:-disabled}
- local chain wallets: ${RUN_DIR}/local-chain-wallets.env

stop all:
  ${ROOT_DIR}/scripts/dev-down.sh

status:
  ${ROOT_DIR}/scripts/dev-status.sh
EOF
