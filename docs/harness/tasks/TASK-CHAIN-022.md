# TASK-CHAIN-022

## Summary

Implement the first Foundry-only real Groth16 backend under the fixed outer
proof/public-signal envelope and `proofData-v1`: replace the current empty
`proofBytes` placeholder with one real verifier-facing Groth16 lane on BN254,
keep `VerifierContext` / `verifierGateHash` / `shadow-batch-v1` unchanged, and
pin Go/Foundry parity for limb lifting, proof-bytes codec, and verifier
verdicts without widening into production withdrawal rewrite.

## Scope

- build directly on `TASK-CHAIN-021`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- keep the existing verifier-facing boundary frozen:
  - preserve `VerifierContext`
  - preserve `shadow-batch-v1` public inputs
  - preserve `verifierGateHash`
  - preserve the outer proof/public-signal envelope from `TASK-CHAIN-019`
  - preserve `proofData-v1` from `TASK-CHAIN-020`
  - use the `TASK-CHAIN-021` lane choice:
    - fixed-vk `Groth16` on `BN254`
    - fixed `proofTypeHash`
    - `proofBytes = abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)`
    - `bytes32 -> 2x uint128` public-signal lifting
- implement:
  - Go-side exporter support for non-empty Groth16 `proofBytes`
  - Solidity-side dispatch on the fixed Groth16 `proofTypeHash`
  - decoding the Groth16 tuple from `proofData-v1.proofBytes`
  - deriving the six BN254 field inputs from the unchanged outer signals
  - one Foundry-only real cryptographic verifier backend contract
  - Go/Foundry parity fixtures for limb splitting, proof codec, and verifier verdict
- do not implement:
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - a second Solidity toolchain
  - full production Mode B truth switch

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-021.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-021.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-021.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-021.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-021.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-021.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)
- `foundry.toml`

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-022.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-022.md`

## Acceptance criteria

- repo has one real Foundry-only Groth16 backend wired behind the fixed `proofTypeHash`
- Go exporter can emit non-empty `proofBytes` for the fixed Groth16 lane
- Solidity verifier dispatches on that `proofTypeHash`, decodes the Groth16 tuple, and derives six BN254 public inputs from unchanged outer signals
- Go/Foundry parity fixtures pin limb splitting, proof-bytes codec, and verifier verdict expectations
- docs stay explicit that production truth is unchanged and the repo is still not yet Mode B production truth

## Validation

- targeted Go tests for touched rollup/export paths
- narrow Foundry tests for Groth16 proof dispatch/decoding/parity
- `git diff --check`

## Dependencies

- `TASK-CHAIN-021` completed

## Handoff

- return changed files, the real Groth16 backend boundary, validation
  commands, residual limitations, and the recommended next prover/verifier
  follow-up
