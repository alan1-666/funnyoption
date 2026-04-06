# HANDSHAKE-CHAIN-029

## Task

- [TASK-CHAIN-029.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-029.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-028.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-028.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-028.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/chain/service/**`
- `contracts/src/FunnyVault.sol`
- `scripts/local-full-flow.sh`
- this handshake
- `WORKLOG-CHAIN-029.md`

## Files in scope

- `internal/rollup/**`
- `internal/chain/service/**`
- `cmd/rollup/**`
- `scripts/**`
- `migrations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-029.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-029.md`

## Inputs from other threads

- `TASK-CHAIN-028` landed:
  - persisted submission runtime only advances after visible `FunnyRollupCore`
    metadata / accepted state reconciliation
  - `submit-until-idle` exists
  - local chain bootstrap already deploys `FunnyRollupCore` and
    `FunnyRollupVerifier`
- user now wants the same current thread to keep pushing the core
  offchain-matching -> onchain-acceptance lane forward without splitting into
  more workers

## Outputs back to commander

- changed files
- accepted-batch / accepted-withdrawal materialization behavior
- validation commands
- real local broadcast evidence
- residual limitations
- recommended next forced-withdrawal / truth-switch follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `verifierGateHash`
  - outer proof/public-signal envelope
  - `proofData-v1`
  - fixed Groth16 `proofTypeHash`
  - `shadow-batch-v1` public-input shape
- add only:
  - one durable accepted-submission -> accepted-batch mirror
  - one accepted-withdrawal mirror / queue bridge
  - one truthful local slow-withdraw claim queue derived from accepted leaves
  - one local full-flow broadcast proof that the live lane can reach accepted
    onchain state
- do not widen into:
  - full production truth switch for settlement/accounting
  - forced withdrawal / freeze / escape hatch

## Blockers

- do not claim the product is already `Mode B`
- do not introduce a second Solidity toolchain
- do not silently treat accepted roots as production truth for balances /
  positions / payouts

## Status

- completed

## Completion notes

- accepted submissions now materialize into durable:
  - `rollup_accepted_batches`
  - `rollup_accepted_withdrawals`
- canonical `WITHDRAWAL_CLAIM` work is now queued only from accepted
  withdrawal leaves
- local `/api/v1/withdrawals` now surfaces effective accepted-claim truth
- the local lane has been proven with:
  - real `recordBatchMetadata(...)`
  - real `acceptVerifiedBatch(...)`
  - real accepted withdrawal leaf
  - real `ClaimProcessed` confirmation
