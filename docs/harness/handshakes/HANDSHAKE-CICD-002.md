# HANDSHAKE-CICD-002

## Task

- [TASK-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-002.md)

## Thread owner

- implementation worker for platform/deployment optimization

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-CICD-001.md`
- this handshake
- `WORKLOG-CICD-001.md`
- `WORKLOG-CICD-002.md`
- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `deploy/staging/docker-compose.staging.yml`
- `deploy/docker/**`

## Files in scope

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-002.md`
- `docs/harness/worklogs/WORKLOG-CICD-002.md`
- `deploy/docker/**` only if one narrow cache-boundary fix is needed

## Inputs from other threads

- `TASK-CICD-001` push-to-deploy is working on staging, but it is currently full-stack:
  - `validate` always runs all Go tests and both frontend builds
  - remote deploy always runs `docker compose up -d --build --remove-orphans`
- service topology:
  - backend Go services: `api`, `ws`, `matching`, `account`, `ledger`, `settlement`, `chain`
  - frontend services: `web`, `admin`
- current Go service Dockerfiles use `COPY . .`, so docs/script-only changes may still invalidate backend build cache unless the deploy script skips unchanged services or Dockerfiles are narrowed carefully

## Outputs back to commander

- changed files
- final path-to-service change map
- selective deploy fallback policy
- validation notes and dry-run evidence
- residual limitations if any service still needs a broader rebuild than ideal

## Handoff notes

- final path-to-service map and fallback policy are documented in
  `docs/deploy/staging-bsc-testnet.md`
- `scripts/deploy-staging.sh --print-plan --diff-base <ref>` is the single
  source for selective service planning
- GitHub Actions `plan` job consumes that script output, `validate` gates
  Go/web/admin work by plan booleans, and `deploy-staging` skips entirely when
  `skip_deploy=1`
- manual `workflow_dispatch` keeps `deploy_scope=full` as the explicit full
  deploy override; `deploy_scope=selective` diffs the target ref against its
  first parent when possible
- residual limitation:
  - Go service Dockerfiles still use `COPY . .`, so cache invalidation from
    unrelated repo files remains broad inside any selected backend image build
  - service selection avoids restarting untouched services, but Docker cache
    pruning could still make a selected image rebuild more expensive than ideal

## Blockers

- do not break the currently working full deploy path; keep a manual full-deploy fallback
- do not weaken SSH/secrets handling
- if `admin` depends on `web` shared package changes, model that dependency explicitly rather than incorrectly skipping admin rebuilds
- if a path-to-service map is too risky for some shared directories, choose a conservative fallback and document it
- commander review found a rollout blocker:
  - workflow now passes `--service` / selective flags to the remote script in the server's current checkout
  - the script parses flags before `sync_release_ref`
  - therefore a server checkout with an older `scripts/deploy-staging.sh` can reject newly added flags before it fetches the target ref
  - hand this to `TASK-CICD-003`; do not mark the selective-deploy rollout as production-safe until that bootstrap order is fixed

## Status

- completed
