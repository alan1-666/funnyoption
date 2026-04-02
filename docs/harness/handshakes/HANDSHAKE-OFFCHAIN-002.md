# HANDSHAKE-OFFCHAIN-002

## Task

- [`TASK-OFFCHAIN-002.md`](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/ledger-service.md`
- `docs/topics/kafka-topics.md`
- `docs/sql/schema.md`
- this handshake
- `WORKLOG-OFFCHAIN-002.md`

## Files in scope

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

## Inputs from other threads

- harness lane is already in place
- this task does not depend on chain hardening

## Outputs back to commander

- changed files
- end-to-end validation notes
- exact blocker if any local path still fails

## Handoff notes back to commander

- worker fixed two cold-start consistency defects:
  - matching now restores `max(sequence_no)` and hydrates resting `NEW/PARTIALLY_FILLED` limit orders from PostgreSQL before consuming Kafka
  - account now preserves restored freeze metadata for passive order updates and releases BUY-side price-improvement surplus instead of leaving it frozen
- local validation executed end-to-end with screenshots, API checks, websocket probes, repeated `dev-down/dev-up`, `go test ./...`, and `web npm run build`
- follow-up `TASK-OFFCHAIN-004` closed the remaining finality blocker:
  - resolved markets now reject new orders before pre-freeze
  - settlement cancels active resting orders with `MARKET_RESOLVED`
  - matching cold start no longer restores tradable resting liquidity for resolved markets
- see `WORKLOG-OFFCHAIN-002.md` for exact commands, observed trade ids / payout ids, and the pass/fail matrix

## Blockers

- no blocker remains on the main local regression path after `TASK-OFFCHAIN-004`
- reused local DB still contains historical pre-fix frozen quote leftovers for older filled BUY orders; new flows are fixed, but old rows are not backfilled

## Status

- completed
