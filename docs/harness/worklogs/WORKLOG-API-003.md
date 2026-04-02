# WORKLOG-API-003

### 2026-04-02 16:05 Asia/Shanghai

- read:
  - `PLAN-2026-04-01-master.md`
  - `WORKLOG-API-002.md`
  - `internal/api/middleware.go`
  - `internal/api/handler/order_handler.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
- changed:
  - created a follow-up task to add replay/idempotency protection to privileged bootstrap orders
- validated:
  - `TASK-API-002` removed the bare `user_id` fallback and moved bootstrap order placement onto an authenticated operator-proof lane
  - that privileged lane still relies on a short signature window without session-style nonce or explicit idempotency semantics
- blockers:
  - none yet; worker should choose the narrowest replay/idempotency model that keeps admin bootstrap working
- next:
  - launch `TASK-API-003`

### 2026-04-02 16:24 Asia/Shanghai

- read:
  - `internal/api/dto/operator_auth.go`
  - `internal/api/handler/sql_store.go`
  - `internal/account/service/balance_book.go`
  - `internal/account/service/sql_store.go`
  - `internal/matching/engine/engine.go`
  - `internal/api/router_test.go`
- changed:
  - chose a narrower bootstrap idempotency key derived from the already signed bootstrap payload (`wallet_address`, `market_id`, `user_id`, `quantity`, `outcome`, `price`, `requested_at`) instead of adding an explicit bootstrap nonce, because the repo already signs and forwards these fields end-to-end and that kept the change inside API order ingress without widening the admin contract or session model
  - derived a deterministic bootstrap `order_id` from that key and added privileged bootstrap replay checks in `CreateOrder` against persisted `orders` and the latest `freeze_records` entry for the same `ref_type=ORDER` / `ref_id=order_id`
  - added a keyed in-handler replay gate so concurrent duplicate bootstrap submissions on the same API instance serialize before the persisted replay check runs
  - extended API SQL/test stores with `GetOrder` and `GetLatestFreezeByRef`; left the admin bootstrap route source unchanged because it already forwards the signed fields the derived key depends on
- validated:
  - `gofmt -w /Users/zhangza/code/funnyoption/internal/api/dto/operator_auth.go /Users/zhangza/code/funnyoption/internal/api/handler/bootstrap_replay.go /Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go /Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go /Users/zhangza/code/funnyoption/internal/api/handler/order_handler_test.go /Users/zhangza/code/funnyoption/internal/api/router_test.go`
  - `go test ./internal/api/...`
  - `go test ./internal/api/... -run 'Test(CreateOrderWithOperatorBootstrapProofPublishesCommand|CreateOrderRejectsReplayedOperatorBootstrapOrder|CreateOrderWithSessionSignaturePublishesCommand|EngineTradeWriteRejectsReplayedOperatorBootstrapOrder)$'`
  - `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - focused proof:
    - first privileged bootstrap order still succeeds in `TestCreateOrderWithOperatorBootstrapProofPublishesCommand`
    - replayed privileged bootstrap order now returns `409` before pre-freeze in `TestCreateOrderRejectsReplayedOperatorBootstrapOrder` and route-level `TestEngineTradeWriteRejectsReplayedOperatorBootstrapOrder`
    - normal session-backed orders still succeed in `TestCreateOrderWithSessionSignaturePublishesCommand`
- blockers:
  - none at this task boundary
- next:
  - commander can treat the derived bootstrap `order_id` as the authoritative uniqueness handle for replayed bootstrap orders
  - if ingress hardening continues later, decide whether an operator should be able to intentionally authorize a second otherwise-identical bootstrap order by signing the same payload again with a new `requested_at`
