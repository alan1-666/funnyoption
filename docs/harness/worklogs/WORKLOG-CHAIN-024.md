# WORKLOG-CHAIN-024

### 2026-04-06 00:51 CST

- read:
  - `PLAN.md`
  - `PLAN-2026-04-01-master.md`
  - `docs/architecture/order-flow.md`
  - `docs/architecture/oracle-settled-crypto-markets.md`
  - `docs/sql/schema.md`
  - `internal/api/handler/order_handler.go`
  - `internal/matching/service/sql_store.go`
  - `internal/oracle/service/sql_store.go`
- changed:
  - created the market-expiry lifecycle hardening task, handshake, and worklog
- validated:
  - commander review confirmed the current runtime gap:
    - order ingress only trusts `market.status == OPEN`
    - matching restore only trusts `markets.status == OPEN`
    - ordinary markets can therefore remain tradable past `close_at`
  - commander review also confirmed the current oracle lane remains explicit:
    - oracle markets auto-resolve from `resolve_at`
    - non-oracle markets still require manual resolution
- blockers:
  - none yet
- next:
  - launch one worker on `TASK-CHAIN-024`

### 2026-04-06 01:05 CST

- read:
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-007.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-007.md`
  - `internal/api/handler/order_handler_test.go`
  - `internal/matching/service/consumer_test.go`
  - `internal/oracle/service/worker_test.go`
- changed:
  - added one shared handler-side lifecycle helper so runtime-effective market
    status becomes `CLOSED` once stored `OPEN` rows pass `close_at`
  - gated `CreateOrder` and `CreateFirstLiquidity` on that effective
    post-`close_at` trading boundary instead of raw `status == OPEN`
  - made `GetMarket` / `ListMarkets` surface the same runtime-effective status,
    including truthful `CLOSED` readback for expired unresolved markets and
    truthful `OPEN` / `CLOSED` list filters
  - hardened matching restore + `MarketIsTradable(...)` with the same
    `close_at` gate so restarted matching workers no longer reload expired
    resting orders as tradable
  - documented that `resolve_at` remains the oracle settlement timestamp while
    non-oracle markets stay closed-awaiting-manual-resolution
- validated:
  - targeted coverage now exists for:
    - order rejection after `close_at`
    - first-liquidity rejection after `close_at`
    - runtime-effective `OPEN -> CLOSED` status derivation at the `close_at`
      boundary
    - matching tradability rejection at the `close_at` boundary
    - oracle worker regression staying anchored to `resolve_at`
  - validation commands:
    - `gofmt -w internal/api/handler/market_lifecycle.go internal/api/handler/market_lifecycle_test.go internal/api/handler/order_handler.go internal/api/handler/order_handler_test.go internal/api/handler/sql_store.go internal/matching/service/market_lifecycle.go internal/matching/service/market_lifecycle_test.go internal/matching/service/sql_store.go`
    - `go test ./internal/api/handler ./internal/matching/service ./internal/oracle/service ./internal/settlement/service`
    - `git diff --check`
- blockers:
  - none
- next:
  - hand back the chosen lifecycle contract, changed files, validation
    commands, and residual limitations to commander
