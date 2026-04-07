# TASK-CHAIN-023

## Summary

Implement the fixed-vk Groth16 prover artifact pipeline on top of the existing
Foundry-only backend: replace the current one shared fixture proof with
batch-specific proof artifacts generated from actual outer signals, while
keeping the outer proof/public-signal envelope, `proofData-v1`, fixed
`proofTypeHash`, and current production-truth boundary unchanged.

## Scope

- build directly on `TASK-CHAIN-022`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- keep the existing verifier-facing boundary frozen:
  - preserve `VerifierContext`
  - preserve `shadow-batch-v1` public inputs
  - preserve `verifierGateHash`
  - preserve the outer proof/public-signal envelope from `TASK-CHAIN-019`
  - preserve `proofData-v1`
  - preserve the fixed Groth16 `proofTypeHash`
  - preserve the `bytes32 -> 2x uint128` BN254 lifting rule
- implement:
  - Go-side batch-specific proof artifact generation for the fixed-vk Groth16 lane
  - deterministic artifact materialization from actual outer signals
  - updated Go/Foundry parity fixtures for per-batch proofBytes and verifier verdicts
  - if needed, narrow repo-local proving helpers/scripts to support reproducible artifact generation
- do not implement:
  - a new outer proof/public-signal schema
  - `proofData-v2`
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - production truth switch

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-022.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-022.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-022.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-022.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-022.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-022.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupGroth16Backend.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupGroth16Backend.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)
- `foundry.toml`

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-023.md`
- narrow repo-local proving helper paths only if required

## Acceptance criteria

- repo can produce batch-specific fixed-vk Groth16 proof artifacts from actual outer signals
- outer proof/public-signal envelope and `proofData-v1` remain unchanged
- Go/Foundry parity fixtures pin batch-specific proofBytes and verifier verdicts
- docs stay explicit that production truth is unchanged and the repo is still not yet Mode B production truth

## Validation

- targeted Go tests for touched rollup/proving paths
- narrow Foundry tests for batch-specific Groth16 proof verification
- `git diff --check`

## Dependencies

- `TASK-CHAIN-022` completed

## Handoff

- return changed files, the batch-specific proof artifact pipeline, validation
  commands, residual limitations, and the recommended next prover/verifier
  follow-up
