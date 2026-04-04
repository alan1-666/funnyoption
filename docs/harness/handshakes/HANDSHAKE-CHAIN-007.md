# HANDSHAKE-CHAIN-007

## Task

- [TASK-CHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-007.md)

## Thread owner

- chain/oracle reliability worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `HANDSHAKE-CHAIN-006.md`
- `WORKLOG-CHAIN-006.md`
- `internal/oracle/service/worker.go`
- `internal/oracle/service/sql_store.go`
- `internal/settlement/service/processor.go`
- `internal/account/service/event_processor.go`
- this handshake
- `WORKLOG-CHAIN-007.md`

## Files in scope

- `internal/oracle/service/**`
- `cmd/oracle/**` only if the retry contract needs a narrow startup or replay
  hook
- `internal/settlement/**` only if a tiny correctness guard is strictly needed
- `docs/architecture/oracle-settled-crypto-markets.md` only if runtime truth
  would otherwise be unclear
- `docs/sql/schema.md`
- `migrations/**` only if the chosen retry marker requires one
- `docs/harness/handshakes/HANDSHAKE-CHAIN-007.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-007.md`

## Inputs from other threads

- `TASK-CHAIN-006` completed the first oracle runtime slice and fixed duplicate
  emits while status is already `OBSERVED`
- one residual reliability tradeoff remains:
  - if the worker writes `OBSERVED` successfully and publish fails afterward,
    there is not yet one explicit retry-safe dispatch contract

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary of:
  - publish failure after `OBSERVED`
  - repeated poll / restart behavior
  - downstream settlement/account side-effect safety

## Blockers

- do not widen into multi-provider arbitration
- do not remove the current duplicate-emit guard
- do not introduce a second Solidity toolchain
- do not break non-oracle market resolution behavior

## Status

- completed

## Handoff notes back to commander

- oracle `OBSERVED` rows now carry an explicit `evidence.dispatch` checkpoint:
  - `PENDING` means the final observation is stored and later poll / restart may
    safely retry the resolved `market.event`
  - `DISPATCHED` preserves the duplicate-emit guard for already-sent
    observations
- worker now retries only `OBSERVED + dispatch=PENDING` rows; it no longer
  re-observes the provider for that narrow failure mode
- settlement gained one tiny first-resolve guard:
  - only the first resolved `market.event` that flips `markets.status` to
    `RESOLVED` continues into cancel + payout publish
  - duplicate resolved events for the same market/outcome now no-op, which
    closes the restart window where publish succeeded but the dispatch marker
    was not persisted yet
- no migration was required; the retry marker lives inside
  `market_resolutions.evidence`
