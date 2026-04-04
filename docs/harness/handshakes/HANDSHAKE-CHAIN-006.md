# HANDSHAKE-CHAIN-006

## Task

- [TASK-CHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-006.md)

## Thread owner

- chain/oracle implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/architecture/order-flow.md`
- `docs/sql/schema.md`
- `foundry.toml`
- `admin/app/api/operator/markets/route.ts`
- `admin/app/api/operator/markets/[marketId]/resolve/route.ts`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/settlement/service/sql_store.go`
- this handshake
- `WORKLOG-CHAIN-006.md`

## Files in scope

- `internal/api/**` only where market metadata validation / manual resolve guard
  needs narrow updates
- `internal/settlement/**` only where resolution ownership truthfulness needs a
  narrow fix
- one new oracle worker / service package in a narrow boundary
- `cmd/oracle/**` for the dedicated worker entrypoint only
- `admin/lib/operator-auth.ts`
- `admin/app/api/operator/markets/route.ts`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-006.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-006.md`
- narrow docs / schema notes only if implementation truth diverges from the
  design
- Foundry-side files only if a truly necessary contract placeholder appears

## Inputs from other threads

- `TASK-CHAIN-005` is complete and fixed the contract:
  - metadata lives in `markets.metadata.resolution`
  - first cut reuses `market_resolutions`
  - dedicated oracle worker is the chosen resolver boundary
  - manual operator resolve stays fallback only before `OBSERVED / RESOLVED`
- commander review found one required implementation truthfulness guard:
  - if manual fallback wins after oracle error states, the final
    `market_resolutions` row must not retain stale `ORACLE_PRICE` ownership
    metadata from a previous failed observation attempt
- if any future chain-side helper is needed, it must stay on Foundry; this
  first slice should not require one

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary of:
  - oracle resolution success path
  - retryable / terminal error handling
  - manual resolve conflict guard

## Blockers

- keep manual operator resolve as fallback / override, not primary path
- do not widen into multi-provider arbitration
- do not introduce a second Solidity toolchain
- do not break current non-oracle market semantics
- no open blockers in this slice after the duplicate-emit guard landed

## Status

- completed

## Handoff notes back to commander

- runtime slice is landed without widening beyond single-provider
  `BINANCE + HTTP_JSON + LAST_PRICE`
- added a standalone oracle worker under `cmd/oracle` and `internal/oracle/service`
  that scans eligible markets, writes `market_resolutions`, and republishes the
  existing `market.event`
- follow-up duplicate-emit guard is now in place:
  - once the same oracle observation is already recorded as `OBSERVED`, the
    worker skips instead of republishes
  - repeated polling therefore no longer re-triggers settlement/account side
    effects before the row advances to `RESOLVED`
- manual resolve now rejects oracle markets once resolution is already
  `OBSERVED / RESOLVED`, and settlement now overwrites stale oracle ownership
  when manual fallback wins from earlier error states
- admin create signing now covers `metadata.resolution` so oracle metadata
  cannot be changed after operator authorization
