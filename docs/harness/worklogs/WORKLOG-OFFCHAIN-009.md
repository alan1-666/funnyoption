# WORKLOG-OFFCHAIN-009

### 2026-04-02 00:40 Asia/Shanghai

- read:
  - `PLAN-2026-04-01-master.md`
  - `docs/operations/local-offchain-lifecycle.md`
  - `WORKLOG-ADMIN-001.md`
- changed:
  - created a follow-up off-chain task for explicit first-liquidity bootstrap on fresh markets
- validated:
  - the task isolates the remaining fresh-market tradability gap from deposit-listener and admin-auth work
- blockers:
  - none yet
- next:
  - start after `TASK-ADMIN-002`

### 2026-04-02 02:03 Asia/Shanghai

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `TASK-ADMIN-002.md`
  - `TASK-OFFCHAIN-009.md`
- changed:
  - resequenced first-liquidity work so it lands after admin-service extraction
  - updated task language to target the dedicated admin service instead of the transitional public-web admin shell
- validated:
  - current plan, task, and handshake now agree that operator-only bootstrap UX should not be added to the public web shell as the long-term home
- blockers:
  - none yet; this task is intentionally blocked on the service-boundary decision landing first
- next:
  - start after `TASK-ADMIN-002`

### 2026-04-02 12:55 Asia/Shanghai

- read:
  - `WORKLOG-ADMIN-001.md`
  - `WORKLOG-ADMIN-002.md`
  - `web/app/admin/page.tsx`
  - `web/components/admin-market-ops.tsx`
  - `web/components/market-studio.tsx`
  - `internal/api/handler/order_handler.go`
  - `internal/api/dto/order.go`
  - `cmd/local-lifecycle/main.go`
  - `cmd/local-lifecycle/proof_env.go`
- changed:
  - added `POST /api/v1/admin/markets/:market_id/first-liquidity` so the repo now has an explicit paired-inventory issuance path instead of hidden lifecycle-only position seeding
  - added rollback logic around first-liquidity collateral / inventory issuance so balance and position reads stay aligned if publish fails mid-flight
  - updated `cmd/local-lifecycle` to call the explicit first-liquidity endpoint, wait for paired `YES` / `NO` inventory visibility, and then continue with the normal signed sell + buy order path
  - added a narrow dedicated `admin/` service that creates markets and combines explicit first-liquidity issuance with the first sell-order submit
  - updated the local lifecycle runbook and added a small migration note in `web/app/admin/page.tsx` that points bootstrap work toward the dedicated service instead of growing the transitional public-web shell
- validated:
  - `go test ./internal/api/handler ./cmd/local-lifecycle ./admin`
  - `cd /Users/zhangza/code/funnyoption/web && npm run build`
  - lifecycle proof:
    - `set -a; source /Users/zhangza/code/funnyoption/.env.local; set +a; go run ./cmd/local-lifecycle`
    - created market `1775105346947`
    - issued first-liquidity inventory `liq_1775105347023_be6a083ec1e9`
    - queued sell order `ord_1775105347594_493217ef6108`
    - queued buy order `ord_1775105348174_47bee16314b9`
    - matched trade `trd_9`
    - resolved market `YES`
  - dedicated admin service runtime proof:
    - `export FUNNYOPTION_ADMIN_HTTP_ADDR=:3011; set -a; source /Users/zhangza/code/funnyoption/.env.local; set +a; go run ./admin`
    - `GET http://127.0.0.1:3011/healthz` returned `{"service":"admin","status":"ok"}`
    - `GET http://127.0.0.1:3011/` rendered the dedicated admin service page with market-intake and first-liquidity forms
  - dedicated admin service bootstrap proof against a fresh market:
    - `POST http://127.0.0.1:3011/markets` created market `1775105522604`
    - `POST http://127.0.0.1:3011/first-liquidity` issued paired inventory `liq_1775105539094_5b8e1e5c2425` and queued sell order `ord_1775105539152_c669e49d9736`
    - `GET /api/v1/positions?user_id=1002&market_id=1775105522604` returned explicit `YES` and `NO` inventory rows with quantity `40`
    - `GET /api/v1/orders?user_id=1002&market_id=1775105522604` returned one resting `SELL YES` order with `remaining_quantity=40`
    - `GET /api/v1/markets/1775105522604` returned `runtime.active_order_count=1`
- blockers:
  - operator wallet gating for the new dedicated `admin/` runtime is still a follow-up owned by `TASK-ADMIN-002`; the bootstrap flow currently uses direct operator user ids once inside the dedicated service boundary
  - local port `3001` was already occupied by another process during validation, so the new service was verified on `3011` via `FUNNYOPTION_ADMIN_HTTP_ADDR=:3011`
- next:
  - commander can treat hidden first-liquidity seeding as removed from the lifecycle proof
  - the next admin follow-up should graft wallet-gated operator auth onto the new `admin/` runtime instead of back-porting privileged bootstrap UX into `web/app/admin`
  - if this flow needs to become stricter later, add an operator-authenticated market-onboarding command that records who issued the paired inventory, but keep the paired-inventory semantics explicit
