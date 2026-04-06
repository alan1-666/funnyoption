# TASK-CHAIN-028

## Summary

Harden the live rollup submission runtime so persisted
`rollup_shadow_submissions` only advance after the expected
`FunnyRollupCore` onchain state is actually visible, and add one narrow
`submit-until-idle` command path that can drive the local submission lane to a
stable stop without changing current production truth.

## Scope

- build directly on `TASK-CHAIN-027`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- keep the current verifier/proof boundary frozen:
  - preserve `VerifierContext`
  - preserve the outer proof/public-signal envelope
  - preserve `proofData-v1`
  - preserve the fixed Groth16 `proofTypeHash`
  - preserve `shadow-batch-v1` public inputs
- implement:
  - one narrow onchain reconciliation path after
    `recordBatchMetadata(...)` receipt success
  - one narrow onchain reconciliation path after
    `acceptVerifiedBatch(...)` receipt success
  - one stable Go-side export/runtime contract that compares the persisted
    submission bundle with the actual `FunnyRollupCore` getters instead of
    trusting receipt success alone
  - one minimal `cmd/rollup -mode=submit-until-idle` path that keeps polling
    until the lane reaches a stable idle/blocking condition or timeout
- do not implement:
  - production truth switch
  - withdrawal rewrite
  - forced-withdrawal runtime
  - a new proof/public-signal envelope
  - a second Solidity toolchain

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-027.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-027.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-027.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-027.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-027.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-027.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/chain/service](/Users/zhangza/code/funnyoption/internal/chain/service)
- [/Users/zhangza/code/funnyoption/cmd/rollup](/Users/zhangza/code/funnyoption/cmd/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)

## Owned files

- `internal/rollup/**`
- `internal/chain/service/**`
- `cmd/rollup/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-028.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-028.md`

## Acceptance criteria

- receipt success alone is no longer enough to advance the live submission
  state machine
- the runtime verifies the expected `FunnyRollupCore` state for:
  - recorded metadata
  - accepted batch state root
  before promoting the persisted submission status
- a metadata/acceptance mismatch does not silently promote the submission
- one command can drive the runtime until the lane is idle, blocked, failed,
  or the command timeout expires
- docs stay explicit that the repo is still not `Mode B` production truth

## Validation

- targeted Go tests for onchain reconciliation / submit-until-idle behavior
- `go test ./internal/rollup ./internal/chain/service ./cmd/rollup`
- existing `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- one local `cmd/rollup -mode=submit-until-idle` dry/local run
- `git diff --check`

## Dependencies

- `TASK-CHAIN-027` completed

## Handoff

- return changed files, onchain-reconciliation behavior, validation commands,
  residual limitations, and the recommended next follow-up for a true
  state-transition prover / accepted-root truth switch
