# HANDSHAKE-CHAIN-020

## Task

- [TASK-CHAIN-020.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-020.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-019.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-019.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-020.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-020.md`

## Inputs from other threads

- `TASK-CHAIN-019` landed:
  - the first fixed outer proof/public-signal schema on top of
    `VerifierArtifactBundle`
  - deterministic Go/Foundry parity for outer proof bytes
  - a verifier that decodes that outer schema and constrains it against the
    unchanged `VerifierContext`
- commander accepted `TASK-CHAIN-019` as completed
- commander wants the next slice to keep the outer proof/public-signal envelope
  frozen and replace only the current placeholder
  `proofData = abi.encode(proofTypeHash)` with one explicit prover-facing inner
  payload shape

## Outputs back to commander

- changed files
- inner `proofData` schema contract
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `verifierGateHash`
  - the outer proof/public-signal envelope from `TASK-CHAIN-019`
  - `shadow-batch-v1` public-input shape
- replace only the inner proof-data placeholder lane:
  - define one explicit `proofData` schema version
  - export its fields deterministically from Go
  - decode and constrain it in the current verifier
- landed inner `proofData-v1` under the unchanged outer envelope:
  - `proofDataSchemaHash = keccak256("funny-rollup-proof-data-v1")`
  - `proofData = abi.encode(proofDataSchemaHash, proofTypeHash,
    batchEncodingHash, authProofHash, verifierGateHash, proofBytes)`
  - current placeholder lane keeps
    `proofTypeHash = keccak256("funny-rollup-proof-placeholder-v1")`
    and `proofBytes = bytes("")`
- Go now deterministically exports:
  - `proofData` schema version/hash/field order
  - decoded inner `proofData` fields
  - final `proofData` bytes and outer `verifierProof` bytes
- `contracts/src/FunnyRollupVerifier.sol` now decodes that inner schema and
  constrains inner/outer/context parity for
  `batchEncodingHash` / `authProofHash` / `verifierGateHash`
- repo truth stays unchanged:
  - SQL/Kafka settlement is still production truth
  - direct-vault claim is still production truth
  - this tranche is not a claim that FunnyOption is already `Mode B`

## Blockers

- do not claim the product is already Mode B
- do not widen into full prover, final cryptographic verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
