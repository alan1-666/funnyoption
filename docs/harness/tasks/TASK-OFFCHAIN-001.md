# TASK-OFFCHAIN-001

## Summary

Close out the off-chain MVP loop: homepage, market detail, order entry, matching, settlement, live quote surfaces, and operator-facing visibility.

## Scope

- order -> matching -> account -> ledger -> settlement loop
- realtime `depth / ticker / candle / market` websocket surfaces
- local dev stack stability
- detail-page read model quality

## Inputs to read

- [`/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md`](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [`/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md`](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- [`/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md`](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [`/Users/zhangza/code/funnyoption/docs/sql/schema.md`](/Users/zhangza/code/funnyoption/docs/sql/schema.md)

## Owned files

- `internal/api/**`
- `internal/account/**`
- `internal/matching/**`
- `internal/ledger/**`
- `internal/settlement/**`
- `internal/ws/**`
- `web/**`

## Acceptance criteria

- local stack can demonstrate order, fill, settlement, and live detail updates
- account and ledger semantics are internally consistent
- detail page shows live depth, ticker, and candle data

## Validation

- local API and WS verification
- trade lifecycle checks
- frontend build and interactive smoke tests

## Dependencies

- harness framework can run in parallel

## Handoff

- this is an umbrella lane only; execution should happen via narrower child tasks
- current first child task: [`TASK-OFFCHAIN-002.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md)
