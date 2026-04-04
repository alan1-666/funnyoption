# TASK-OFFCHAIN-010

## Summary

Rerun the local core business flow after the API bootstrap-auth hardening sequence and return a concrete pass/fail matrix plus regression evidence.

## Scope

- validate the current local stack against the core lifecycle:
  - health check
  - admin market creation
  - admin first-liquidity bootstrap
  - user session creation
  - listener-driven deposit credit
  - user order placement and matching
  - market resolution
  - portfolio / orders / payouts read surfaces
- explicitly verify the `TASK-API-004` policy in runtime behavior:
  - the first privileged bootstrap sell order succeeds
  - a second otherwise-identical bootstrap sell order with a fresh operator proof is rejected with a clear duplicate response
  - a normal session-backed user order still succeeds
- record a pass/fail matrix, exact commands, key response snippets, and any blockers in the worklog
- do not make broad business-code edits in this task; if a regression is found, capture a precise failure report and hand it back to commander for follow-up tasking

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md](/Users/zhangza/code/funnyoption/docs/operations/core-business-test-flow.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md](/Users/zhangza/code/funnyoption/docs/operations/local-lifecycle-runbook.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md](/Users/zhangza/code/funnyoption/docs/operations/local-offchain-lifecycle.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md)

## Owned files

- `docs/harness/worklogs/WORKLOG-OFFCHAIN-010.md`
- no product code files by default

## Acceptance criteria

- worklog contains a pass/fail matrix for the local core business flow
- first admin bootstrap attempt succeeds in the proof run
- a second same-terms bootstrap attempt with a fresh operator proof is shown to be rejected by the current semantic-uniqueness policy
- one normal session-backed user order succeeds after that verification
- if any step fails, the handoff names the exact endpoint/script/log path and the smallest likely owner area for a follow-up worker

## Validation

- `/Users/zhangza/code/funnyoption/scripts/dev-up.sh`
- `/Users/zhangza/code/funnyoption/scripts/local-lifecycle.sh`
- targeted API checks from `docs/operations/core-business-test-flow.md`
- attach concise response snippets and log pointers in the worklog

## Dependencies

- `TASK-API-004` output is the baseline

## Handoff

- return the pass/fail matrix, exact commands, and proof snippets
- clearly separate confirmed regressions from environment-only blockers
- propose one narrow follow-up task per distinct failure, with suggested ownership
