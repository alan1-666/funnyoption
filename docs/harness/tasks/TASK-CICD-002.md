# TASK-CICD-002

## Summary

Optimize staging CI/CD so a push only validates, rebuilds, and redeploys services whose owned paths or shared dependencies changed; docs-only pushes should skip service deployment.

## Scope

- inspect the current `TASK-CICD-001` workflow/script implementation and define one explicit path-to-service change map
- update GitHub Actions so validation is selective:
  - Go tests run only when Go backend paths or shared Go dependencies changed
  - `web` build runs only when `web` or its frontend build dependencies changed
  - `admin` build runs only when `admin`, `web`, or its frontend build dependencies changed
  - docs/harness-only pushes skip service build/deploy but may still run lightweight syntax checks
- update `scripts/deploy-staging.sh` so remote compose rebuild/restart is limited to an explicit service subset when safe, instead of always rebuilding every service
- keep a safe fallback for broad-impact paths:
  - `go.mod`, `go.sum`, `internal/shared/**`, `proto/**`, `deploy/docker/**`, `deploy/staging/**`, and `scripts/deploy-staging.sh` can intentionally trigger all affected backend/frontend services
  - migration changes should still run the `migrate` compose profile before restarting dependent services
- document the change map, fallback policy, manual override behavior, and one example for docs-only/no-op deployment
- do not weaken secret handling or SSH hardening from `TASK-CICD-001`

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md)
- [/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml](/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml)
- [/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh)
- [/Users/zhangza/code/funnyoption/deploy/staging/docker-compose.staging.yml](/Users/zhangza/code/funnyoption/deploy/staging/docker-compose.staging.yml)
- [/Users/zhangza/code/funnyoption/deploy/docker](/Users/zhangza/code/funnyoption/deploy/docker)

## Owned files

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-002.md`
- `docs/harness/worklogs/WORKLOG-CICD-002.md`
- `deploy/docker/**` only if a narrow Dockerfile change is needed to preserve or improve cache boundaries

## Acceptance criteria

- a push touching only docs/harness files does not rebuild/redeploy backend or frontend services
- a push touching only one Go service path rebuilds/restarts only that service plus any explicitly required dependents, not the whole compose stack
- a push touching only `web/**` rebuilds/restarts `web` and `admin` only if admin actually depends on the changed `web` path according to the documented map
- migration changes still run migrations before restarting affected services
- manual `workflow_dispatch` still works and has a clear override story for full deploy versus selective deploy
- docs explain the service-change map and the safe fallback cases
- no plaintext secrets are introduced

## Validation

- `bash -n scripts/deploy-staging.sh`
- YAML syntax check for `.github/workflows/staging-deploy.yml`
- one local dry-run or scripted proof for:
  - docs-only change => no service deployment
  - one backend-service change => one service subset selected
  - broad shared change => safe fallback set selected
- if practical, one real staging workflow run after a tiny scoped change

## Dependencies

- `TASK-CICD-001` output is the baseline

## Handoff

- return the final path-to-service map, fallback policy, and changed files
- include validation commands and one or more dry-run proofs
- note any residual limitation caused by current Dockerfile build contexts or shared package coupling
