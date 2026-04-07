#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${FUNNYOPTION_STAGING_COMPOSE_FILE:-deploy/staging/docker-compose.staging.yml}"
ENV_FILE="${FUNNYOPTION_STAGING_ENV_FILE:-deploy/staging/.env.staging}"
DIFF_BASE_REF="${FUNNYOPTION_DEPLOY_DIFF_BASE:-}"
RELEASE_REF="${FUNNYOPTION_DEPLOY_REF:-}"
REMOTE_NAME="${FUNNYOPTION_DEPLOY_REMOTE:-origin}"
RUN_MIGRATIONS=1
RUN_GIT_SYNC=1
PRINT_PLAN=0
FORCE_ALL_SERVICES=0
EXPLICIT_SERVICES=()
PLANNED_SERVICES=()
CHANGED_PATHS=()
PLAN_SOURCE="full"
GO_VALIDATION_REQUIRED=0
WEB_VALIDATION_REQUIRED=0
ADMIN_VALIDATION_REQUIRED=0
MIGRATIONS_REQUIRED=1
APP_SERVICES=(account matching ledger settlement chain api ws market-maker notification web admin)
BACKEND_SERVICES=(account matching ledger settlement chain api ws market-maker notification)
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
  --diff-base <git-ref>           Infer impacted services from git diff against this ref.
  --service <name>                Deploy one compose service. Repeatable.
  --all-services                  Force a full service deploy.
  --print-plan                    Print the resolved deploy/validation plan and exit.
  --health-url <url>              Add one HTTP smoke-check URL. Repeatable.
  --skip-healthcheck              Skip all post-deploy HTTP smoke checks.
  --skip-git-sync                 Do not fetch/checkout a git ref before deploy.
  --skip-migrations               Skip the migrate compose profile.
  -h, --help                      Show this help text.

Environment:
  FUNNYOPTION_DEPLOY_DIFF_BASE
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

is_valid_service() {
  case "$1" in
    account|matching|ledger|settlement|chain|api|ws|market-maker|notification|web|admin)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

service_already_selected() {
  local target="$1"
  local service

  if [[ ${#PLANNED_SERVICES[@]} -gt 0 ]]; then
    for service in "${PLANNED_SERVICES[@]}"; do
      if [[ "${service}" == "${target}" ]]; then
        return 0
      fi
    done
  fi

  return 1
}

queue_service() {
  local service="$1"

  is_valid_service "${service}" || fail "unknown compose service: ${service}"

  if ! service_already_selected "${service}"; then
    PLANNED_SERVICES+=("${service}")
  fi
}

select_service() {
  local service="$1"

  queue_service "${service}"

  case "${service}" in
    account|matching|ledger|settlement|chain|api|ws|market-maker|notification)
      GO_VALIDATION_REQUIRED=1
      ;;
    web)
      WEB_VALIDATION_REQUIRED=1
      ;;
    admin)
      ADMIN_VALIDATION_REQUIRED=1
      ;;
  esac
}

select_all_services() {
  local service

  for service in "${APP_SERVICES[@]}"; do
    select_service "${service}"
  done
}

select_backend_services() {
  local service

  for service in "${BACKEND_SERVICES[@]}"; do
    select_service "${service}"
  done
}

queue_backend_services() {
  local service

  for service in "${BACKEND_SERVICES[@]}"; do
    queue_service "${service}"
  done
}

join_csv() {
  local IFS=,

  printf '%s' "$*"
}

should_run_migrations() {
  [[ "${RUN_MIGRATIONS}" -eq 1 && "${MIGRATIONS_REQUIRED}" -eq 1 ]]
}

should_skip_deploy() {
  [[ ${#PLANNED_SERVICES[@]} -eq 0 ]] && ! should_run_migrations
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

collect_changed_paths() {
  local base_ref="$1"
  local target_ref="$2"
  local path

  CHANGED_PATHS=()
  while IFS= read -r path; do
    [[ -n "${path}" ]] || continue
    CHANGED_PATHS+=("${path}")
  done < <(git -C "${ROOT_DIR}" diff --name-only "${base_ref}" "${target_ref}" --)
}

classify_changed_path() {
  local path="$1"

  case "${path}" in
    backend/go.mod|backend/go.sum|backend/internal/shared/*|backend/proto/*)
      select_backend_services
      ;;
    backend/internal/gen/accountv1/*)
      select_service account
      select_service api
      select_service chain
      ;;
    backend/migrations/*)
      queue_backend_services
      MIGRATIONS_REQUIRED=1
      ;;
    deploy/docker/*|deploy/staging/*|scripts/deploy-staging.sh)
      select_all_services
      MIGRATIONS_REQUIRED=1
      ;;
    backend/cmd/account/*|backend/internal/account/model/*|backend/internal/account/repository/*|backend/internal/account/service/*)
      select_service account
      ;;
    backend/internal/account/client/*)
      select_service account
      select_service api
      select_service chain
      ;;
    backend/cmd/matching/*|backend/internal/matching/*)
      select_service matching
      ;;
    backend/cmd/market-maker/*|backend/internal/marketmaker/*)
      select_service market-maker
      ;;
    backend/cmd/ledger/*|backend/internal/ledger/*)
      select_service ledger
      ;;
    backend/cmd/settlement/*|backend/internal/settlement/*)
      select_service settlement
      ;;
    backend/cmd/chain/*|backend/internal/chain/*)
      select_service chain
      ;;
    backend/cmd/api/*|backend/internal/api/*)
      select_service api
      ;;
    backend/cmd/ws/*|backend/internal/ws/*)
      select_service ws
      ;;
    backend/cmd/notification/*|backend/internal/notification/*)
      select_service notification
      ;;
    web/package.json|web/package-lock.json)
      select_service web
      select_service admin
      ;;
    web/*)
      select_service web
      ;;
    admin/*)
      select_service admin
      ;;
    *.go|backend/cmd/*|backend/internal/*)
      select_backend_services
      ;;
  esac
}

infer_plan_from_diff() {
  local target_ref="$1"
  local path

  PLAN_SOURCE="diff"
  MIGRATIONS_REQUIRED=0
  GO_VALIDATION_REQUIRED=0
  WEB_VALIDATION_REQUIRED=0
  ADMIN_VALIDATION_REQUIRED=0
  PLANNED_SERVICES=()

  if ! git -C "${ROOT_DIR}" rev-parse --verify "${DIFF_BASE_REF}^{commit}" >/dev/null 2>&1; then
    echo "deploy-staging: diff base ${DIFF_BASE_REF} not found; falling back to a full service deploy" >&2
    PLAN_SOURCE="full-fallback"
    MIGRATIONS_REQUIRED=1
    select_all_services
    return 0
  fi

  if ! git -C "${ROOT_DIR}" rev-parse --verify "${target_ref}^{commit}" >/dev/null 2>&1; then
    fail "target git ref not found: ${target_ref}"
  fi

  collect_changed_paths "${DIFF_BASE_REF}" "${target_ref}"
  if [[ ${#CHANGED_PATHS[@]} -gt 0 ]]; then
    for path in "${CHANGED_PATHS[@]}"; do
      classify_changed_path "${path}"
    done
  fi
}

resolve_plan() {
  local target_ref="${RELEASE_REF:-HEAD}"
  local service

  PLAN_SOURCE="full"
  CHANGED_PATHS=()
  PLANNED_SERVICES=()
  MIGRATIONS_REQUIRED=1
  GO_VALIDATION_REQUIRED=0
  WEB_VALIDATION_REQUIRED=0
  ADMIN_VALIDATION_REQUIRED=0

  if [[ "${FORCE_ALL_SERVICES}" -eq 1 ]]; then
    select_all_services
    return 0
  fi

  if [[ ${#EXPLICIT_SERVICES[@]} -gt 0 ]]; then
    PLAN_SOURCE="explicit"
    for service in "${EXPLICIT_SERVICES[@]}"; do
      select_service "${service}"
    done
    return 0
  fi

  if [[ -n "${DIFF_BASE_REF}" ]]; then
    infer_plan_from_diff "${target_ref}"
    return 0
  fi

  select_all_services
}

print_plan() {
  local changed_paths_csv=""
  local deploy_services_csv=""

  if [[ ${#CHANGED_PATHS[@]} -gt 0 ]]; then
    changed_paths_csv="$(join_csv "${CHANGED_PATHS[@]}")"
  fi

  if [[ ${#PLANNED_SERVICES[@]} -gt 0 ]]; then
    deploy_services_csv="$(join_csv "${PLANNED_SERVICES[@]}")"
  fi

  printf 'plan_source=%s\n' "${PLAN_SOURCE}"
  printf 'deploy_ref=%s\n' "${RELEASE_REF:-HEAD}"
  printf 'diff_base_ref=%s\n' "${DIFF_BASE_REF}"
  printf 'deploy_services=%s\n' "${deploy_services_csv}"
  if should_skip_deploy; then
    printf 'skip_deploy=1\n'
  else
    printf 'skip_deploy=0\n'
  fi
  if should_run_migrations; then
    printf 'run_migrations=1\n'
  else
    printf 'run_migrations=0\n'
  fi
  printf 'validate_go=%s\n' "${GO_VALIDATION_REQUIRED}"
  printf 'validate_web=%s\n' "${WEB_VALIDATION_REQUIRED}"
  printf 'validate_admin=%s\n' "${ADMIN_VALIDATION_REQUIRED}"
  printf 'changed_paths=%s\n' "${changed_paths_csv}"
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
    --diff-base)
      [[ $# -ge 2 ]] || fail "--diff-base requires a value"
      DIFF_BASE_REF="$2"
      shift 2
      ;;
    --service)
      [[ $# -ge 2 ]] || fail "--service requires a value"
      is_valid_service "$2" || fail "unknown compose service: $2"
      EXPLICIT_SERVICES+=("$2")
      shift 2
      ;;
    --all-services)
      FORCE_ALL_SERVICES=1
      shift
      ;;
    --print-plan)
      PRINT_PLAN=1
      shift
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

if [[ "${PRINT_PLAN}" -eq 0 ]]; then
  require_command docker
  require_command curl

  [[ -f "${ROOT_DIR}/${COMPOSE_FILE}" ]] || fail "compose file not found: ${ROOT_DIR}/${COMPOSE_FILE}"
  [[ -f "${ROOT_DIR}/${ENV_FILE}" ]] || fail "env file not found: ${ROOT_DIR}/${ENV_FILE}"
fi

if [[ "${FORCE_ALL_SERVICES}" -eq 1 && -n "${DIFF_BASE_REF}" ]]; then
  fail "--all-services and --diff-base cannot be used together"
fi

if [[ "${FORCE_ALL_SERVICES}" -eq 1 && ${#EXPLICIT_SERVICES[@]} -gt 0 ]]; then
  fail "--all-services and --service cannot be used together"
fi

if [[ -n "${DIFF_BASE_REF}" && ${#EXPLICIT_SERVICES[@]} -gt 0 ]]; then
  fail "--diff-base and --service cannot be used together"
fi

if [[ "${RUN_GIT_SYNC}" -eq 1 ]]; then
  sync_release_ref "${RELEASE_REF}"
fi

resolve_plan

if [[ "${PRINT_PLAN}" -eq 1 ]]; then
  print_plan
  exit 0
fi

if should_skip_deploy; then
  echo "no staging service changes detected; skipping compose deploy"
  echo "staging deployment completed"
  exit 0
fi

if should_run_migrations; then
  echo "running staging migrations"
  run_compose --profile ops run --rm migrate
fi

if [[ ${#PLANNED_SERVICES[@]} -gt 0 ]]; then
  echo "building and starting staging services: ${PLANNED_SERVICES[*]}"
  run_compose up -d --build --remove-orphans "${PLANNED_SERVICES[@]}"
else
  echo "no compose services selected; skipping service rebuild/restart"
fi

echo "current compose status"
run_compose ps

for url in "${HEALTHCHECK_URLS[@]}"; do
  run_healthcheck "${url}"
done

echo "staging deployment completed"
