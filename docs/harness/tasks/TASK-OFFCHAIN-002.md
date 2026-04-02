# TASK-OFFCHAIN-002

## Summary

Close the local off-chain regression path for the current MVP: homepage, market detail, order entry, matching, settlement, and live candle/detail updates.

## Scope

- verify local dev stack stability for repeated `dev-up / dev-down`
- verify and fix order -> matching -> account -> ledger -> settlement consistency
- verify market detail `depth / ticker / candle / market` realtime surfaces
- produce one reproducible local regression flow with seeded markets and users

## Inputs to read

- [`/Users/zhangza/code/funnyoption/AGENTS.md`](/Users/zhangza/code/funnyoption/AGENTS.md)
- [`/Users/zhangza/code/funnyoption/PLAN.md`](/Users/zhangza/code/funnyoption/PLAN.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md`](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md`](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md`](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [`/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md`](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [`/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md`](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- [`/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md`](/Users/zhangza/code/funnyoption/docs/topics/kafka-topics.md)
- [`/Users/zhangza/code/funnyoption/docs/sql/schema.md`](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md`](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md)
- [`/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md`](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md)

## Owned files

- `scripts/dev-up.sh`
- `scripts/dev-down.sh`
- `scripts/dev-status.sh`
- `internal/api/**`
- `internal/account/**`
- `internal/matching/**`
- `internal/ledger/**`
- `internal/settlement/**`
- `internal/ws/**`
- `web/app/markets/**`
- `web/components/live-market-panel*`
- `web/components/order-ticket*`
- `README.md`

## Acceptance criteria

- local stack starts and stops reliably via scripts
- one seeded local market can demonstrate:
  - homepage visible
  - detail page visible
  - at least one filled trade
  - balances and positions updated consistently
  - market resolution triggers settlement
  - detail page websocket streams show depth, ticker, and candle updates
- worker records the exact local verification steps in the worklog

## Validation

- `go test ./...`
- `npm run build`
- local API checks
- local browser or websocket smoke checks

## Dependencies

- none

## Handoff

- return with a concise pass/fail matrix for:
  - homepage
  - detail page
  - matching
  - settlement
  - candle push
- propose the next read-surface cleanup task only after this is stable
