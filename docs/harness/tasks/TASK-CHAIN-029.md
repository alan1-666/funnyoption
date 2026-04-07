# TASK-CHAIN-029

## Summary

Turn accepted rollup submissions into one truthful local slow-withdrawal lane:
materialize accepted batches from persisted accepted submissions, derive
accepted withdrawal leaves, queue canonical withdrawal-claim work only after
accepted-root visibility, and prove the lane end-to-end with one real local
pending submission broadcast.

## Scope

- build directly on `TASK-CHAIN-028`
- keep the current product truth explicit:
  - SQL/Kafka settlement remains production truth for balances / positions /
    payouts
  - direct-vault deposits remain production truth
  - do not claim FunnyOption is already full `Mode B`
- implement:
  - one accepted-submission materialization path
  - one durable accepted-batch mirror
  - one durable accepted-withdrawal mirror
  - one canonical `WITHDRAWAL_CLAIM` queue that is created only after the
    corresponding withdrawal leaf is present in an accepted batch
  - one runtime/bootstrap path that can re-materialize accepted submissions
    idempotently after restart
  - one real local pending-submission broadcast validation path that creates
    journal entries, prepares a batch, submits it, and observes accepted state
- do not implement:
  - full production truth switch for balances / positions / settlement
  - forced-withdrawal freeze / escape hatch
  - a new proof/public-signal contract
  - a second Solidity toolchain

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-028.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-028.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-028.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-028.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-028.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-028.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/backend/internal/chain/service](/Users/zhangza/code/funnyoption/backend/internal/chain/service)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
- [/Users/zhangza/code/funnyoption/scripts/local-full-flow.sh](/Users/zhangza/code/funnyoption/scripts/local-full-flow.sh)

## Owned files

- `internal/rollup/**`
- `internal/chain/service/**`
- `cmd/rollup/**`
- `scripts/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-029.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-029.md`

## Acceptance criteria

- accepted submissions can be materialized into durable accepted-batch records
- accepted withdrawal leaves become durable local records
- canonical `WITHDRAWAL_CLAIM` queue items are only created once the related
  withdrawal leaf is in an accepted batch
- restart/bootstrap can re-materialize accepted submissions idempotently
- one local run creates a real pending submission and drives it through
  `recordBatchMetadata(...)` + `acceptVerifiedBatch(...)`
- docs stay explicit that full production truth has not yet switched

## Validation

- targeted Go tests for accepted-submission materialization and claim queuing
- `go test ./internal/rollup ./internal/chain/service ./cmd/rollup`
- existing `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- one local full-flow + rollup submit run that produces a non-empty pending
  submission and reaches accepted onchain state
- `git diff --check`

## Dependencies

- `TASK-CHAIN-028` completed

## Handoff

- return changed files, accepted-batch/withdrawal materialization behavior,
  validation commands, the real local broadcast evidence, residual limitations,
  and the recommended next follow-up for forced withdrawal / fuller truth
  switch
