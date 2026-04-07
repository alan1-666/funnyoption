# TASK-CHAIN-025

## Summary

Promote manual-resolution markets from a generic post-`close_at` `CLOSED` state to
one explicit runtime-effective `WAITING_RESOLUTION` state, while preserving
`resolve_at` as the automatic oracle resolution timestamp and blocking manual
operator resolution until the market has truthfully reached its adjudication
window.

## Scope

- build directly on `TASK-CHAIN-024`
- keep current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- keep the chosen runtime-effective status model:
  - do not add a background job that persists lifecycle transitions back into
    `markets.status`
  - continue deriving effective status from stored `status + close_at +
    resolve_at + resolution mode`
- add one explicit adjudication-window state:
  - all unresolved markets become runtime `CLOSED` after `close_at`
  - manual-resolution markets become runtime `WAITING_RESOLUTION` only once
    `resolve_at` is reached
  - oracle-eligible markets remain runtime `CLOSED` after `close_at` until the
    oracle lane resolves them at/after `resolve_at`
  - `RESOLVED` still only comes from a real resolve event / settlement path
- tighten operator resolution boundaries:
  - ordinary/manual markets may only be resolved once they are truthfully
    `WAITING_RESOLUTION` at/after `resolve_at`
  - oracle markets must not be manually resolved through the ordinary operator
    path
  - no market may be resolved before its adjudication window
- expose the new status truthfully to:
  - public market reads
  - admin market reads and resolve UI
  - frontend market status labels / cards / detail views
- do not implement:
  - background lifecycle persistence jobs
  - rollup/prover/contract changes
  - fake time-only auto-resolution for non-oracle markets
  - a broad admin refactor beyond the status/resolve surfaces needed here

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/market_lifecycle.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/market_lifecycle.go)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/backend/internal/oracle/service/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/oracle/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/backend/internal/oracle/service/worker.go](/Users/zhangza/code/funnyoption/backend/internal/oracle/service/worker.go)
- [/Users/zhangza/code/funnyoption/web/lib/types.ts](/Users/zhangza/code/funnyoption/web/lib/types.ts)
- [/Users/zhangza/code/funnyoption/web/lib/locale.ts](/Users/zhangza/code/funnyoption/web/lib/locale.ts)
- [/Users/zhangza/code/funnyoption/web/components/admin-market-ops.tsx](/Users/zhangza/code/funnyoption/web/components/admin-market-ops.tsx)

## Owned files

- `internal/api/handler/**`
- `internal/oracle/service/**` only if lifecycle coordination needs a narrow touch
- `web/components/**` only where status labels or resolve surfaces need updates
- `web/lib/types.ts`
- `web/lib/locale.ts`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-025.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-025.md`

## Acceptance criteria

- runtime-effective status becomes:
  - `OPEN` while trading is still open
  - `CLOSED` for all unresolved markets after `close_at` and before their
    adjudication window
  - `WAITING_RESOLUTION` for non-oracle markets once `resolve_at` is reached
    and before manual resolution
  - `RESOLVED` only after a real resolve event lands
- manual operator resolve is rejected for:
  - markets still effectively `OPEN`
  - oracle markets on the ordinary resolve path
  - already `RESOLVED` markets
- admin/public reads can distinguish `WAITING_RESOLUTION` from `CLOSED`
- frontend/admin labels and filters present the new state truthfully

## Validation

- targeted Go tests for runtime-effective `WAITING_RESOLUTION`
- targeted Go tests for resolve-gating on manual vs oracle markets
- `cd web && npm run build`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-024` completed

## Handoff

- return changed files, validation commands, and the chosen lifecycle contract
- call out the before/after behavior for:
  - ordinary-market post-close read status
  - manual resolve before/after adjudication window
  - oracle-market post-close read status
  - oracle-market auto-resolution at `resolve_at`
