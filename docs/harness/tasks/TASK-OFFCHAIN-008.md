# TASK-OFFCHAIN-008

## Summary

Normalize the local API collection contract so empty list endpoints return `{"items":[]}` instead of `{"items":null}`, allowing homepage, detail, and control to distinguish healthy empty state from broken responses.

## Scope

- fix the API/query layer so collection endpoints serialize empty result sets as empty arrays
- cover at least the read surfaces already exercised by the off-chain UI:
  - `GET /api/v1/trades`
  - `GET /api/v1/chain-transactions`
- prefer a shared fix that also keeps the other `OrderHandler` collection endpoints consistent
- add or update handler tests so empty collection semantics are protected

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-008.md)

## Owned files

- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/api/handler/order_handler_test.go`

## Acceptance criteria

- empty collection endpoints used by the live read surfaces return `{"items":[]}` instead of `{"items":null}`
- the fix is applied consistently enough that new empty collection endpoints do not silently regress to `null`
- handler or query-layer tests cover empty list serialization
- worker records curl or HTTP evidence for the empty collection contract in the worklog

## Validation

- `cd /Users/zhangza/code/funnyoption && go test ./internal/api/...`
- curl or equivalent checks that prove empty `trades` and empty `chain-transactions` now serialize as `{"items":[]}`

## Dependencies

- `TASK-OFFCHAIN-006` output is the baseline

## Handoff

- return the corrected collection-response contract
- note whether any remaining off-chain read-surface blockers still exist before chain hardening
