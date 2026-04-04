# HANDSHAKE-OFFCHAIN-012

## Task

- [TASK-OFFCHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-012.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/operations/local-lifecycle-runbook.md`
- `docs/operations/local-offchain-lifecycle.md`
- `WORKLOG-OFFCHAIN-010.md`
- this handshake
- `WORKLOG-OFFCHAIN-012.md`

## Files in scope

- `cmd/local-lifecycle/**`
- `docs/operations/local-lifecycle-runbook.md`
- `docs/operations/local-offchain-lifecycle.md`
- `scripts/local-lifecycle.sh` only if needed
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-012.md`

## Inputs from other threads

- `TASK-OFFCHAIN-010` confirmed shared runtime behavior is healthy
- the local wrapper still fails because `cmd/local-lifecycle` submits a second
  explicit maker `SELL` after `/api/v1/admin/markets/:market_id/first-liquidity`
  already queued the bootstrap order
- this is a local proof-runner/docs mismatch, not a shared API/runtime product
  regression

## Outputs back to commander

- changed files
- exact validation commands
- proof that `./scripts/local-lifecycle.sh` no longer sends the stale duplicate
  maker sell
- any remaining blocker if the wrapper still fails

## Blockers

- do not reopen shared API, account, matching, settlement, or staging deploy
  behavior in this task
- keep scope narrow to the local lifecycle runner and docs

## Status

- completed

## Handoff notes

- `cmd/local-lifecycle` no longer submits a stale second explicit maker `SELL`
  after `/api/v1/admin/markets/:market_id/first-liquidity`
- `./scripts/local-lifecycle.sh` is green again on the current local stack
- docs now describe one-shot first-liquidity truthfully:
  - the admin first-liquidity endpoint both issues paired inventory and queues
    the bootstrap `SELL`
  - the runner now waits for that queued bootstrap order and only then places
    the crossing `BUY`
- residual local-state caveat remains documented in the worklog:
  - on persistent `anvil` plus reused local postgres, deterministic deposit
    evidence can be reused across runs unless the local DB is reset
