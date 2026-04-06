#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${FUNNYOPTION_ENV_FILE:-${ROOT_DIR}/.env.local}"
RUN_DIR="${ROOT_DIR}/.run/dev"
LOG_DIR="${ROOT_DIR}/.logs/dev"
PID_FILE="${RUN_DIR}/anvil.pid"
LOG_FILE="${LOG_DIR}/anvil.log"
LOCAL_CHAIN_ENV="${RUN_DIR}/local-chain.env"
LOCAL_CHAIN_WALLETS="${RUN_DIR}/local-chain-wallets.env"
FRESH_START_FILE="${RUN_DIR}/local-chain-fresh-start"

mkdir -p "${RUN_DIR}" "${LOG_DIR}"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  source "${ENV_FILE}"
  set +a
fi

: "${FUNNYOPTION_LOCAL_CHAIN_MODE:=anvil}"
: "${FUNNYOPTION_LOCAL_CHAIN_HOST:=127.0.0.1}"
: "${FUNNYOPTION_LOCAL_CHAIN_PORT:=8545}"
: "${FUNNYOPTION_LOCAL_CHAIN_CHAIN_ID:=31337}"
: "${FUNNYOPTION_LOCAL_CHAIN_MNEMONIC:=test test test test test test test test test test test junk}"
: "${FUNNYOPTION_LOCAL_CHAIN_OPERATOR_MNEMONIC_INDEX:=0}"
: "${FUNNYOPTION_LOCAL_CHAIN_BUYER_PRIVATE_KEY:=0x59c6995e998f97a5a004497e5daef0d4f7dcd0cfd5401397dbeed52b21965b1d}"
: "${FUNNYOPTION_LOCAL_CHAIN_MAKER_PRIVATE_KEY:=0x8b3a350cf5c34c9194ca85829f093d784c2f2c6c3a0eb1f3f3f94a639a6a39d1}"

RPC_URL="http://${FUNNYOPTION_LOCAL_CHAIN_HOST}:${FUNNYOPTION_LOCAL_CHAIN_PORT}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

is_running() {
  local pid_file="$1"
  [[ -f "${pid_file}" ]] || return 1
  local pid
  pid="$(cat "${pid_file}" 2>/dev/null || true)"
  [[ -n "${pid}" ]] || return 1
  kill -0 "${pid}" >/dev/null 2>&1
}

port_is_listening() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
}

describe_port_listener() {
  local port="$1"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true
}

wait_for_rpc() {
  local elapsed=0
  while (( elapsed < 30 )); do
    if rpc_block_number >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  return 1
}

normalize_hex() {
  local value="$1"
  value="${value#0x}"
  value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
  echo "0x${value}"
}

rpc_request() {
  local method="$1"
  local params="${2:-[]}"
  curl -fsS \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}" \
    "${RPC_URL}"
}

rpc_block_number() {
  local response
  response="$(rpc_request "eth_blockNumber")" || return 1
  printf '%s\n' "${response}" | sed -n 's/.*"result":"\([^"]*\)".*/\1/p' | tail -n 1
}

rpc_code() {
  local address="$1"
  local response
  response="$(rpc_request "eth_getCode" "[\"${address}\",\"latest\"]")" || return 1
  printf '%s\n' "${response}" | sed -n 's/.*"result":"\([^"]*\)".*/\1/p' | tail -n 1
}

contract_exists() {
  local address="$1"
  [[ -n "${address}" ]] || return 1
  local code
  code="$(rpc_code "${address}" 2>/dev/null || true)"
  [[ -n "${code}" && "${code}" != "0x" ]]
}

read_env_value() {
  local file="$1"
  local key="$2"
  [[ -f "${file}" ]] || return 0
  sed -n "s/^export ${key}=//p; s/^${key}=//p" "${file}" | tail -n 1 | sed -e "s/^'//" -e "s/'$//"
}

compile_contracts() {
  local output
  if ! output="$(cd "${ROOT_DIR}" && forge build 2>&1)"; then
    echo "${output}" >&2
    cat >&2 <<EOF

local-chain bootstrap could not compile the contracts.
If this is the first Foundry run on this machine, you may need network access once so Foundry can install solc 0.8.24.
After that, rerun:
  ${ROOT_DIR}/scripts/local-chain-up.sh
EOF
    exit 1
  fi
}

deploy_contract() {
  local contract="$1"
  shift
  local output
  if ! output="$(cd "${ROOT_DIR}" && forge create "${contract}" --broadcast --rpc-url "${RPC_URL}" --private-key "${OPERATOR_PRIVATE_KEY}" "$@" 2>&1)"; then
    echo "${output}" >&2
    exit 1
  fi
  local address
  address="$(printf '%s\n' "${output}" | sed -n 's/^Deployed to: //p' | tail -n 1 | tr '[:upper:]' '[:lower:]')"
  if [[ -z "${address}" ]]; then
    echo "${output}" >&2
    echo "failed to parse deployed address for ${contract}" >&2
    exit 1
  fi
  echo "${address}"
}

send_eth() {
  local to="$1"
  local amount="$2"
  cast send --rpc-url "${RPC_URL}" --private-key "${OPERATOR_PRIVATE_KEY}" --value "${amount}" "${to}" >/dev/null
}

mint_token() {
  local to="$1"
  local amount="$2"
  cast send --rpc-url "${RPC_URL}" --private-key "${OPERATOR_PRIVATE_KEY}" "${TOKEN_ADDRESS}" "mint(address,uint256)" "${to}" "${amount}" >/dev/null
}

write_outputs() {
  local start_block="$1"

  cat >"${LOCAL_CHAIN_ENV}" <<EOF
export FUNNYOPTION_LOCAL_CHAIN_MODE='anvil'
export FUNNYOPTION_CHAIN_RPC_URL='${RPC_URL}'
export FUNNYOPTION_CHAIN_RPC_FALLBACK_URLS=''
export FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY='${OPERATOR_PRIVATE_KEY}'
export FUNNYOPTION_CHAIN_NAME='anvil'
export FUNNYOPTION_NETWORK_NAME='local'
export FUNNYOPTION_CHAIN_ID='${FUNNYOPTION_LOCAL_CHAIN_CHAIN_ID}'
export FUNNYOPTION_VAULT_ADDRESS='${VAULT_ADDRESS}'
export FUNNYOPTION_CHAIN_CONFIRMATIONS='0'
export FUNNYOPTION_CHAIN_START_BLOCK='${start_block}'
export FUNNYOPTION_CHAIN_POLL_INTERVAL='1s'
export FUNNYOPTION_CHAIN_CLAIM_POLL_INTERVAL='3s'
export FUNNYOPTION_ROLLUP_POLL_INTERVAL='3s'
export FUNNYOPTION_ROLLUP_BATCH_LIMIT='256'
export FUNNYOPTION_CHAIN_GAS_LIMIT='250000'
export FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS='${TOKEN_ADDRESS}'
export FUNNYOPTION_ROLLUP_CORE_ADDRESS='${ROLLUP_CORE_ADDRESS}'
export FUNNYOPTION_ROLLUP_VERIFIER_ADDRESS='${ROLLUP_VERIFIER_ADDRESS}'
export FUNNYOPTION_COLLATERAL_SYMBOL='USDT'
export FUNNYOPTION_COLLATERAL_DECIMALS='6'
export FUNNYOPTION_COLLATERAL_ACCOUNTING_DECIMALS='2'
export FUNNYOPTION_FRONTEND_CHAIN_NAME='Anvil Local'
export FUNNYOPTION_CHAIN_EXPLORER_URL=''
export FUNNYOPTION_NATIVE_CURRENCY_NAME='Ethereum'
export FUNNYOPTION_NATIVE_CURRENCY_SYMBOL='ETH'
export FUNNYOPTION_NATIVE_CURRENCY_DECIMALS='18'
export FUNNYOPTION_OPERATOR_WALLETS='${OPERATOR_ADDRESS}'
EOF

  cat >"${LOCAL_CHAIN_WALLETS}" <<EOF
export FUNNYOPTION_LOCAL_CHAIN_OPERATOR_ADDRESS=${OPERATOR_ADDRESS}
export FUNNYOPTION_LOCAL_CHAIN_OPERATOR_PRIVATE_KEY=${OPERATOR_PRIVATE_KEY}

export FUNNYOPTION_LOCAL_CHAIN_BUYER_USER_ID=1001
export FUNNYOPTION_LOCAL_CHAIN_BUYER_ADDRESS=${BUYER_ADDRESS}
export FUNNYOPTION_LOCAL_CHAIN_BUYER_PRIVATE_KEY=${BUYER_PRIVATE_KEY}

export FUNNYOPTION_LOCAL_CHAIN_MAKER_USER_ID=1002
export FUNNYOPTION_LOCAL_CHAIN_MAKER_ADDRESS=${MAKER_ADDRESS}
export FUNNYOPTION_LOCAL_CHAIN_MAKER_PRIVATE_KEY=${MAKER_PRIVATE_KEY}
EOF
}

require_command anvil
require_command forge
require_command cast
require_command curl
require_command lsof

if [[ "${FUNNYOPTION_LOCAL_CHAIN_MODE}" != "anvil" ]]; then
  echo "FUNNYOPTION_LOCAL_CHAIN_MODE must be 'anvil' to use ${BASH_SOURCE[0]}" >&2
  exit 1
fi

if is_running "${PID_FILE}"; then
  echo "reusing managed anvil (pid $(cat "${PID_FILE}"))"
  rm -f "${FRESH_START_FILE}"
else
  if port_is_listening "${FUNNYOPTION_LOCAL_CHAIN_PORT}"; then
    echo "local chain port ${FUNNYOPTION_LOCAL_CHAIN_PORT} is already in use" >&2
    describe_port_listener "${FUNNYOPTION_LOCAL_CHAIN_PORT}" >&2
    exit 1
  fi

  rm -f "${PID_FILE}"
  nohup anvil \
    --host "${FUNNYOPTION_LOCAL_CHAIN_HOST}" \
    --port "${FUNNYOPTION_LOCAL_CHAIN_PORT}" \
    --chain-id "${FUNNYOPTION_LOCAL_CHAIN_CHAIN_ID}" \
    --mnemonic "${FUNNYOPTION_LOCAL_CHAIN_MNEMONIC}" \
    >"${LOG_FILE}" 2>&1 &
  echo $! >"${PID_FILE}"

  if ! wait_for_rpc; then
    echo "anvil did not become ready. log: ${LOG_FILE}" >&2
    tail -n 40 "${LOG_FILE}" >&2 || true
    exit 1
  fi
  echo "started managed anvil on ${RPC_URL} (pid $(cat "${PID_FILE}"))"
  printf 'fresh\n' >"${FRESH_START_FILE}"
fi

OPERATOR_PRIVATE_KEY="$(normalize_hex "$(cast wallet private-key --mnemonic "${FUNNYOPTION_LOCAL_CHAIN_MNEMONIC}" --mnemonic-index "${FUNNYOPTION_LOCAL_CHAIN_OPERATOR_MNEMONIC_INDEX}")")"
OPERATOR_ADDRESS="$(cast wallet address --private-key "${OPERATOR_PRIVATE_KEY}" | tr '[:upper:]' '[:lower:]')"
BUYER_PRIVATE_KEY="$(normalize_hex "${FUNNYOPTION_LOCAL_CHAIN_BUYER_PRIVATE_KEY}")"
BUYER_ADDRESS="$(cast wallet address --private-key "${BUYER_PRIVATE_KEY}" | tr '[:upper:]' '[:lower:]')"
MAKER_PRIVATE_KEY="$(normalize_hex "${FUNNYOPTION_LOCAL_CHAIN_MAKER_PRIVATE_KEY}")"
MAKER_ADDRESS="$(cast wallet address --private-key "${MAKER_PRIVATE_KEY}" | tr '[:upper:]' '[:lower:]')"

TOKEN_ADDRESS=""
VAULT_ADDRESS=""
ROLLUP_CORE_ADDRESS=""
ROLLUP_VERIFIER_ADDRESS=""
EXISTING_START_BLOCK=""
if [[ -f "${LOCAL_CHAIN_ENV}" ]]; then
  TOKEN_ADDRESS="$(read_env_value "${LOCAL_CHAIN_ENV}" "FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS")"
  VAULT_ADDRESS="$(read_env_value "${LOCAL_CHAIN_ENV}" "FUNNYOPTION_VAULT_ADDRESS")"
  ROLLUP_CORE_ADDRESS="$(read_env_value "${LOCAL_CHAIN_ENV}" "FUNNYOPTION_ROLLUP_CORE_ADDRESS")"
  ROLLUP_VERIFIER_ADDRESS="$(read_env_value "${LOCAL_CHAIN_ENV}" "FUNNYOPTION_ROLLUP_VERIFIER_ADDRESS")"
  EXISTING_START_BLOCK="$(read_env_value "${LOCAL_CHAIN_ENV}" "FUNNYOPTION_CHAIN_START_BLOCK")"
fi

DEPLOYED_FRESH=0
if ! contract_exists "${TOKEN_ADDRESS}" || ! contract_exists "${VAULT_ADDRESS}"; then
  DEPLOYED_FRESH=1
  compile_contracts
  TOKEN_ADDRESS="$(deploy_contract "contracts/src/MockUSDT.sol:MockUSDT")"
  VAULT_ADDRESS="$(deploy_contract "contracts/src/FunnyVault.sol:FunnyVault" --constructor-args "${TOKEN_ADDRESS}" "${OPERATOR_ADDRESS}")"

  send_eth "${BUYER_ADDRESS}" "10ether"
  send_eth "${MAKER_ADDRESS}" "10ether"

  mint_token "${OPERATOR_ADDRESS}" "500000000000"
  mint_token "${BUYER_ADDRESS}" "500000000000"
  mint_token "${MAKER_ADDRESS}" "500000000000"
fi

if ! contract_exists "${ROLLUP_CORE_ADDRESS}" || ! contract_exists "${ROLLUP_VERIFIER_ADDRESS}"; then
  compile_contracts
  GENESIS_STATE_ROOT="$(cd "${ROOT_DIR}" && go run ./cmd/rollup -mode=print-genesis-root | sed -n 's/.*"genesis_state_root": *"\([^"]*\)".*/\1/p' | tail -n 1)"
  if [[ -z "${GENESIS_STATE_ROOT}" ]]; then
    echo "failed to compute rollup genesis state root" >&2
    exit 1
  fi
  ROLLUP_CORE_ADDRESS="$(deploy_contract "contracts/src/FunnyRollupCore.sol:FunnyRollupCore" --constructor-args "${OPERATOR_ADDRESS}" "${GENESIS_STATE_ROOT}")"
  ROLLUP_VERIFIER_ADDRESS="$(deploy_contract "contracts/src/FunnyRollupVerifier.sol:FunnyRollupVerifier")"
  cast send --rpc-url "${RPC_URL}" --private-key "${OPERATOR_PRIVATE_KEY}" "${ROLLUP_CORE_ADDRESS}" "setVerifier(address)" "${ROLLUP_VERIFIER_ADDRESS}" >/dev/null
fi

if (( DEPLOYED_FRESH == 0 )) && [[ -n "${EXISTING_START_BLOCK}" ]]; then
  START_BLOCK="${EXISTING_START_BLOCK}"
else
  START_BLOCK="$(rpc_block_number)"
  START_BLOCK="${START_BLOCK#0x}"
  START_BLOCK="$((16#${START_BLOCK}))"
  START_BLOCK="$((START_BLOCK + 1))"
fi
write_outputs "${START_BLOCK}"

cat <<EOF
local anvil chain is ready.

- rpc: ${RPC_URL}
- chain_id: ${FUNNYOPTION_LOCAL_CHAIN_CHAIN_ID}
- token: ${TOKEN_ADDRESS}
- vault: ${VAULT_ADDRESS}
- rollup_core: ${ROLLUP_CORE_ADDRESS}
- rollup_verifier: ${ROLLUP_VERIFIER_ADDRESS}
- operator wallet: ${OPERATOR_ADDRESS}
- buyer wallet: ${BUYER_ADDRESS}
- maker wallet: ${MAKER_ADDRESS}
- generated env: ${LOCAL_CHAIN_ENV}
- wallet file: ${LOCAL_CHAIN_WALLETS}
EOF
