#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${FUNNYOPTION_STAGING_COMPOSE_FILE:-deploy/staging/docker-compose.staging.yml}"
ENV_FILE="${FUNNYOPTION_STAGING_ENV_FILE:-deploy/staging/.env.staging}"
RELEASE_REF="${FUNNYOPTION_DEPLOY_REF:-}"
REMOTE_NAME="${FUNNYOPTION_DEPLOY_REMOTE:-origin}"
RUN_MIGRATIONS=1
RUN_GIT_SYNC=1
HEALTHCHECK_URLS=(
  "${FUNNYOPTION_STAGING_WEB_HEALTHCHECK_URL:-https://funnyoption.xyz/healthz}"
  "${FUNNYOPTION_STAGING_ADMIN_HEALTHCHECK_URL:-https://admin.funnyoption.xyz/}"
)

usage() {
  cat <<'EOF'
Usage: scripts/deploy-staging.sh [options]

Deploy the staging stack from an existing server-side repository checkout.

Options:
  --ref <git-ref>                 Fetch and deploy this ref or commit SHA.
  --remote <name>                 Git remote used for fetches. Default: origin.
  --compose-file <path>           Compose file path relative to repo root.
  --env-file <path>               Server-only env file path relative to repo root.
  --health-url <url>              Add one HTTP smoke-check URL. Repeatable.
  --skip-healthcheck              Skip all post-deploy HTTP smoke checks.
  --skip-git-sync                 Do not fetch/checkout a git ref before deploy.
  --skip-migrations               Skip the migrate compose profile.
  -h, --help                      Show this help text.

Environment:
  FUNNYOPTION_DEPLOY_REF
  FUNNYOPTION_DEPLOY_REMOTE
  FUNNYOPTION_STAGING_COMPOSE_FILE
  FUNNYOPTION_STAGING_ENV_FILE
  FUNNYOPTION_STAGING_WEB_HEALTHCHECK_URL
  FUNNYOPTION_STAGING_ADMIN_HEALTHCHECK_URL
EOF
}

fail() {
  echo "deploy-staging: $*" >&2
  exit 1
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

has_tracked_changes() {
  ! git -C "${ROOT_DIR}" diff --quiet --ignore-submodules --
}

has_staged_changes() {
  ! git -C "${ROOT_DIR}" diff --cached --quiet --ignore-submodules --
}

sync_release_ref() {
  local target_ref="$1"

  [[ -n "${target_ref}" ]] || fail "--ref is required when git sync is enabled"

  if has_tracked_changes || has_staged_changes; then
    fail "tracked git changes exist in ${ROOT_DIR}; clean the server checkout before deploying"
  fi

  echo "fetching ${target_ref} from ${REMOTE_NAME}"
  git -C "${ROOT_DIR}" fetch --prune "${REMOTE_NAME}"

  echo "checking out ${target_ref}"
  git -C "${ROOT_DIR}" checkout --detach "${target_ref}"
}

run_compose() {
  docker compose \
    --env-file "${ROOT_DIR}/${ENV_FILE}" \
    -f "${ROOT_DIR}/${COMPOSE_FILE}" \
    "$@"
}

run_healthcheck() {
  local url="$1"

  [[ -n "${url}" ]] || return 0

  echo "checking ${url}"
  curl --fail --show-error --silent --max-time 20 --retry 6 --retry-delay 5 "${url}" >/dev/null
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      [[ $# -ge 2 ]] || fail "--ref requires a value"
      RELEASE_REF="$2"
      shift 2
      ;;
    --remote)
      [[ $# -ge 2 ]] || fail "--remote requires a value"
      REMOTE_NAME="$2"
      shift 2
      ;;
    --compose-file)
      [[ $# -ge 2 ]] || fail "--compose-file requires a value"
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --env-file)
      [[ $# -ge 2 ]] || fail "--env-file requires a value"
      ENV_FILE="$2"
      shift 2
      ;;
    --health-url)
      [[ $# -ge 2 ]] || fail "--health-url requires a value"
      if [[ ${#HEALTHCHECK_URLS[@]} -eq 2 ]] \
        && [[ "${HEALTHCHECK_URLS[0]}" == "${FUNNYOPTION_STAGING_WEB_HEALTHCHECK_URL:-https://funnyoption.xyz/healthz}" ]] \
        && [[ "${HEALTHCHECK_URLS[1]}" == "${FUNNYOPTION_STAGING_ADMIN_HEALTHCHECK_URL:-https://admin.funnyoption.xyz/}" ]]; then
        HEALTHCHECK_URLS=()
      fi
      HEALTHCHECK_URLS+=("$2")
      shift 2
      ;;
    --skip-healthcheck)
      HEALTHCHECK_URLS=()
      shift
      ;;
    --skip-git-sync)
      RUN_GIT_SYNC=0
      shift
      ;;
    --skip-migrations)
      RUN_MIGRATIONS=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

require_command git
require_command docker
require_command curl

[[ -f "${ROOT_DIR}/${COMPOSE_FILE}" ]] || fail "compose file not found: ${ROOT_DIR}/${COMPOSE_FILE}"
[[ -f "${ROOT_DIR}/${ENV_FILE}" ]] || fail "env file not found: ${ROOT_DIR}/${ENV_FILE}"

if [[ "${RUN_GIT_SYNC}" -eq 1 ]]; then
  sync_release_ref "${RELEASE_REF}"
fi

if [[ "${RUN_MIGRATIONS}" -eq 1 ]]; then
  echo "running staging migrations"
  run_compose --profile ops run --rm migrate
fi

echo "building and starting staging services"
run_compose up -d --build --remove-orphans

echo "current compose status"
run_compose ps

for url in "${HEALTHCHECK_URLS[@]}"; do
  run_healthcheck "${url}"
done

echo "staging deployment completed"
