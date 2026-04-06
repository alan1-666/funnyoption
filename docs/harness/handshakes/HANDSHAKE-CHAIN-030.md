# HANDSHAKE-CHAIN-030

## Task

- [TASK-CHAIN-030.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-030.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-029.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-029.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-029.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/api/handler/sql_store.go`
- this handshake
- `WORKLOG-CHAIN-030.md`

## Files in scope

- `internal/rollup/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-030.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-030.md`

## Inputs from other threads

- `TASK-CHAIN-029` landed:
  - accepted batches / accepted withdrawals are durable
  - `WITHDRAWAL_CLAIM` only comes from accepted leaves
  - withdrawals API already reads accepted-claim truth
- user wants the same current session to keep pushing the core
  offchain-matching -> onchain-acceptance lane forward without splitting into
  more workers

## Outputs back to commander

- changed files
- accepted balance / position / payout materialization behavior
- validation commands
- local accepted read-surface evidence
- residual limitations
- recommended next forced-withdrawal / fuller truth-switch follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `verifierGateHash`
  - outer proof/public-signal envelope
  - `proofData-v1`
  - fixed Groth16 `proofTypeHash`
  - `shadow-batch-v1` public-input shape
- add only:
  - one deterministic accepted replay snapshot
  - one durable accepted balance / position / payout mirror
  - one truthful read-surface switch to accepted data once accepted batches are
    visible
- do not widen into:
  - forced-withdrawal / freeze / escape hatch
  - a mutable backend truth rewrite for matching / account / settlement writes

## Blockers

- do not claim the product is already `Mode B`
- do not introduce a second Solidity toolchain
- do not silently pretend accepted read truth means the entire mutable backend
  write path has switched

## Status

- completed

## Completion notes

- accepted-submission materialization now rebuilds deterministic accepted
  snapshot tables for:
  - balances
  - positions
  - settlement payouts
- `/api/v1/balances`, `/api/v1/positions`, and `/api/v1/payouts` now prefer
  accepted read truth once accepted batches exist
- local live API verification confirmed those three surfaces read accepted
  snapshot rows directly
