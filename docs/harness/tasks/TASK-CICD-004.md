# TASK-CICD-004

## Summary

Simplify staging push-to-deploy so GitHub Actions becomes a thin trigger that
invokes one fixed server-side deploy entrypoint, while the server entrypoint
fetches the exact target SHA and delegates selective rebuild/restart planning
to the repo deploy script.

## Scope

- inspect the current `TASK-CICD-001` through `TASK-CICD-003` workflow/script
  shape and identify which orchestration logic should move out of
  `.github/workflows/staging-deploy.yml`
- add one repo-owned, server-installable shell entrypoint for staging deploys
  so the server has a stable command path that GitHub Actions can call on every
  push
- use this concrete install target unless a blocking server constraint is
  discovered:
  - installed host path: `/usr/local/bin/funnyoption-staging-deploy`
  - installed from repo source path:
    `deploy/staging/server-deploy-entrypoint.sh`
  - deploy lock file: `/var/lock/funnyoption-staging-deploy.lock`
  - repo clone path argument: `--repo /opt/funnyoption-staging`
- simplify GitHub Actions so it primarily:
  - resolves the target SHA/ref
  - authenticates over SSH
  - invokes the fixed server-side entrypoint with explicit deploy intent
- default push behavior should be:
  - capture the currently deployed repo `HEAD` as the diff base
  - fetch and check out the requested target SHA/ref
  - call repo `scripts/deploy-staging.sh --skip-git-sync --diff-base <previous_head>`
  - let the repo deploy script continue deciding docs-only no-op deploys,
    selective service restarts, and migration needs
- keep the server-side deploy semantics safe:
  - fetch and deploy the exact target SHA, not an implicit `git pull origin main`
  - preserve the dirty-checkout guard before deploy
  - preserve selective service rebuild/restart and docs-only no-op behavior
  - preserve or add a host-level deploy lock so overlapping pushes do not race
  - preserve a manual full-deploy override and rollback story
- document the one-time server install/update procedure for the fixed entrypoint
- do not introduce plaintext secrets or weaken SSH host verification

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-001.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-004.md)
- [/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml](/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml)
- [/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh)
- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)

## Owned files

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `deploy/staging/server-deploy-entrypoint.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-004.md`
- `docs/harness/worklogs/WORKLOG-CICD-004.md`

## Acceptance criteria

- `push` to `main` still triggers staging deployment, but the workflow now acts
  as a thin SSH trigger instead of embedding most deploy orchestration inline
- one fixed server-side entrypoint can be installed once and then reused across
  deploys
- the fixed server-side entrypoint path and lock path match the chosen
  commander decision unless the worker documents a blocking server constraint:
  - `/usr/local/bin/funnyoption-staging-deploy`
  - `/var/lock/funnyoption-staging-deploy.lock`
- the server entrypoint deploys the exact requested SHA/ref, not the moving tip
  of `main`
- selective service deploys, docs-only no-op deploys, migration behavior, and
  manual full-deploy override still work after the simplification
- overlapping deploy triggers are serialized safely
- docs explain:
  - the server install path for the fixed entrypoint
  - the GitHub secret inputs still required
  - manual deploy, redeploy, and rollback commands under the new flow

## Validation

- `bash -n scripts/deploy-staging.sh`
- `bash -n deploy/staging/server-deploy-entrypoint.sh`
- YAML syntax check for `.github/workflows/staging-deploy.yml`
- one local or staging proof that the workflow now calls the fixed server-side
  entrypoint with an explicit target SHA/ref
- one local or staging proof that docs-only and one-service deploy selection
  still resolve correctly after the simplification

## Dependencies

- `TASK-CICD-003` output is the baseline

## Handoff

- return changed files and the final deploy control flow
- record the install location and invocation shape for the server entrypoint
- include validation commands and at least one proof for exact-SHA deploy plus
  one proof for selective deploy behavior
- call out any residual compromise, such as checks intentionally removed from
  GitHub Actions and moved onto the server
