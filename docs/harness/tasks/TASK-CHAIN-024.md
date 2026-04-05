# TASK-CHAIN-024

## Summary

Harden market-expiry lifecycle semantics so `close_at` truthfully stops trading
for all markets, `resolve_at` stays the canonical settlement timestamp for
oracle-eligible markets, and ordinary markets no longer implicitly remain
tradable or "auto-settle" just because wall clock time advanced.

## Scope

- build on the current oracle runtime from `TASK-CHAIN-005` through
  `TASK-CHAIN-007`
- keep current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- fix the current runtime hole:
  - order ingress treats `market.status == OPEN` as sufficient
  - matching restore treats `markets.status == OPEN` as sufficient
  - `close_at` is metadata today, not a hard trading boundary
- choose the narrowest truthful implementation:
  - enforce `close_at` in order ingress and matching runtime
  - add one durable close-time contract so expired markets do not rely on a
    stale `OPEN` row forever
  - preserve `resolve_at` as the canonical settlement timestamp for
    oracle-settled markets
  - keep non-oracle markets closed and awaiting manual resolution after
    `close_at`; do not invent time-only auto-settlement without a resolver
  - if resting orders remain after close, define one explicit cancellation
    contract/reason or equivalent truthful invariant
  - expose truthful read/runtime semantics so clients can distinguish
    closed-unresolved from resolved
- do not implement:
  - rollup/prover changes
  - new on-chain contracts
  - multi-provider oracle arbitration
  - a fake auto-resolution path for markets with no resolver contract

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-007.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-007.md)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/matching/service/sql_store.go](/Users/zhangza/code/funnyoption/internal/matching/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/matching/service/consumer.go](/Users/zhangza/code/funnyoption/internal/matching/service/consumer.go)
- [/Users/zhangza/code/funnyoption/internal/oracle/service/sql_store.go](/Users/zhangza/code/funnyoption/internal/oracle/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go](/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go)
- [/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go](/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go)

## Owned files

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

## Acceptance criteria

- no new orders are accepted after `close_at`, even if a stale market row still
  says `OPEN`
- matching does not continue to restore or execute resting orders for a market
  that is past `close_at`
- oracle markets continue to auto-resolve at `resolve_at`
- non-oracle markets become truthfully closed-awaiting-resolution after
  `close_at` and do not pretend that time alone has settled them
- validation includes:
  - targeted Go tests for post-`close_at` order rejection
  - one proof that matching/runtime no longer treats a past-`close_at` market
    as tradable
  - one proof that the oracle `resolve_at` lane still works

## Validation

- targeted Go tests for ingress and matching lifecycle gating
- targeted Go tests for any chosen close-time cancellation / runtime marker
- one oracle-lane regression proof
- `git diff --check`

## Dependencies

- `TASK-CHAIN-007` runtime baseline is complete

## Handoff

- return changed files, validation commands, and the chosen lifecycle contract
- call out the before/after behavior for:
  - order placement after `close_at`
  - resting orders after `close_at`
  - ordinary-market post-close state
  - oracle-market `resolve_at` behavior
