# HANDSHAKE-CHAIN-019

## Task

- [TASK-CHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-019.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-018.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-018.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-019.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-019.md`

## Inputs from other threads

- `TASK-CHAIN-018` landed:
  - the first real `FunnyRollupVerifier` contract
  - deterministic `VerifierArtifactBundle` proof-envelope export
  - onchain `verifierGateHash` recomputation from `VerifierContext`
- commander accepted `TASK-CHAIN-018` as completed
- commander wants the next slice to replace the current placeholder
  `abi.encode(proofTypeHash, verifierGateHash)` envelope with one explicit
  proof/public-signal schema, while keeping context/digest/public-input
  boundaries stable

## Outputs back to commander

- changed files
- proof/public-signal schema contract
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- `VerifierArtifactBundle` now exports one explicit first proof/public-signal
  schema on top of the unchanged acceptance contract:
  - `verifierPublicSignals = { batchEncodingHash, authProofHash,
    verifierGateHash }`
  - `proofData = abi.encode(proofTypeHash)` for the current placeholder inner
    payload
  - `verifierProof = abi.encode(proofSchemaHash, publicSignalsSchemaHash,
    verifierPublicSignals, proofData)`
- Go now deterministically exports:
  - proof schema version/hash
  - public-signal schema version/hash
  - public-signal field order and values
  - placeholder `proofData`
  - final verifier-facing `proof` bytes
- `contracts/src/FunnyRollupVerifier.sol` now decodes that schema directly and
  constrains:
  - `batchEncodingHash`
  - `authProofHash`
  - `verifierGateHash`
  against the supplied `VerifierContext` / recomputed gate hash instead of
  consuming the old two-word envelope
- `VerifierContext`, `verifierGateHash`, and `shadow-batch-v1` public-input
  shape stay unchanged
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
