# TASK-OFFCHAIN-005

## Summary

Audit the historical stale-freeze issue found in reused local databases and produce a repeatable local-only runbook for detection and cleanup guidance without changing live trading code.

## Scope

- inspect the current schema and local DB state related to `orders`, `freeze_records`, and `account_balances`
- produce repeatable audit queries or a small helper script that identifies suspicious leftover freezes from the pre-fix BUY reserve bug
- document safe local repair guidance for reused developer databases
- explicitly separate local cleanup guidance from any future production-grade reconciliation design
- keep runtime service code out of scope while `TASK-OFFCHAIN-004` is active

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-005.md)

## Owned files

- `scripts/**`
- `docs/sql/**`
- `docs/deploy/**`
- `docs/operations/**`

## Acceptance criteria

- there is one repeatable way to audit a local DB for stale pre-fix freezes
- the repo contains a clear local-only runbook describing how to inspect and clean reused developer data
- the worker records whether the current local DB actually contains stale freezes and shows the evidence
- no runtime service files under `internal/**` are modified by this task

## Validation

- run the audit query or helper against the current local DB
- verify the documented cleanup steps match the current schema

## Dependencies

- `TASK-OFFCHAIN-002` findings are the input
- this task may run in parallel with `TASK-OFFCHAIN-004`

## Handoff

- return the audit helper, the runbook path, and the recommended next action:
  - local manual cleanup only
  - local scriptable cleanup
  - later productized reconciliation task
