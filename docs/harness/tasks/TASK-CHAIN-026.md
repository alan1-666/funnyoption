# TASK-CHAIN-026

## Summary

Implement the first deterministic shadow-batch submission pipeline that bridges
off-chain matching/settlement replay outputs into onchain
`FunnyRollupCore.recordBatchMetadata(...)` and
`FunnyRollupCore.acceptVerifiedBatch(...)` payloads, while keeping current
production truth unchanged.

## Scope

- build directly on `TASK-CHAIN-023`, `TASK-CHAIN-024`, and `TASK-CHAIN-025`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- keep the current verifier-facing/proof-facing boundary frozen:
  - preserve `VerifierContext`
  - preserve the current outer proof/public-signal envelope
  - preserve `proofData-v1`
  - preserve the fixed Groth16 `proofTypeHash`
  - preserve `shadow-batch-v1` public inputs
- implement:
  - one deterministic submission-bundle contract for a stored shadow batch
  - one stable export for:
    - `recordBatchMetadata(...)`
    - `acceptVerifiedBatch(...)`
    - ABI-encoded calldata bytes for both calls
  - one persisted shadow submission lane in Postgres so onchain-ready payloads
    are durable and auditable instead of chat-only or ad hoc local JSON
  - one minimal repo command that can:
    - materialize the next pending shadow batch if needed
    - prepare/persist the next shadow submission bundle
    - print the resulting submission bundle as JSON
- do not implement:
  - live tx broadcasting to BSC
  - production truth switch
  - withdrawal rewrite
  - forced-withdrawal runtime
  - a new proof/public-signal envelope

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-023.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-023.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-023.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-023.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-023.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-023.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)

## Owned files

- `internal/rollup/**`
- `cmd/rollup/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-026.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-026.md`

## Acceptance criteria

- repo can deterministically build one persisted shadow submission bundle for a
  stored batch
- the submission bundle includes both:
  - `recordBatchMetadata(...)` export
  - `acceptVerifiedBatch(...)` export
  - ABI-encoded calldata for each call
- submission readiness is explicit:
  - `READY` if auth rows are fully `JOINED`
  - `BLOCKED_AUTH` otherwise
- one repo command can prepare the next pending submission and print the bundle
  as JSON
- docs stay explicit that the repo is still not `Mode B` production truth

## Validation

- targeted Go tests for rollup submission-bundle export / calldata encoding
- `go test ./internal/rollup ./cmd/rollup`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-023` completed
- `TASK-CHAIN-025` completed

## Handoff

- return changed files, submission pipeline behavior, validation commands,
  residual limitations, and the recommended next follow-up for live
  tx-submission / state-root acceptance runtime
