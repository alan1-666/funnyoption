# TASK-CICD-003

## Summary

Fix the selective-deploy bootstrap order so GitHub Actions can safely deploy a commit that changes `scripts/deploy-staging.sh` even when the server checkout still has an older copy of that script.

## Scope

- inspect the `TASK-CICD-002` workflow/script changes and the commander review note in `HANDSHAKE-CICD-002.md`
- make the remote deploy path self-bootstrap-safe:
  - fetch and check out the target ref before executing the deploy script version from that ref, or
  - invoke the target-ref script through another robust mechanism that does not depend on the old checkout's argument parser understanding new flags
- keep selective service deployment, docs-only skip behavior, and manual full-deploy override intact
- do not weaken SSH/secrets handling
- document the final bootstrap behavior and any one-time operator action that is still needed after this fix

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CICD-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CICD-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CICD-003.md)
- [/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml](/Users/zhangza/code/funnyoption/.github/workflows/staging-deploy.yml)
- [/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh](/Users/zhangza/code/funnyoption/scripts/deploy-staging.sh)
- [/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)

## Owned files

- `.github/workflows/staging-deploy.yml`
- `scripts/deploy-staging.sh`
- `docs/deploy/staging-bsc-testnet.md`
- `docs/harness/handshakes/HANDSHAKE-CICD-003.md`
- `docs/harness/worklogs/WORKLOG-CICD-003.md`

## Acceptance criteria

- a server checkout that still has an older `scripts/deploy-staging.sh` can successfully deploy a newer commit whose workflow passes selective-deploy flags
- selective service deployment and docs-only no-op behavior from `TASK-CICD-002` still work
- manual full deploy remains available
- validation includes one proof or dry-run that the remote command no longer depends on the old checkout's deploy-script parser before syncing the target ref

## Validation

- `bash -n scripts/deploy-staging.sh`
- YAML syntax check for `.github/workflows/staging-deploy.yml`
- one local or staging dry-run/proof for the bootstrap-safe remote invocation path

## Dependencies

- `TASK-CICD-002` output is the baseline

## Handoff

- return changed files, the final remote bootstrap sequence, and validation notes
- call out any remaining one-time manual server action if an old checkout still needs it
