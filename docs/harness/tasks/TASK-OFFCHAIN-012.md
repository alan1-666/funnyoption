# TASK-OFFCHAIN-012

## Summary

Realign the local lifecycle runner and its docs with the current one-shot
first-liquidity contract so `./scripts/local-lifecycle.sh` becomes green again
without changing shared product semantics.

## Scope

- update the local proof runner so it does not submit a second explicit maker
  `SELL` after `POST /api/v1/admin/markets/:market_id/first-liquidity` already
  queued the bootstrap order
- update the local lifecycle docs/runbooks so they describe the current
  one-shot bootstrap behavior truthfully
- keep scope narrow:
  - `cmd/local-lifecycle/**`
  - `docs/operations/local-lifecycle-runbook.md`
  - `docs/operations/local-offchain-lifecycle.md`
  - `scripts/local-lifecycle.sh` only if needed for wrapper clarity
- do not reopen shared API, account, matching, settlement, or staging deploy
  behavior in this task

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md](/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-012.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-012.md)

## Owned files

- `cmd/local-lifecycle/**`
- `docs/operations/local-lifecycle-runbook.md`
- `docs/operations/local-offchain-lifecycle.md`
- `scripts/local-lifecycle.sh` only if a wrapper update is required
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-012.md`

## Acceptance criteria

- `./scripts/local-lifecycle.sh` finishes successfully on the current local
  stack without attempting a duplicate maker sell
- the local lifecycle docs describe the one-shot first-liquidity contract
  truthfully
- no shared product-runtime behavior changes are introduced
- worklog records the exact commands, resulting ids, and before/after behavior

## Validation

- `./scripts/dev-up.sh`
- `./scripts/local-lifecycle.sh`
- targeted readbacks or logs only if needed to prove the runner no longer sends
  the stale second sell

## Dependencies

- `TASK-OFFCHAIN-010` output is the baseline

## Handoff

- return changed files, validation commands, and the exact point where the old
  duplicate-sell step was removed
- if the wrapper still fails, capture the precise command/log and keep the
  follow-up narrow
