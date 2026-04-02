# TASK-OFFCHAIN-004

## Summary

Restore resolved-market finality so a market that is already `RESOLVED` cannot accept new orders, cannot keep active resting liquidity, and cannot rehydrate stale resting orders on cold start.

## Scope

- reject new order ingress when the target market is not tradable
- ensure market resolution clears or neutralizes open resting orders so settlement is terminal
- prevent matching cold-start restore from rehydrating orders for non-tradable markets
- verify freeze release behavior remains correct when post-resolution resting orders are cancelled
- keep historical stale-freeze backfill out of scope unless it is required to land the finality fix safely

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-004.md)

## Owned files

- `internal/api/handler/**`
- `internal/matching/engine/**`
- `internal/matching/service/**`
- `internal/account/service/**`
- `internal/settlement/service/**`
- `README.md`

## Acceptance criteria

- `POST /api/v1/orders` against a `RESOLVED` market is rejected and does not leave a new freeze behind
- resolving a market transitions any remaining resting orders for that market into a terminal non-active state and releases their remaining freezes through the normal event flow
- cold-start restore does not rehydrate resting orders whose market is already non-tradable
- local regression proves that after `POST /api/v1/markets/:id/resolve`, a repeated cold restart plus another order attempt produces no new trade and no active depth for the resolved market
- worker records the exact commands, observed HTTP status, and pass/fail matrix back into the worklog

## Validation

- `go test ./internal/api/...`
- `go test ./internal/matching/...`
- `go test ./internal/account/service`
- `go test ./internal/settlement/...`
- targeted local regression for resolve -> restart -> re-order rejection

## Dependencies

- `TASK-OFFCHAIN-002` findings are the input; do not reopen unrelated read-surface cleanup in this task

## Handoff

- return whether resolved-market finality is fully restored
- explicitly note whether historical stale freezes still require a separate reconciliation task
