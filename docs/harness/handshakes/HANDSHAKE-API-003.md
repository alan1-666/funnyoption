# HANDSHAKE-API-003

## Task

- [TASK-API-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-003.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `WORKLOG-API-002.md`
- this handshake
- `WORKLOG-API-003.md`

## Files in scope

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Inputs from other threads

- `/api/v1/orders` no longer accepts bare unauthenticated `user_id` writes
- privileged bootstrap orders now use an explicit operator-proof lane
- the remaining explicit gap is replay/idempotency for that privileged bootstrap-order lane

## Outputs back to commander

- changed files
- the chosen anti-replay/idempotency model
- authorized and replayed validation notes
- any remaining deeper order-ingress gaps

## Blockers

- do not widen into general session nonce redesign for all user orders
- do not reintroduce the removed bare-`user_id` fallback

## Status

- completed

## Handoff notes

- changed files:
  - `internal/api/dto/operator_auth.go`
  - `internal/api/handler/bootstrap_replay.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/sql_store.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/api/router_test.go`
  - `docs/harness/worklogs/WORKLOG-API-003.md`
  - `docs/harness/handshakes/HANDSHAKE-API-003.md`
- chosen anti-replay/idempotency model:
  - derive a bootstrap replay key from the signed bootstrap payload (`wallet_address`, `market_id`, `user_id`, `quantity`, `outcome`, `price`, `requested_at`), hash it into a deterministic bootstrap `order_id`, serialize same-key requests in-process, then reject replays when that `order_id` already exists in `orders` or the latest `freeze_records` row for `ref_type=ORDER` / `ref_id=order_id` is not `RELEASED`
- authoritative uniqueness key:
  - `dto.CreateOrderRequest.BootstrapReplayKey()`
  - hashed transport handle: `dto.CreateOrderRequest.BootstrapOrderID()`
- validation:
  - `go test ./internal/api/...`
  - `go test ./internal/api/... -run 'Test(CreateOrderWithOperatorBootstrapProofPublishesCommand|CreateOrderRejectsReplayedOperatorBootstrapOrder|CreateOrderWithSessionSignaturePublishesCommand|EngineTradeWriteRejectsReplayedOperatorBootstrapOrder)$'`
  - `cd /Users/zhangza/code/funnyoption/admin && npm run build`
- remaining deeper gap:
  - this task blocks replay/retry of the same signed bootstrap payload, but an operator can still intentionally authorize a second otherwise-identical bootstrap order by issuing a new proof with a different `requested_at`; that broader market-bootstrap policy was kept out of scope here
