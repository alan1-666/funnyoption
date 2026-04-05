# HANDSHAKE-CHAIN-016

## Task

- [TASK-CHAIN-016.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-016.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-015.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-015.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-015.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-016.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-016.md`

## Inputs from other threads

- `TASK-CHAIN-015` landed:
  - `BuildVerifierStateRootAcceptanceContract(history, batch)`
  - `FunnyRollupCore.acceptVerifiedBatch(...)`
  - strict rejection of non-`JOINED` auth rows before verifier verdict
- commander accepted `TASK-CHAIN-015` as completed
- commander found one non-blocking follow-up:
  - accepted batches are not yet required to anchor against prior
    `recordBatchMetadata(...)`
  - without that, the current metadata subset check can still be satisfied by
    self-consistent calldata alone

## Outputs back to commander

- changed files
- stabilized verifier/export contract
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- `BuildVerifierStateRootAcceptanceContract(history, batch)` now keeps the
  existing `public_inputs` / `l1_batch_metadata` / auth-status projection and
  also exports one stable `solidity_export` artifact for
  `FunnyRollupCore.acceptVerifiedBatch(...)`
- the new export freezes:
  - contract / function identity
  - argument order
  - struct field names and Solidity types
  - `AuthJoinStatus` enum ordinals
  - normalized `0x`-prefixed `bytes32` calldata values for the exported args
- `FunnyRollupCore.acceptVerifiedBatch(...)` now rejects target batches unless:
  - the same `batch_id` was previously recorded through
    `recordBatchMetadata(...)`
  - recorded metadata matches the acceptance calldata
  - every projected auth row is `JOINED`
- `shadow-batch-v1` public-input shape remains unchanged
- repo truth remains unchanged:
  - current SQL/Kafka settlement is still production truth
  - direct-vault claim is still production truth
  - this does not make FunnyOption already `Mode B`

## Blockers

- do not claim the product is already Mode B
- do not widen into full prover, full verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
