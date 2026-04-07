# TASK-CHAIN-020

## Summary

Stabilize the first inner `proofData` schema beneath the fixed
`funny-rollup-proof-envelope-v1` outer proof/public-signal envelope: replace
the current placeholder `proofData = abi.encode(proofTypeHash)` lane with one
explicit prover-facing inner payload shape that a later real prover can emit
and the current verifier can decode, without reopening the existing
`VerifierContext`, `verifierGateHash`, outer public-signal schema, or
production-truth boundary.

## Scope

- build directly on `TASK-CHAIN-019`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- keep the existing outer boundary frozen:
  - preserve `VerifierContext`
  - preserve `shadow-batch-v1` public inputs
  - preserve `verifierGateHash`
  - preserve the outer proof/public-signal envelope from `TASK-CHAIN-019`
- stabilize the next inner artifact layer:
  - define one explicit `proofData` schema version
  - define the deterministic field order and encoding for that inner payload
  - export those fields from Go in a deterministic way
  - make the current verifier decode and enforce that inner schema
  - keep the current verifier as a placeholder/digest-consistency gate, not a
    final cryptographic verifier
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-019.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-019.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-019.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-020.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-020.md`

## Acceptance criteria

- repo has one explicit inner `proofData` schema layered under the unchanged
  outer proof/public-signal envelope
- Go exports that inner schema deterministically
- current verifier decodes and enforces that inner schema instead of the old
  single-word placeholder payload
- `VerifierContext`, `verifierGateHash`, outer proof/public-signal schema, and
  `shadow-batch-v1` public-input shape remain unchanged
- docs stay explicit that production truth is unchanged and no final
  cryptographic verifier/prover is present yet

## Validation

- targeted Go tests for touched rollup/artifact paths
- narrow Foundry tests for inner proof-data schema decoding and parity
- `git diff --check`

## Dependencies

- `TASK-CHAIN-019` completed

## Handoff

- return changed files, the inner `proofData` schema contract, validation
  commands, residual limitations, and the recommended next prover/verifier
  follow-up
