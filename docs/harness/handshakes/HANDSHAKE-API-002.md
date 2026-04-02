# HANDSHAKE-API-002

## Task

- [TASK-API-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-002.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `WORKLOG-ADMIN-004.md`
- `WORKLOG-API-001.md`
- this handshake
- `WORKLOG-API-002.md`

## Files in scope

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Inputs from other threads

- shared core API privileged market routes are now operator-authenticated at the backend boundary
- the API router is now modular and rate-limited
- the remaining explicit gap is the transitional bare-`user_id` order-write fallback used by admin bootstrap

## Outputs back to commander

- changed files
- the chosen authenticated bootstrap-order path
- authorized and unauthorized validation notes for `/api/v1/orders`
- any remaining deeper order-ingress gaps

## Blockers

- do not widen into unrelated matching or settlement changes
- do not re-open the deprecated Go admin runtime or bypass the dedicated Next admin service

## Status

- completed

## Handoff notes

- chosen authenticated bootstrap-order path:
  - a narrow privileged operator-proof order lane on `POST /api/v1/orders`
  - the dedicated Next admin service now forwards the same signed `ISSUE_FIRST_LIQUIDITY` bootstrap proof into both:
    - `POST /api/v1/admin/markets/:market_id/first-liquidity`
    - `POST /api/v1/orders`
  - this was chosen over building a synthetic bootstrap actor session because operator-proof verification already exists end-to-end in this repo, while a real bootstrap session would widen into extra session-state and lifecycle work outside this task
- `/api/v1/orders` protection now:
  - trade-write middleware rejects requests unless they contain either:
    - a complete wallet-session envelope, or
    - a complete `operator` proof envelope
  - `CreateOrder` also rejects bare unauthenticated `user_id` writes directly, so the fallback is removed at the handler boundary too
  - operator-authenticated orders are intentionally narrow:
    - sell
    - limit
    - GTC
    - bootstrap payload must match the signed operator draft
- changed files:
  - `internal/api/middleware.go`
  - `internal/api/routes_auth.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/api/dto/order.go`
  - `internal/api/dto/operator_auth.go`
  - `internal/api/router_test.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
  - `docs/harness/worklogs/WORKLOG-API-002.md`
  - `docs/harness/handshakes/HANDSHAKE-API-002.md`
- validation notes:
  - `go test ./internal/api/...`
  - `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - focused unauthorized proof:
    - `TestEngineTradeWriteRejectsBareUserIDWithoutAuthEnvelope` returns `401`
  - focused authorized proof:
    - `TestEngineTradeWriteSupportsOperatorBootstrapOrder`
    - `TestCreateOrderWithOperatorBootstrapProofPublishesCommand`
  - runtime bootstrap proof:
    - temporary mock core API on `http://127.0.0.1:8086`
    - temporary admin dev server on `http://127.0.0.1:3016`
    - signed `POST /api/operator/markets/88/first-liquidity` returned `202`
    - response included `first_liquidity_id=liq_mock_1`, `order_id=ord_mock_1`, `order_status=QUEUED`
    - mock log confirmed the admin route forwarded `/api/v1/orders` with `requested_at` plus the full `operator` proof envelope
- remaining deeper order-ingress gaps:
  - operator bootstrap orders now have an explicit authenticated lane, but that proof model still lacks session-style per-order nonce/replay protection
