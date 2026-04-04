# HANDSHAKE-CICD-003

## Task

- [TASK-CICD-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md)

## Thread owner

- implementation worker for platform/deployment bootstrap hardening

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-CICD-002.md`
- this handshake
- `WORKLOG-CICD-002.md`
- `WORKLOG-CICD-003.md`
- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`

## Files in scope

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-003.md`
- `docs/harness/worklogs/WORKLOG-CICD-003.md`

## Inputs from other threads

- `TASK-CICD-002` added selective deploy planning and service subset flags
- commander review found the rollout gap:
  - the workflow invokes `bash ./scripts/deploy-staging.sh --service ...` in the server's current checkout
  - `deploy-staging.sh` parses CLI flags before `sync_release_ref`
  - if the server still has an older script, newly introduced flags can be rejected before the new ref is fetched

## Outputs back to commander

- changed files:
  - `.github/workflows/staging-deploy.yml`
  - `docs/deploy/staging-bsc-testnet.md`
  - `docs/harness/handshakes/HANDSHAKE-CICD-003.md`
  - `docs/harness/worklogs/WORKLOG-CICD-003.md`
- final remote bootstrap sequence:
  - SSH into `STAGING_DEPLOY_PATH`
  - fail fast if tracked/staged local edits exist
  - `git fetch --prune origin`
  - `git checkout --detach <deploy_ref>`
  - invoke the checked-out `./scripts/deploy-staging.sh` with
    `--skip-git-sync`, the selected `--service` flags, and
    `--skip-migrations` when the plan says migrations are unnecessary
- validation commands and proof/dry-run notes are recorded in
  `WORKLOG-CICD-003.md`
- no one-time manual `git pull` is required after this fix; the only remaining
  operator action is to clean tracked/staged local edits in the server clone if
  a dirty checkout blocks the bootstrap guard

## Blockers

- preserve selective deploy and docs-only skip behavior
- preserve manual full deploy fallback
- do not weaken SSH/secrets handling

## Status

- completed
