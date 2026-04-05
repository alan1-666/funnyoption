# HANDSHAKE-CHAIN-022

## Task

- [TASK-CHAIN-022.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-022.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-021.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- `foundry.toml`
- this handshake
- `WORKLOG-CHAIN-022.md`

## Files in scope

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-022.md`

## Inputs from other threads

- `TASK-CHAIN-021` landed:
  - first real lane = fixed-vk `Groth16` on `BN254`
  - fixed `proofTypeHash`
  - fixed `proofBytes` ABI codec
  - fixed `bytes32 -> 2x uint128` BN254 lifting contract
  - explicit decision that real prover output can stay inside `proofData-v1`
- commander accepted `TASK-CHAIN-021` as completed
- commander wants the next slice to implement that exact lane without
  reopening `VerifierContext`, outer proof/public-signal schema, or
  `proofData-v1`

## Outputs back to commander

- changed files
- real Groth16 backend boundary
- validation commands
- residual limitations
- recommended next prover/verifier follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `verifierGateHash`
  - outer proof/public-signal envelope
  - `proofData-v1`
  - `shadow-batch-v1` public-input shape
- implement exactly:
  - `proofTypeHash =
    keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
  - `proofBytes = abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
  - BN254 field inputs ordered as:
    `batchEncodingHashHi, batchEncodingHashLo, authProofHashHi,
    authProofHashLo, verifierGateHashHi, verifierGateHashLo`
- landed in this tranche:
  - Go exporter now emits non-empty fixed-fixture Groth16 `proofBytes` inside
    unchanged `proofData-v1`
  - `FunnyRollupVerifier` now dispatches on the fixed Groth16
    `proofTypeHash`, decodes the tuple codec, derives the six BN254 inputs
    from unchanged outer public signals, and forwards them into one
    Foundry-only fixed-vk `FunnyRollupGroth16Backend`
  - Go / Foundry parity fixtures now pin limb splitting, proof-bytes codec,
    and one expected verifier `true` verdict for the shared fixture lane
- repo truth stays unchanged:
  - SQL/Kafka settlement is still production truth
  - direct-vault claim is still production truth
  - this tranche is not a claim that FunnyOption is already `Mode B`

## Blockers

- do not claim the product is already Mode B
- do not widen into production withdrawal-claim rewrite or forced-withdrawal runtime
- if contracts are touched, stay on Foundry only

## Status

- completed
