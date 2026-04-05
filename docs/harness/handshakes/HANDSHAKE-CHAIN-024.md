# HANDSHAKE-CHAIN-024

## Task

- [TASK-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-024.md)

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
- `docs/harness/handshakes/HANDSHAKE-CHAIN-007.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-007.md`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/matching/service/sql_store.go`
- `internal/matching/service/consumer.go`
- `internal/oracle/service/sql_store.go`
- `internal/oracle/service/worker.go`
- `internal/settlement/service/processor.go`
- this handshake
- `WORKLOG-CHAIN-024.md`

## Files in scope

- `internal/api/handler/**`
- `internal/matching/service/**`
- `internal/oracle/service/**` only if lifecycle coordination needs a narrow
  touch
- `internal/settlement/**` only if close-time cancellation or truthful
  unresolved-state behavior needs a narrow contract note
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-024.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-024.md`
- `migrations/**` only if the chosen lifecycle contract truly requires a
  durable marker instead of derived runtime logic

## Inputs from other threads

- `TASK-CHAIN-005` through `TASK-CHAIN-007` landed:
  - oracle-settled crypto markets use `resolve_at` as the canonical settlement
    timestamp
  - oracle worker auto-resolves eligible markets
  - retry-safe dispatch after `OBSERVED` is in place
- commander review confirmed one residual product/runtime gap:
  - ordinary markets can still remain tradable after `close_at`
  - order ingress and matching currently trust `market.status == OPEN`
  - there is no truthful runtime contract yet for
    closed-but-unresolved non-oracle markets

## Outputs back to commander

- changed files
- chosen lifecycle contract
- validation commands
- before/after behavior for `close_at` and `resolve_at`
- residual limitations

## Handoff notes

- keep truthful boundaries:
  - oracle markets may auto-resolve at `resolve_at`
  - non-oracle markets must not pretend that wall clock time alone settles them
- prioritize product/runtime correctness over further rollup work in this slice
- choose the narrowest safe implementation; avoid widening into a large market
  status refactor unless it is truly required
- chosen lifecycle contract:
  - runtime-effective market status is derived from stored `status + close_at`
  - `OPEN + now >= close_at` is exposed and enforced as `CLOSED`
  - order ingress, first-liquidity bootstrap, matching tradability checks, and
    matching restore all share that same boundary
  - oracle auto-resolution remains anchored to `resolve_at`
  - non-oracle markets remain `CLOSED` until a manual resolve event lands

## Blockers

- do not claim the product is already Mode B
- do not widen into rollup/prover work
- do not add a fake auto-resolution path for markets without a resolver

## Status

- completed
