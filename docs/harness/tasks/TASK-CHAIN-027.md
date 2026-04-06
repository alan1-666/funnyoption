# TASK-CHAIN-027

## Summary

Implement the first live onchain submission runtime that consumes persisted
`rollup_shadow_submissions`, submits
`FunnyRollupCore.recordBatchMetadata(...)` and
`FunnyRollupCore.acceptVerifiedBatch(...)`, and durably tracks the submission
lane state without changing current production truth.

## Scope

- build directly on `TASK-CHAIN-026`
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
  - one narrow live submitter runtime for `rollup_shadow_submissions`
  - durable submission-lane state transitions for:
    - `READY`
    - `BLOCKED_AUTH`
    - `RECORD_SUBMITTED`
    - `ACCEPT_SUBMITTED`
    - `ACCEPTED`
    - `FAILED`
  - durable tx-hash / receipt tracking for metadata and acceptance legs
  - one minimal command/runtime entrypoint that can:
    - prepare the next submission if needed
    - submit the record leg
    - confirm the record leg and submit the acceptance leg
    - confirm the acceptance leg
  - one minimal chain-service bootstrap path for the submitter when rollup core
    config is present
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-026.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-026.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-026.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-026.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-026.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-026.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/chain/service](/Users/zhangza/code/funnyoption/internal/chain/service)
- [/Users/zhangza/code/funnyoption/internal/shared/config/config.go](/Users/zhangza/code/funnyoption/internal/shared/config/config.go)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)

## Owned files

- `internal/rollup/**`
- `internal/chain/service/**`
- `internal/shared/config/config.go`
- `cmd/rollup/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-027.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-027.md`

## Acceptance criteria

- repo can durably progress one persisted submission from `READY` to
  `RECORD_SUBMITTED` to `ACCEPT_SUBMITTED` to `ACCEPTED`
- `BLOCKED_AUTH` submissions never broadcast
- tx hashes, timestamps, and last failure are durably stored
- restart-safe polling can continue from stored tx state without rebuilding
  production truth
- one command can drive the next submission forward
- chain service can optionally bootstrap the submitter when rollup core config
  is present
- docs stay explicit that the repo is still not `Mode B` production truth

## Validation

- targeted Go tests for rollup submission-state transitions / tx tracking
- `go test ./internal/rollup ./internal/chain/service ./cmd/rollup`
- existing `forge test --match-path contracts/test/FunnyRollupCore.t.sol`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-026` completed

## Handoff

- return changed files, submission-runtime behavior, validation commands,
  residual limitations, and the recommended next follow-up for a true
  prover-driven / chain-accepted state-root lane
