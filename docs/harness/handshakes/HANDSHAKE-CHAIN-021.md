# HANDSHAKE-CHAIN-021

## Task

- [TASK-CHAIN-021.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-021.md)

## Thread owner

- chain/rollup design worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-020.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-020.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupVerifier.sol`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/test/**`
- this handshake
- `WORKLOG-CHAIN-021.md`

## Files in scope

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
- `internal/rollup/**` only if needed for narrow contract-shape notes or placeholders
- `contracts/src/**` only if needed for narrow contract-boundary notes or placeholders

## Inputs from other threads

- `TASK-CHAIN-020` landed:
  - one fixed `proofData-v1` schema under the unchanged outer envelope
  - deterministic Go/Foundry parity for inner `proofData`
  - verifier checks for inner/outer/context parity while keeping production
    truth unchanged
- commander accepted `TASK-CHAIN-020` as completed
- commander wants the next slice to decide the first real proving-system /
  proof-bytes contract before any worker attempts real prover output or
  cryptographic verification

## Outputs back to commander

- changed files
- chosen proving-system / proof-bytes contract
- rejected options
- migration consequences
- recommended next real prover/verifier implementation tranche

## Handoff notes

- keep unchanged unless the design conclusion explicitly requires otherwise:
  - `VerifierContext`
  - `verifierGateHash`
  - the outer proof/public-signal envelope from `TASK-CHAIN-019`
  - `proofData-v1` from `TASK-CHAIN-020`
  - `shadow-batch-v1` public-input shape
- explicitly decide:
  - what `proofTypeHash` identifies
  - whether `proofData-v1.proofBytes` is enough for the first real proving system
  - whether vk/circuit metadata needs a `proofData-v2`
  - what the next prover/verifier implementation worker should emit and verify
- landed design decision:
  - first real proving-system contract = fixed-vk `Groth16` on `BN254`
  - first real `proofTypeHash =
    keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1")`
  - `proofTypeHash` now explicitly means the full verifier-facing proof
    contract:
    - proving system + curve
    - outer `bytes32` public-signal lifting rule
    - exact circuit / verifying-key lane
    - `proofBytes` ABI codec
  - first real prover output stays inside `proofData-v1.proofBytes`
  - fixed first real `proofBytes` contract =
    `abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
  - fixed `BN254` public-input lifting contract =
    split each outer `bytes32` public signal into `hi/lo uint128` limbs in the
    order
    `batchEncodingHashHi, batchEncodingHashLo, authProofHashHi,
    authProofHashLo, verifierGateHashHi, verifierGateHashLo`
  - `proofData-v2` is explicitly **not** required for that first fixed-vk lane
  - `proofData-v2` is only required if verifier-relevant metadata such as
    `vkHash`, `circuitHash`, aggregation-program id, or multiple proof payloads
    must travel separately from `proofTypeHash + proofBytes`
- rejected options:
  - treating `proofTypeHash` as only `Groth16` / `Plonk` / proving-family label
  - introducing `proofData-v2` before the first real prover/verifier worker
  - choosing a first lane that needs a second Solidity contract toolchain
    instead of the repo's existing Foundry path
- next implementation tranche should:
  - keep the outer proof/public-signal envelope and `proofData-v1` unchanged
  - replace the placeholder `proofTypeHash` with the fixed Groth16 lane above
  - accept non-empty `proofBytes`, decode the Groth16 tuple, derive the six
    `BN254` field inputs from the unchanged outer signals, and call one real
    cryptographic verifier backend
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
