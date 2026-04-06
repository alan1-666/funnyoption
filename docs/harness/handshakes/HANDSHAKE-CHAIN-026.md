# HANDSHAKE-CHAIN-026

## Task

- [TASK-CHAIN-026.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-026.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-023.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-023.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-023.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `contracts/src/FunnyRollupCore.sol`
- `contracts/src/FunnyRollupVerifier.sol`
- this handshake
- `WORKLOG-CHAIN-026.md`

## Files in scope

- `internal/rollup/**`
- `cmd/rollup/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-026.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-026.md`

## Inputs from other threads

- `TASK-CHAIN-023` landed:
  - deterministic batch-specific fixed-vk Groth16 proof artifacts
  - stable outer proof/public-signal envelope
  - stable `proofData-v1`
  - stable verifier-facing `acceptVerifiedBatch(...)` boundary
- `TASK-CHAIN-024` and `TASK-CHAIN-025` landed:
  - product lifecycle truth is tightened
  - no new lifecycle follow-up is required to open this rollup submission lane
- commander/user now want one unified current-session tranche that moves the
  repo closer to the real offchain-matching -> onchain-settlement path without
  splitting work into more worker threads

## Outputs back to commander

- changed files
- deterministic shadow submission pipeline
- validation commands
- residual limitations
- recommended next live-submission/runtime follow-up

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
  - one persisted submission lane
  - one stable bundle that exports both `recordBatchMetadata` and
    `acceptVerifiedBatch`
  - one minimal repo command that prepares and prints the next bundle
- do not widen into:
  - live tx broadcasting
  - withdrawal rewrite
  - production truth switch

## Blockers

- do not claim the product is already `Mode B`
- do not add a second Solidity toolchain
- do not turn the submission lane into a production chain runtime in this
  tranche

## Status

- completed
