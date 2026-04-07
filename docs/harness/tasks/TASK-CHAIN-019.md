# TASK-CHAIN-019

## Summary

Stabilize the first proof/public-signal schema on top of
`VerifierArtifactBundle`: replace the current placeholder proof-envelope
contract with an explicit proof/public-signal artifact shape that a later real
prover can emit and the current verifier can decode, without reopening the
existing `VerifierContext`, `verifierGateHash`, or production-truth boundary.

## Scope

- build directly on `TASK-CHAIN-018`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- stabilize the next artifact layer:
  - keep `VerifierArtifactBundle` as the source of truth
  - preserve the current `VerifierContext` shape
  - preserve `shadow-batch-v1` public inputs
  - preserve the current `JOINED` auth-status gate and metadata anchoring
- replace the placeholder proof-envelope contract with an explicit
  proof/public-signal schema:
  - define the next proof bytes layout
  - define any public-signal fields that must stay aligned with
    `verifierGateHash` / `authProofHash`
  - export those fields from Go in a deterministic way
  - make the current verifier decode and enforce that schema
  - do not yet implement the final cryptographic proof system
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-018.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-018.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-018.md)
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
- `docs/harness/handshakes/HANDSHAKE-CHAIN-019.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-019.md`

## Acceptance criteria

- repo has one explicit proof/public-signal schema layered on top of
  `VerifierArtifactBundle`
- Go exports that schema deterministically
- current verifier decodes and enforces that schema instead of the old simple
  proof envelope
- `VerifierContext`, `verifierGateHash`, and `shadow-batch-v1` public-input
  shape remain unchanged
- docs stay explicit that production truth is unchanged and no final
  cryptographic verifier/prover is present yet

## Validation

- targeted Go tests for touched rollup/artifact paths
- narrow Foundry tests for proof/public-signal schema decoding and parity
- `git diff --check`

## Dependencies

- `TASK-CHAIN-018` completed

## Handoff

- return changed files, the proof/public-signal schema contract, validation
  commands, residual limitations, and the recommended next prover/verifier
  follow-up
