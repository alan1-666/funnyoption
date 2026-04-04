# TASK-CHAIN-006

## Summary

Implement the first oracle-settled crypto market runtime slice: validate
oracle-resolution metadata, run one dedicated oracle worker, emit the existing
resolution event path, and harden manual fallback semantics.

## Scope

- implement the first runtime slice from
  `docs/architecture/oracle-settled-crypto-markets.md`
- support only the first-cut contract:
  - `category_key = CRYPTO`
  - binary `YES / NO`
  - `metadata.resolution.mode = ORACLE_PRICE`
  - `HTTP_JSON`
  - one provider key
  - one source kind / one price field
- add market create / validation support for `metadata.resolution`
- add one dedicated oracle worker that:
  - scans eligible markets by `resolve_at`
  - fetches and normalizes one provider price
  - computes `YES / NO`
  - writes `market_resolutions` as `OBSERVED`
  - publishes the existing `market.event` shape
- keep the first cut narrow:
  - no new SQL table
  - no new Kafka evidence topic
  - no on-chain helper contract
  - no multi-provider arbitration
- harden manual fallback semantics:
  - reject manual resolve when an oracle market is already `OBSERVED` or
    `RESOLVED`
  - when a manual fallback wins from `PENDING / RETRYABLE_ERROR / TERMINAL_ERROR`
    states, the final `market_resolutions` row must truthfully reflect operator
    ownership instead of preserving stale oracle ownership fields

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/foundry.toml](/Users/zhangza/code/funnyoption/foundry.toml)
- [/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/route.ts](/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/route.ts)
- [/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/[marketId]/resolve/route.ts](/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/[marketId]/resolve/route.ts)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/settlement/service/sql_store.go](/Users/zhangza/code/funnyoption/internal/settlement/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-006.md)

## Owned files

- `internal/api/**` only where market metadata validation / manual resolve guard
  needs narrow updates
- `internal/settlement/**` only where resolution ownership truthfulness needs a
  narrow fix
- one new oracle worker / service package in a narrow boundary
- `docs/harness/handshakes/HANDSHAKE-CHAIN-006.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-006.md`
- narrow docs / schema notes only if implementation truth diverges from the
  design
- Foundry-side files only if a truly necessary contract placeholder appears; do
  not introduce a second Solidity framework

## Acceptance criteria

- market creation / validation rejects malformed `metadata.resolution` for the
  supported first-cut oracle lane
- one dedicated oracle worker can resolve at least one supported market by:
  - writing `market_resolutions` with `resolver_type = ORACLE_PRICE`
  - recording canonical evidence
  - emitting the existing `market.event`
- manual resolve is rejected when the oracle market is already `OBSERVED` or
  `RESOLVED`
- manual fallback from earlier error states truthfully overwrites the final
  resolution record as operator-owned
- validation includes:
  - targeted Go tests
  - one local or staged proof of an oracle-resolved market
  - one proof of the manual conflict guard

## Validation

- targeted Go tests for market validation, oracle resolution, and manual guard
- one local or staged proof for:
  - successful oracle resolution
  - retryable or terminal error behavior
  - rejected manual resolve after `OBSERVED`

## Dependencies

- `TASK-CHAIN-005` design contract is now the baseline

## Handoff

- return changed files, validation commands, and one clear flow summary
- call out residual tradeoffs such as single-provider limits or latest-row-only
  audit depth
