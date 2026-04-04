# TASK-CHAIN-003

## Summary

Reconcile the legacy local `chain_deposits` schema drift seen in reused databases and add a safe, documented repair path that does not depend on silent runtime compatibility workarounds alone.

## Scope

- inspect the `TASK-CHAIN-002` handoff and current migrations/docs for the `chain_deposits` width drift
- verify the exact current schema shape expected by repo migrations versus the legacy shape observed in older local databases
- add a narrow, safe repair path for reused local DBs if drift is confirmed:
  - idempotent SQL or migration guidance
  - dry-run / rollback-safe validation commands
  - clear warnings about the intended environment and non-goals
- keep listener-driven local deposit proof working after the repair path is documented
- stay out of order/session API code and bootstrap-order semantics

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-003.md)

## Owned files

- `migrations/**` if a forward migration or repair migration is needed
- `docs/sql/**`
- `docs/operations/**` for narrowly related runbook updates
- `docs/harness/worklogs/WORKLOG-CHAIN-003.md`
- `internal/chain/service/**` only if a schema repair lets a compatibility workaround be simplified without changing product semantics

## Acceptance criteria

- repo clearly states the expected `chain_deposits` schema shape and the observed legacy drift
- a reused local DB with legacy `chain_deposits` widths has a safe repair path or a clearly documented blocker if no safe generic repair can be proven
- repair guidance is idempotent or dry-run/rollback-safe
- listener-driven local deposit proof still passes after the documented repair path is applied
- no order/session API behavior changes are introduced in this task

## Validation

- inspect current migration DDL and a running local DB schema if available
- if SQL repair is added, validate with a rollback-safe dry run against the local DB
- `cd /Users/zhangza/code/funnyoption && go test ./internal/chain/...`
- optionally rerun the local lifecycle proof if the chain service or chain docs change

## Dependencies

- `TASK-CHAIN-002` output is the baseline

## Handoff

- return the exact schema drift finding and the chosen repair path
- include dry-run / apply commands and any proof snippets
- if a code-level compatibility workaround remains intentionally, explain why and what future condition would let us remove it
