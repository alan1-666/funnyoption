# HANDSHAKE-CHAIN-027

## Task

- [TASK-CHAIN-027.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-027.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-026.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-026.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-026.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/chain/service/**`
- `internal/shared/config/config.go`
- `contracts/src/FunnyRollupCore.sol`
- this handshake
- `WORKLOG-CHAIN-027.md`

## Files in scope

- `internal/rollup/**`
- `internal/chain/service/**`
- `internal/shared/config/config.go`
- `cmd/rollup/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-027.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-027.md`

## Inputs from other threads

- `TASK-CHAIN-026` landed:
  - persisted deterministic `rollup_shadow_submissions`
  - stable `recordBatchMetadata(...)` and `acceptVerifiedBatch(...)` calldata
  - one minimal command that prepares the next submission bundle
- commander/user now want the current merged thread to keep moving the real
  offchain-matching -> onchain-acceptance lane forward without splitting into
  more worker threads

## Outputs back to commander

- changed files
- live submission-runtime behavior
- validation commands
- residual limitations
- recommended next prover/acceptance follow-up

## Handoff notes

- keep unchanged:
  - production truth
  - `VerifierContext`
  - `verifierGateHash`
  - outer proof/public-signal envelope
  - `proofData-v1`
  - fixed Groth16 `proofTypeHash`
  - `shadow-batch-v1` public-input shape
- add only:
  - one durable submission runtime / state machine
  - tx-hash / receipt tracking
  - one minimal chain-service bootstrap
  - one minimal repo command that can drive the runtime forward
- do not widen into:
  - production truth switch
  - withdrawal rewrite
  - forced-withdrawal runtime

## Blockers

- do not claim the product is already `Mode B`
- do not add a second Solidity toolchain
- do not treat accepted roots as current production truth

## Status

- completed
