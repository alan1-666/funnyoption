# HANDSHAKE-OFFCHAIN-008

## Task

- [TASK-OFFCHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-008.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-OFFCHAIN-006.md`
- `HANDSHAKE-OFFCHAIN-006.md`
- `WORKLOG-OFFCHAIN-006.md`
- this handshake
- `WORKLOG-OFFCHAIN-008.md`

## Files in scope

- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/api/handler/order_handler_test.go`

## Inputs from other threads

- `TASK-OFFCHAIN-006` fixed the frontend so SSR surfaces now report broken collection responses honestly
- commander review found the next backend truthfulness gap: empty collection reads still come back as `{"items":null}` because nil slices flow through the API layer

## Outputs back to commander

- changed files
- validation notes for empty `trades` and empty `chain-transactions`
- clear statement of whether the local API now uses `[]` for empty list endpoints

## Handoff notes back to commander

- the API query/handler layer now normalizes nil collection results into `[]`, not `null`, across the current list/report endpoints in `OrderHandler`.
- empty `trades` and empty `chain-transactions` were both revalidated over HTTP on a temporary API instance and now return `{"items":[]}` with `HTTP 200`.
- the default local API on `127.0.0.1:8080` was stale during worker validation, so local developers need a restart if they want the default port to reflect this fix immediately.

## Blockers

- do not widen into chain execution logic or unrelated frontend work

## Status

- completed
