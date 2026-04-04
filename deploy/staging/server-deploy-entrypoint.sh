#!/usr/bin/env bash

set -euo pipefail

LOCK_FILE="${FUNNYOPTION_STAGING_DEPLOY_LOCK_FILE:-/var/lock/funnyoption-staging-deploy.lock}"
REMOTE_NAME="${FUNNYOPTION_DEPLOY_REMOTE:-origin}"
REPO_PATH=""
TARGET_REF=""
FORCE_ALL_SERVICES=0

usage() {
  cat <<'EOF'
Usage: funnyoption-staging-deploy --repo <path> --ref <git-ref-or-sha> [options]

Fetch and deploy one exact staging target from a fixed host-side command path.

Options:
  --repo <path>                   Server-side repo checkout to deploy from.
  --ref <git-ref-or-sha>          Exact target commit or ref to fetch and deploy.
  --remote <name>                 Git remote used for fetches. Default: origin.
  --lock-file <path>              Host deploy lock file. Default: /var/lock/funnyoption-staging-deploy.lock
  --all-services                  Force a full service deploy instead of diffing against the current HEAD.
  -h, --help                      Show this help text.

Environment:
  FUNNYOPTION_DEPLOY_REMOTE
  FUNNYOPTION_STAGING_DEPLOY_LOCK_FILE
EOF
}

fail() {
  echo "funnyoption-staging-deploy: $*" >&2
  exit 1
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

looks_like_commit_sha() {
  [[ "$1" =~ ^[0-9a-fA-F]{7,40}$ ]]
}

resolve_commitish() {
  local candidate="$1"

  git -C "${REPO_PATH}" rev-parse --verify "${candidate}^{commit}" 2>/dev/null || return 1
}

has_tracked_changes() {
  ! git -C "${REPO_PATH}" diff --quiet --ignore-submodules --
}

has_staged_changes() {
  ! git -C "${REPO_PATH}" diff --cached --quiet --ignore-submodules --
}

resolve_target_commit() {
  local target_ref="$1"
  local branch_ref=""
  local remote_branch_ref=""
  local tag_ref=""

  if looks_like_commit_sha "${target_ref}" && resolve_commitish "${target_ref}" >/dev/null; then
    resolve_commitish "${target_ref}"
    return 0
  fi

  if [[ "${target_ref}" == refs/heads/* ]]; then
    branch_ref="${target_ref#refs/heads/}"
    remote_branch_ref="${REMOTE_NAME}/${branch_ref}"
  elif [[ "${target_ref}" == refs/remotes/${REMOTE_NAME}/* ]]; then
    remote_branch_ref="${target_ref#refs/remotes/}"
  elif [[ "${target_ref}" == refs/tags/* ]]; then
    tag_ref="${target_ref#refs/tags/}"
  elif [[ "${target_ref}" == "${REMOTE_NAME}/"* ]]; then
    remote_branch_ref="${target_ref}"
  else
    branch_ref="${target_ref}"
    remote_branch_ref="${REMOTE_NAME}/${branch_ref}"
  fi

  if [[ -n "${remote_branch_ref}" ]] && resolve_commitish "${remote_branch_ref}" >/dev/null; then
    resolve_commitish "${remote_branch_ref}"
    return 0
  fi

  if [[ -n "${tag_ref}" ]] && resolve_commitish "refs/tags/${tag_ref}" >/dev/null; then
    resolve_commitish "refs/tags/${tag_ref}"
    return 0
  fi

  if resolve_commitish "${target_ref}" >/dev/null; then
    resolve_commitish "${target_ref}"
    return 0
  fi

  if [[ -n "${branch_ref}" ]] && resolve_commitish "${branch_ref}" >/dev/null; then
    resolve_commitish "${branch_ref}"
    return 0
  fi

  if [[ -n "${tag_ref}" ]] && resolve_commitish "${tag_ref}" >/dev/null; then
    resolve_commitish "${tag_ref}"
    return 0
  fi

  return 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      [[ $# -ge 2 ]] || fail "--repo requires a value"
      REPO_PATH="$2"
      shift 2
      ;;
    --ref)
      [[ $# -ge 2 ]] || fail "--ref requires a value"
      TARGET_REF="$2"
      shift 2
      ;;
    --remote)
      [[ $# -ge 2 ]] || fail "--remote requires a value"
      REMOTE_NAME="$2"
      shift 2
      ;;
    --lock-file)
      [[ $# -ge 2 ]] || fail "--lock-file requires a value"
      LOCK_FILE="$2"
      shift 2
      ;;
    --all-services)
      FORCE_ALL_SERVICES=1
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

[[ -n "${REPO_PATH}" ]] || fail "--repo is required"
[[ -n "${TARGET_REF}" ]] || fail "--ref is required"

require_command bash
require_command flock
require_command git

git -C "${REPO_PATH}" rev-parse --is-inside-work-tree >/dev/null 2>&1 \
  || fail "repo checkout not found or not a git work tree: ${REPO_PATH}"

if ! exec 9<>"${LOCK_FILE}"; then
  fail "cannot open lock file ${LOCK_FILE}; ensure the deploy user can write it"
fi

echo "acquiring deploy lock ${LOCK_FILE}"
flock 9
echo "deploy lock acquired"

if has_tracked_changes || has_staged_changes; then
  fail "tracked git changes exist in ${REPO_PATH}; clean the checkout before deploying"
fi

previous_head="$(git -C "${REPO_PATH}" rev-parse --verify HEAD 2>/dev/null || true)"

echo "fetching ${TARGET_REF} from ${REMOTE_NAME}"
git -C "${REPO_PATH}" fetch --prune "${REMOTE_NAME}"

if ! target_commit="$(resolve_target_commit "${TARGET_REF}")"; then
  fail "target git ref not found after fetch: ${TARGET_REF}"
fi

echo "checking out ${target_commit}"
git -C "${REPO_PATH}" checkout --detach "${target_commit}"

deploy_args=(--skip-git-sync)
deploy_mode="selective"

if [[ "${FORCE_ALL_SERVICES}" -eq 1 ]]; then
  deploy_args+=(--all-services)
  deploy_mode="full"
elif [[ -n "${previous_head}" ]]; then
  deploy_args+=(--diff-base "${previous_head}")
else
  deploy_args+=(--all-services)
  deploy_mode="full-fallback"
fi

echo "repo_path=${REPO_PATH}"
echo "target_ref=${TARGET_REF}"
echo "target_commit=${target_commit}"
echo "diff_base=${previous_head}"
echo "deploy_mode=${deploy_mode}"
echo "lock_file=${LOCK_FILE}"

cd "${REPO_PATH}"
FUNNYOPTION_DEPLOY_REF="${target_commit}" bash ./scripts/deploy-staging.sh "${deploy_args[@]}"
