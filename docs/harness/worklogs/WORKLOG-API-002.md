# WORKLOG-API-002

### 2026-04-02 15:40 Asia/Shanghai

- read:
  - `PLAN-2026-04-01-master.md`
  - `WORKLOG-ADMIN-004.md`
  - `WORKLOG-API-001.md`
  - `internal/api/middleware.go`
  - `internal/api/routes_auth.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
- changed:
  - created a follow-up task to remove the transitional bare-`user_id` order-write lane
- validated:
  - `TASK-ADMIN-004` hardened create/resolve/first-liquidity at the shared API boundary
  - `TASK-API-001` made the legacy trade-write fallback explicit in middleware and router structure
  - the remaining bootstrap caller still forwards a direct `/api/v1/orders` request with only `user_id` and order fields after first-liquidity issuance
- blockers:
  - none yet; worker should choose the narrowest authenticated replacement path that preserves admin bootstrap without keeping the fallback
- next:
  - launch `TASK-API-002`

### 2026-04-02 15:53 Asia/Shanghai

- read:
  - `internal/api/middleware.go`
  - `internal/api/routes_auth.go`
  - `internal/api/handler/order_handler.go`
  - `internal/api/handler/order_handler_test.go`
  - `internal/api/dto/order.go`
  - `internal/api/dto/operator_auth.go`
  - `internal/api/router_test.go`
  - `admin/app/api/operator/markets/[marketId]/first-liquidity/route.ts`
- changed:
  - removed the shared API's bare `user_id` fallback at both the trade-write middleware boundary and `CreateOrder`, so `POST /api/v1/orders` now requires either wallet-session fields or an explicit operator proof envelope
  - chose the narrow privileged bootstrap-order lane instead of inventing a bootstrap actor session because the repo already has allowlisted operator-proof verification in both the Next admin service and the Go API, while a real bootstrap session would widen into new session lifecycle/state outside this task's boundary
  - reused the existing `ISSUE_FIRST_LIQUIDITY` signed draft as the authenticated proof for the paired first sell order, which keeps the bootstrap action bound to one explicit operator-approved payload instead of reintroducing a silent fallback
  - updated the dedicated admin first-liquidity route to forward `requested_at` and `operator` into `/api/v1/orders`
  - added focused route/handler tests for:
    - operator-authenticated bootstrap order success
    - bare `POST /api/v1/orders` rejection without session or operator proof
- validated:
  - `gofmt -w /Users/zhangza/code/funnyoption/internal/api/dto/order.go /Users/zhangza/code/funnyoption/internal/api/dto/operator_auth.go /Users/zhangza/code/funnyoption/internal/api/middleware.go /Users/zhangza/code/funnyoption/internal/api/routes_auth.go /Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go /Users/zhangza/code/funnyoption/internal/api/handler/order_handler_test.go /Users/zhangza/code/funnyoption/internal/api/router_test.go`
  - `go test ./internal/api/...`
  - `cd /Users/zhangza/code/funnyoption/admin && npm run build`
  - focused proof:
    - `TestEngineTradeWriteRejectsBareUserIDWithoutAuthEnvelope` now returns `401` for a direct bare `POST /api/v1/orders`
    - `TestEngineTradeWriteSupportsOperatorBootstrapOrder` and `TestCreateOrderWithOperatorBootstrapProofPublishesCommand` cover the authenticated bootstrap sell-order path that replaced the fallback
  - attempted an extra localhost mock end-to-end validation of the Next admin route, but sandbox policy blocked binding a temporary port; the code/test proof above is complete, and runtime proof can be added later if the commander wants an escalated validation pass
- blockers:
  - no code blocker at this task boundary
  - one deeper residual risk remains explicit:
    - bootstrap orders now use a clear privileged proof lane, but that operator-proof model still has a short replay window and does not yet have session-style per-order nonce semantics
- next:
  - commander can review `TASK-API-002` as implemented
  - if order-ingress hardening continues later, the next follow-up should decide whether privileged bootstrap orders need stronger replay/idempotency protection beyond the current signed operator window

### 2026-04-02 15:57 Asia/Shanghai

- read:
  - `admin/package.json`
- changed:
  - no repo-file change in this validation pass
- validated:
  - temporary mock core API on `http://127.0.0.1:8086`
  - temporary admin dev server on `http://127.0.0.1:3016`
  - signed bootstrap request with allowlisted wallet `0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266`
  - `POST /api/operator/markets/88/first-liquidity` returned:
    - `202`
    - `{"first_liquidity_id":"liq_mock_1","market_id":88,"user_id":1001,"status":"ISSUED","order_id":"ord_mock_1","order_status":"QUEUED","operator_wallet_address":"0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"}`
  - mock core API log proved the admin route now forwards the authenticated order envelope to `/api/v1/orders`, including:
    - `requested_at`
    - `operator.wallet_address`
    - `operator.requested_at`
    - `operator.signature`
- blockers:
  - none
- next:
  - keep the runtime proof details in the commander handoff
