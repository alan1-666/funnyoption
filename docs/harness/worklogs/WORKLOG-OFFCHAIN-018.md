# WORKLOG-OFFCHAIN-018

### 2026-04-06 01:16 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/architecture/order-flow.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
  - `docs/sql/schema.md`
  - `internal/matching/service/**`
  - `internal/matching/engine/**`
  - `internal/api/handler/**`
  - `web/app/markets/[marketId]/page.tsx`
  - `web/components/order-ticket.tsx`
  - `web/components/live-market-panel.tsx`
  - `web/lib/api.ts`
- changed:
  - created the market detail order-visibility and lifecycle-closeout task,
    handshake, and worklog
- validated:
  - commander review confirmed the remaining backend lifecycle tail:
    - post-`close_at` orders are blocked and skipped on restore
    - but already-loaded matcher orders are not proactively cancelled yet
  - commander review confirmed the detail-page UX gap:
    - the page only reads market + trades
    - it does not show connected-user order/fill state
    - the left-side summary repeats data already shown in the chart/header
- blockers:
  - none yet
- next:
  - implement the backend close-time cancellation contract and the detail-page
    UX cleanup in one tranche

### 2026-04-06 02:18 CST

- changed:
  - `internal/matching/model/types.go`
  - `internal/matching/engine/engine.go`
  - `internal/matching/engine/engine_test.go`
  - `internal/matching/service/sql_store.go`
  - `internal/matching/service/server.go`
  - `internal/matching/service/order_expiry.go`
  - `internal/matching/service/order_expiry_test.go`
  - `web/app/markets/[marketId]/page.tsx`
  - `web/app/markets/[marketId]/page.module.css`
  - `web/components/order-ticket.tsx`
  - `web/components/order-ticket.module.css`
  - `web/components/market-order-activity.tsx`
  - `web/components/market-order-activity.module.css`
  - `web/lib/locale.ts`
  - `docs/architecture/order-flow.md`
  - `docs/sql/schema.md`
- implemented:
  - matching now runs a narrow close-time sweep that loads past-`close_at`
    active resting limit orders, removes them from in-memory books, persists
    `CANCELLED/MARKET_CLOSED`, republishes order events for freeze release, and
    republishes depth/ticker updates so read surfaces converge without waiting
    for matcher restart
  - market detail now includes a connected-user “我的订单” panel that polls the
    existing `/api/v1/orders` read model and reacts to local order-submitted
    events to show open, partial, filled, and cancelled state directly on the
    market page
  - duplicated left-side summary blocks on the market detail hero were removed;
    the page now relies on the chart/live panel plus the new order panel and
    existing sidebar timing card
  - order ticket now honors runtime `CLOSED` state in the UI and stops offering
    fresh submissions once the market is no longer tradable
- validated:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/matching/... ./internal/api/handler ./internal/oracle/service ./internal/settlement/service`
  - `cd web && npm run build`
  - `git diff --check`
- residual limitations:
  - close-time cancellation is sweep-driven, so a still-running matcher cancels
    expired resting orders on the next sweep tick rather than persisting a
    background market-state transition into `markets.status`
  - ordinary markets still do not auto-resolve from time alone; they close at
    `close_at` and await explicit resolution, while oracle markets continue to
    auto-resolve at `resolve_at`
- next:
  - push this tranche and verify the updated market-detail / lifecycle behavior
    on staging
