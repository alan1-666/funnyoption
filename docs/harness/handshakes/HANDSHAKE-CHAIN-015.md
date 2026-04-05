# HANDSHAKE-CHAIN-015

## Task

- [TASK-CHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-015.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-014.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-014.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-014.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/shared/auth/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-015.md`

## Files in scope

- `internal/rollup/**`
- `internal/shared/auth/**` only if the verifier gate needs narrow metadata-aligned helpers
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-015.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-015.md`

## Inputs from other threads

- `TASK-CHAIN-014` landed:
  - one normalized `VerifierAuthBinding`
  - `BuildVerifierAuthProofContract(history, batch)`
  - `BuildVerifierGateBatchContract(history, batch)`
  - explicit auth row statuses:
    - `JOINED`
    - `MISSING_TRADING_KEY_AUTHORIZED`
    - `NON_VERIFIER_ELIGIBLE`
- commander accepted `TASK-CHAIN-014` as completed
- commander wants the next slice to add the smallest Foundry-only acceptance
  hook on `FunnyRollupCore` without widening into full prover/verifier or
  production withdrawal rewrite

## Outputs back to commander

- changed files
- minimal verifier/state-root acceptance contract
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- the minimal verifier/state-root acceptance tranche now consumes the stable
  `TASK-CHAIN-014` boundary without reopening `shadow-batch-v1`:
  - `internal/rollup` now adds
    `BuildVerifierStateRootAcceptanceContract(history, batch)`
  - it projects:
    - unchanged `public_inputs`
    - unchanged `l1_batch_metadata`
    - target-batch auth status rows for the `JOINED` gate
- `FunnyRollupCore` now has two explicit lanes:
  - `recordBatchMetadata(...)` stays metadata-only
  - `acceptVerifiedBatch(...)` is the new Foundry-only acceptance hook
- the acceptance hook now enforces:
  - sequential accepted `batch_id`
  - `prev_state_root == latestAcceptedStateRoot`
  - metadata subset must match the public-input subset
  - every auth status must be `JOINED`
  - a stub verifier verdict must return success before
    `latestAcceptedStateRoot` advances
- target batches containing either:
  - `MISSING_TRADING_KEY_AUTHORIZED`
  - `NON_VERIFIER_ELIGIBLE`
  are rejected before verifier/state-root acceptance
- the repo still does **not** claim:
  - full prover
  - full verifier
  - production withdrawal rewrite
  - production `Mode B` truth

## Blockers

- do not claim the product is already Mode B
- do not widen into full prover, full verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
