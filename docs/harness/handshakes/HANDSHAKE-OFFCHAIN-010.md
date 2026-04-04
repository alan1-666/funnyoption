# HANDSHAKE-OFFCHAIN-010

## Task

- [TASK-OFFCHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-010.md)

## Thread owner

- implementation worker in validation-first mode

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/operations/core-business-test-flow.md`
- `docs/operations/local-lifecycle-runbook.md`
- `docs/operations/local-offchain-lifecycle.md`
- `WORKLOG-API-004.md`
- this handshake
- `WORKLOG-OFFCHAIN-010.md`

## Files in scope

- `docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md`
- no product code files unless commander explicitly retasks after a concrete regression report

## Inputs from other threads

- `TASK-API-004` is complete and should be validated end-to-end:
  - same-terms second privileged bootstrap sells are rejected even with a fresh `requested_at`
  - normal session-backed order writes should remain unchanged
- `docs/operations/core-business-test-flow.md` is the current manual flow checklist
- local lifecycle proof should stay listener-driven rather than falling back to simulated direct credits

## Outputs back to commander

- pass/fail matrix
- exact commands and response snippets for the proof run
- clear regression reports if any step fails
- suggested follow-up task split with likely owner modules

## Blockers

- do not silently patch business code in this validation pass
- if local env startup or wallet proof is blocked, capture the exact failing command and log path instead of guessing
- do not modify files owned by `TASK-CHAIN-003`

## Status

- completed

## Handoff notes

- runtime validation goal is complete:
  - local listener-driven deposit credit, duplicate bootstrap semantic uniqueness, normal session-backed order placement, market resolution, and terminal balances / positions / orders / payouts all match the current staging truth
- the only failure in the worklog is a tooling/docs mismatch, not a product-runtime regression:
  - `cmd/local-lifecycle` still submits a second explicit maker `SELL` after `/api/v1/admin/markets/:market_id/first-liquidity` already queued the bootstrap order
  - `scripts/local-lifecycle.sh` therefore fails on the wrapper proof path with `insufficient available balance`
- commander should route that wrapper/docs mismatch into a new narrow follow-up task instead of reopening broader API/order work
