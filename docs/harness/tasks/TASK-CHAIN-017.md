# TASK-CHAIN-017

## Summary

Implement the first prover/verifier artifact tranche on top of the stabilized
`solidity_export` boundary: make Go emit the deterministic verifier-facing
artifact shape a future prover can consume, replace the current verifier stub
boundary with a real verifier-facing interface contract, and prove Go/Solidity
digest parity without widening into production withdrawal rewrite.

## Scope

- build directly on `TASK-CHAIN-016`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- consume the stabilized acceptance/export contract:
  - use `BuildVerifierStateRootAcceptanceContract(...).SolidityExport`
  - keep `shadow-batch-v1` public-input shape unchanged
  - preserve current `JOINED` auth-status gate
- land the first verifier-facing artifact lane:
  - emit one deterministic prover/verifier artifact bundle from Go that
    includes the existing acceptance contract plus the exact verifier-gate
    digest inputs
  - prove Go/Solidity parity for `verifierGateHash`
  - freeze any enum/bytes32/argument-order assumptions in tests/docs
- tighten the contract boundary:
  - replace the current minimal verifier stub interface with one real
    verifier-facing interface contract boundary suitable for later verifier
    implementation
  - do not implement the full cryptographic verifier itself yet
- do not implement:
  - full prover generation
  - full verifier logic
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - full rollup contract system

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-016.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-016.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-016.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-016.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-016.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-016.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-017.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-017.md`

## Acceptance criteria

- Go emits one deterministic verifier-facing artifact bundle that a later
  prover/verifier worker can consume without guessing enum/bytes32/call-data
  shape
- Go/Solidity digest parity for `verifierGateHash` is proved in tests
- the verifier boundary is no longer just the current bare stub and is explicit
  enough for later real verifier integration
- `shadow-batch-v1` public-input shape remains unchanged
- docs stay explicit that production truth is unchanged and no full verifier is
  present yet

## Validation

- targeted Go tests for touched rollup/export paths
- narrow Foundry tests for hash parity / interface boundary
- `git diff --check`

## Dependencies

- `TASK-CHAIN-016` completed

## Handoff

- return changed files, the first prover/verifier artifact contract, validation
  commands, residual limitations, and the recommended next real verifier /
  prover follow-up
