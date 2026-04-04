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

- paused
