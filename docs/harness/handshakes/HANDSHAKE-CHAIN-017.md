# HANDSHAKE-CHAIN-017

## Task

- [TASK-CHAIN-017.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-017.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-016.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-016.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-017.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-017.md`

## Inputs from other threads

- `TASK-CHAIN-016` landed:
  - stable `solidity_export` from `BuildVerifierStateRootAcceptanceContract(...)`
  - metadata-anchored `FunnyRollupCore.acceptVerifiedBatch(...)`
  - explicit non-`JOINED` auth-status rejection before verifier verdict
- commander accepted `TASK-CHAIN-016` as completed
- commander wants the next slice to make the first real prover/verifier
  artifact lane explicit without widening into full production rewrite

## Outputs back to commander

- changed files
- first prover/verifier artifact contract
- validation commands
- residual limitations
- recommended next real verifier / prover follow-up

## Handoff notes

- `BuildVerifierArtifactBundle(history, batch)` now directly consumes
  `BuildVerifierStateRootAcceptanceContract(...).SolidityExport` and emits the
  first deterministic prover/verifier artifact bundle with:
  - unchanged acceptance contract
  - deterministic `authProofHash`
  - deterministic `verifierGateHash`
  - verifier-facing `IFunnyRollupBatchVerifier.verifyBatch(context, proof)`
- Go and Solidity now pin one shared fixture proving `verifierGateHash`
  digest parity
- `FunnyRollupCore` no longer talks to a bare hash-only verifier stub; it now
  passes a full verifier context containing:
  - `batchEncodingHash`
  - `publicInputs`
  - `authProofHash`
  - `verifierGateHash`
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
