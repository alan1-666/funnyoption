# HANDSHAKE-CHAIN-025

## Task

- [TASK-CHAIN-025.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-025.md)

## Thread owner

- chain/runtime worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-024.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-024.md`
- `internal/api/handler/market_lifecycle.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/oracle/service/sql_store.go`
- `internal/oracle/service/worker.go`
- `web/lib/types.ts`
- `web/lib/locale.ts`
- `web/components/admin-market-ops.tsx`
- this handshake
- `WORKLOG-CHAIN-025.md`

## Files in scope

- `internal/api/handler/**`
- `internal/oracle/service/**` only if lifecycle coordination needs a narrow touch
- `web/components/**` only where status/resolve reads need updates
- `web/lib/types.ts`
- `web/lib/locale.ts`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-025.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-025.md`

## Inputs from other threads

- `TASK-CHAIN-024` landed:
  - `close_at` is already enforced as the runtime trading boundary
  - expired unresolved markets already stop trading and read back truthfully
    instead of remaining implicitly tradable
- current residual product gap after `TASK-CHAIN-024`:
  - ordinary/manual markets still read as generic `CLOSED` after `close_at`
  - operator resolve path is not yet restricted to a dedicated
    post-`resolve_at` waiting-for-adjudication state
  - product/admin surfaces cannot distinguish “waiting for oracle” from
    “waiting for manual ruling”

## Outputs back to commander

- changed files
- chosen lifecycle contract
- validation commands
- before/after behavior for manual vs oracle post-close states
- residual limitations

## Handoff notes

- keep runtime-effective derivation; do not add a background state-persistence job
- unresolved markets should stay runtime `CLOSED` after `close_at`
- ordinary/manual markets should become `WAITING_RESOLUTION` only once
  `resolve_at` is reached
- oracle markets should remain `CLOSED` after `close_at` until the oracle lane
  reaches `RESOLVED`
- ordinary operator resolve must only work for `WAITING_RESOLUTION`
- do not widen into rollup/prover/contract work

## Blockers

- do not claim the product is already Mode B
- do not add time-only auto-resolution for non-oracle markets
- do not reopen the current oracle `resolve_at` contract

## Status

- completed
