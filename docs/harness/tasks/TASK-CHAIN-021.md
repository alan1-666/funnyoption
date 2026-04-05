# TASK-CHAIN-021

## Summary

Define the first real proof-bytes / proving-system contract under the fixed
outer proof/public-signal envelope and `proofData-v1`: decide whether the
later real prover can keep emitting proof bytes inside `proofData-v1` or
whether the repo needs an explicit `proofData-v2` before cryptographic
verification, while preserving `VerifierContext`, `verifierGateHash`,
`shadow-batch-v1` public inputs, and the current production-truth boundary.

## Scope

- build directly on `TASK-CHAIN-020`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- keep the existing verifier-facing boundary frozen:
  - preserve `VerifierContext`
  - preserve `shadow-batch-v1` public inputs
  - preserve `verifierGateHash`
  - preserve the outer proof/public-signal envelope from `TASK-CHAIN-019`
  - preserve `proofData-v1` exactly as landed in `TASK-CHAIN-020` unless the
    design conclusion explicitly requires a new `proofData-v2`
- decide the next proving-system contract:
  - what `proofTypeHash` semantically identifies
  - whether real prover output can live inside `proofData-v1.proofBytes`
  - whether vk/circuit metadata must be explicit in the schema
  - what Go exports and Solidity verifier inputs must stay stable
  - what the next real prover worker should emit
- do not implement:
  - full prover generation
  - final cryptographic verifier
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - full rollup contract system

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-020.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-020.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-020.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-020.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-020.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-020.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-021.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-021.md`
- `internal/rollup/**` only if needed for narrow contract-shape notes or placeholders
- `contracts/src/**` only if needed for narrow contract-boundary notes or placeholders

## Acceptance criteria

- repo has one explicit design decision for the first real proving-system /
  proof-bytes contract
- docs clearly say whether real prover output remains inside `proofData-v1` or
  needs `proofData-v2`
- docs clearly define what `proofTypeHash` identifies and what verifier-facing
  metadata must remain stable
- docs stay explicit that production truth is unchanged and no final
  cryptographic verifier/prover is present yet

## Validation

- doc consistency checks across architecture/schema/harness files
- `git diff --check`

## Dependencies

- `TASK-CHAIN-020` completed

## Handoff

- return changed files, the chosen proving-system/proof-bytes contract,
  rejected options, migration consequences, and the recommended next real
  prover/verifier implementation tranche
