# HANDSHAKE-CHAIN-028

## Task

- [TASK-CHAIN-028.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-028.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-027.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-027.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-027.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/chain/service/**`
- `cmd/rollup/**`
- `contracts/src/FunnyRollupCore.sol`
- this handshake
- `WORKLOG-CHAIN-028.md`

## Files in scope

- `internal/rollup/**`
- `internal/chain/service/**`
- `cmd/rollup/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-028.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-028.md`

## Inputs from other threads

- `TASK-CHAIN-027` landed:
  - persisted shadow submissions
  - restart-safe metadata/acceptance tx tracking
  - optional local rollup-core/verifier bootstrap
  - one minimal `submit-next` command path
- commander/user now want the current merged thread to keep moving the real
  offchain-matching -> onchain-acceptance lane forward in one session without
  splitting into more workers

## Outputs back to commander

- changed files
- onchain-reconciliation behavior
- validation commands
- residual limitations
- recommended next prover/state-transition follow-up

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
  - one narrow `FunnyRollupCore` read/reconciliation path
  - one stable runtime contract that compares persisted bundle expectations
    against actual onchain metadata/accepted-batch state
  - one minimal `submit-until-idle` command mode
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
