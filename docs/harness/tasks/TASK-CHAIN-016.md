# TASK-CHAIN-016

## Summary

Stabilize the verifier-facing artifact/export boundary and require
`FunnyRollupCore.acceptVerifiedBatch(...)` to anchor against previously
recorded batch metadata, while keeping the existing `shadow-batch-v1`
public-input shape unchanged and avoiding any widening into full
prover/verifier or production withdrawal rewrite.

## Scope

- build directly on `TASK-CHAIN-015`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- make the verifier-facing contract/export boundary explicit:
  - freeze the Go-side acceptance artifact shape emitted from
    `BuildVerifierStateRootAcceptanceContract(history, batch)`
  - document or encode the Go -> Solidity calldata / enum boundary needed by
    future verifier-gated acceptance workers
- tighten the minimal onchain acceptance hook:
  - require `acceptVerifiedBatch(...)` to anchor against prior
    `recordBatchMetadata(...)` state for the same batch id
  - reject acceptance if metadata for the target batch is missing or mismatched
  - keep the current `JOINED` auth-status gate intact
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `internal/rollup/**`
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-016.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-016.md`

## Acceptance criteria

- the Go-side acceptance artifact/export boundary is explicit and stable enough
  for the next verifier-facing worker to consume without guessing enum/calldata
  shape
- `FunnyRollupCore.acceptVerifiedBatch(...)` no longer accepts a batch that has
  not first been anchored via `recordBatchMetadata(...)`
- non-`JOINED` auth rows are still rejected before verifier verdict
- `shadow-batch-v1` public-input shape remains unchanged
- docs stay explicit that production truth is unchanged and no full verifier is
  present yet

## Validation

- targeted Go tests for touched rollup/export paths
- narrow Foundry tests for metadata-anchored acceptance behavior
- `git diff --check`

## Dependencies

- `TASK-CHAIN-015` completed

## Handoff

- return changed files, the stabilized verifier/export contract, validation
  commands, residual limitations, and the recommended next prover/verifier
  follow-up
