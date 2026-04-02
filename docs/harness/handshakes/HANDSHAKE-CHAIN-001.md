# HANDSHAKE-CHAIN-001

## Task

- [TASK-CHAIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-001.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/topics/kafka-topics.md`
- `TASK-OFFCHAIN-008.md`
- `HANDSHAKE-OFFCHAIN-008.md`
- `WORKLOG-OFFCHAIN-008.md`
- this handshake
- `WORKLOG-CHAIN-001.md`

## Files in scope

- `internal/api/dto/order.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/order_handler_test.go`
- `internal/api/handler/sql_store.go`
- `internal/chain/service/claim_processor.go`
- `internal/chain/service/claim_processor_test.go`

## Inputs from other threads

- `TASK-OFFCHAIN-008` closed the empty-collection contract gap, so off-chain read surfaces now tell the truth
- commander review found the next chain reliability gap:
  - claim API requests only trim/lowercase addresses today
  - claim processor turns malformed addresses into zero addresses via `common.HexToAddress(...)`

## Outputs back to commander

- changed files
- validation notes for invalid claim API requests
- validation notes for invalid queued claim tasks
- clear statement of whether malformed claim payloads are now blocked before zero-address submission

## Blockers

- do not widen into deposit listener work or unrelated frontend changes

## Status

- completed
