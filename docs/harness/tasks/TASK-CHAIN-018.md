# TASK-CHAIN-018

## Summary

Implement the first real verifier contract tranche on top of
`VerifierArtifactBundle`: consume the stabilized `VerifierContext`, recompute
and constrain `verifierGateHash` onchain, and prepare the path toward later
real proof checking without widening into production withdrawal rewrite.

## Scope

- build directly on `TASK-CHAIN-017`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- consume the stabilized artifact boundary:
  - use `rollup.VerifierArtifactBundle`
  - keep `shadow-batch-v1` public-input shape unchanged
  - preserve the current `JOINED` auth-status gate and metadata anchoring
- land the first real verifier implementation boundary:
  - implement one Foundry verifier contract that consumes
    `FunnyRollupVerifierTypes.VerifierContext`
  - recompute and constrain `verifierGateHash` onchain from the provided
    context instead of trusting the caller-supplied digest alone
  - keep proof input bytes as an explicit boundary, but do not implement the
    final cryptographic proof system yet
  - prove Go/Solidity parity against `VerifierArtifactBundle` fixtures
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-017.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-017.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-017.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-017.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-017.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-017.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-018.md`

## Acceptance criteria

- repo has one real verifier implementation boundary that consumes
  `FunnyRollupVerifierTypes.VerifierContext`
- onchain verifier no longer blindly trusts the caller-provided
  `verifierGateHash`, and recomputes/constrains it from the supplied context
- Go/Solidity parity remains pinned against `VerifierArtifactBundle` fixtures
- `shadow-batch-v1` public-input shape remains unchanged
- docs stay explicit that production truth is unchanged and no final crypto
  verifier/prover is present yet

## Validation

- targeted Go tests for touched rollup/artifact paths
- narrow Foundry tests for verifier context / digest enforcement
- `git diff --check`

## Dependencies

- `TASK-CHAIN-017` completed

## Handoff

- return changed files, the first real verifier contract boundary, validation
  commands, residual limitations, and the recommended next proof/verifier
  follow-up
