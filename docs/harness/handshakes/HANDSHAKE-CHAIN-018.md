# HANDSHAKE-CHAIN-018

## Task

- [TASK-CHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-018.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-017.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-017.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-018.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-018.md`

## Inputs from other threads

- `TASK-CHAIN-017` landed:
  - `rollup.VerifierArtifactBundle`
  - deterministic `authProofHash`
  - deterministic `verifierGateHash`
  - one verifier-facing `IFunnyRollupBatchVerifier.verifyBatch(context, proof)` boundary
- commander accepted `TASK-CHAIN-017` as completed
- commander wants the next slice to replace the current interface-only verifier
  boundary with the first real verifier implementation contract, while keeping
  the public-input/auth-status contract stable

## Outputs back to commander

- changed files
- first real verifier contract boundary
- validation commands
- residual limitations
- recommended next proof/verifier follow-up

## Handoff notes

- `BuildVerifierArtifactBundle(history, batch)` still consumes the stable
  acceptance `solidity_export`, and now also exports the first placeholder
  `verifierProof = abi.encode(proofTypeHash, verifierGateHash)` fixture
- `contracts/src/FunnyRollupVerifier.sol` now contains one real Foundry
  verifier contract that directly consumes
  `FunnyRollupVerifierTypes.VerifierContext`
- the concrete verifier contract now:
  - requires `batchEncodingHash == keccak256("shadow-batch-v1")`
  - recomputes `verifierGateHash` onchain from the supplied context
  - rejects placeholder proof bytes whose embedded digest does not match the
    recomputed gate hash
- `shadow-batch-v1` public-input shape stays unchanged
- repo truth stays unchanged:
  - SQL/Kafka settlement is still production truth
  - direct-vault claim is still production truth
  - this tranche is not a claim that FunnyOption is already `Mode B`

## Blockers

- do not claim the product is already Mode B
- do not widen into full prover, full verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
