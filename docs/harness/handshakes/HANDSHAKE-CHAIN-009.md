# HANDSHAKE-CHAIN-009

## Task

- [TASK-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-009.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-008.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-008.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-008.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/architecture/order-flow.md`
- `docs/sql/schema.md`
- `internal/api/handler/order_handler.go`
- `internal/matching/**`
- `internal/account/**`
- `internal/settlement/**`
- `internal/chain/**`
- this handshake
- `WORKLOG-CHAIN-009.md`

## Files in scope

- `internal/rollup/**`
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
- narrow touchpoints into existing services only where needed to source shadow
  inputs

## Inputs from other threads

- `TASK-CHAIN-008` closed the architecture contract:
  - `ZK-Rollup` DA only
  - slow / fast / forced withdrawal model
  - sequencer journal and batch input are mandatory future truths
- commander wants the first implementation slice to stay shadow-only:
  - no prover
  - no verifier
  - no production withdrawal-claim rewrite
- current product truth must remain on the existing SQL/Kafka path while this
  slice lands

## Outputs back to commander

- changed files
- new shadow-rollup storage/runtime artifacts
- validation commands
- residual shadow-only limitations
- recommended next prover / L1 tranche

## Handoff notes

- this slice should make the replay contract real before any proof system work
- dedicated rollup artifacts are preferred over mutating current mutable tables
- if a namespace cannot be truthfully populated yet, it must be explicitly
  defaulted and documented
- landed storage/runtime boundary:
  - `migrations/014_rollup_shadow_lane.sql`
  - `internal/rollup/**`
  - `internal/matching/service/sql_store.go`
  - `internal/matching/service/rollup_shadow.go`
  - `internal/chain/service/processor.go`
- landed captured shadow inputs:
  - `ORDER_ACCEPTED`
  - `ORDER_CANCELLED`
  - `TRADE_MATCHED`
  - `DEPOSIT_CREDITED`
  - `WITHDRAWAL_REQUESTED`
- current intentional shadow-only limits:
  - no prover / verifier
  - no L1 state update
  - no production withdrawal claim rewrite
  - no forced withdrawal / freeze runtime
  - no market-resolution / settlement-payout shadow inputs yet
  - `orders_root` still uses deterministic `ZeroNonceRoot()`; replay-protection
    state is not yet shadowed truthfully
  - `positions_funding_root` uses deterministic zero roots for
    `market_funding_root` and `insurance_root`

## Blockers

- do not claim the product is already Mode B after this slice
- do not make SQL snapshots or Kafka offsets implicit replay truth
- do not widen into prover, verifier, or forced-withdrawal implementation
- do not introduce another contract toolchain

## Status

- completed
