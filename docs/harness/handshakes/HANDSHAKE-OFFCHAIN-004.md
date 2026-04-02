# HANDSHAKE-OFFCHAIN-004

## Task

- [TASK-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-004.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/ledger-service.md`
- `docs/sql/schema.md`
- `TASK-OFFCHAIN-002.md`
- `HANDSHAKE-OFFCHAIN-002.md`
- `WORKLOG-OFFCHAIN-002.md`
- this handshake
- `WORKLOG-OFFCHAIN-004.md`

## Files in scope

- `internal/api/handler/**`
- `internal/matching/engine/**`
- `internal/matching/service/**`
- `internal/account/service/**`
- `internal/settlement/service/**`
- `README.md`

## Inputs from other threads

- `TASK-OFFCHAIN-002` established the blocker with exact reproduction:
  - resolved `market 1101` still accepted a new order with HTTP `202`
  - matching restored two resting orders for the resolved market on cold start
  - a new post-resolution trade `trd_6` was produced, breaking settlement finality
- `TASK-OFFCHAIN-003` stays blocked until this task lands

## Outputs back to commander

- changed files
- exact finality regression steps and observed status codes
- note on whether stale pre-fix freezes need a separate reconciliation task

## Handoff notes back to commander

- finality fix landed across ingress, matching restore/command gating, settlement cancellation flow, and account freeze terminality
- changed files:
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/matching/model/types.go`
  - `internal/matching/service/consumer.go`
  - `internal/matching/service/consumer_test.go`
  - `internal/matching/service/sql_store.go`
  - `internal/account/service/balance_book.go`
  - `internal/account/service/balance_book_test.go`
  - `internal/settlement/service/processor.go`
  - `internal/settlement/service/processor_test.go`
  - `internal/settlement/service/sql_store.go`
  - `internal/settlement/service/store.go`
- exact regression result:
  - clean proof market: `220140402`
  - create market -> HTTP `201`
  - create resting buy order -> HTTP `202`, `order_id=ord_1775048079527_112f29d596a0`, `freeze_id=frz_1775048079532_51c8130d1cfb`, `amount=308`
  - resolve market -> HTTP `202`
  - post-resolve observations:
    - `markets.status=RESOLVED`
    - order became `CANCELLED` with `cancel_reason=MARKET_RESOLVED`
    - freeze became `RELEASED` with `remaining_amount=0`
    - `account_balances` for `user_id=1002 asset=USDT` returned to baseline `available=1013360`, `frozen=0`
    - `trades` for `market_id=220140402` stayed empty
    - active `NEW/PARTIALLY_FILLED` orders for the market stayed empty
  - cold restart result:
    - matching boot log reported `restored_trade_sequence=6 restored_resting_orders=0 book_count=0`
    - post-restart `POST /api/v1/orders` on resolved `market_id=220140402` returned HTTP `409` with `{"error":"market is not tradable"}`
    - order count for the market stayed `1`, trade count stayed `0`, freeze count for `user_id=1002` stayed `4`
- stale pre-fix freezes still need a separate reconciliation task:
  - reused local DB still has historical corruption, for example `user_id=1001 asset=USDT` remains `frozen=5100`
  - an older released row such as `frz_1775036086382_5762c323fb07` still has non-zero `remaining_amount`
  - no historical backfill was attempted in this task

## Blockers

- none for the finality fix itself
- historical stale-freeze cleanup remains out of scope and still needs a separate reconciliation task if local DB correctness matters

## Status

- completed
