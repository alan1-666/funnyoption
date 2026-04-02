# HANDSHAKE-CHAIN-002

## Task

- [TASK-CHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-002.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/topics/kafka-topics.md`
- `docs/operations/local-offchain-lifecycle.md`
- `WORKLOG-ADMIN-001.md`
- `TASK-CHAIN-001.md`
- this handshake
- `WORKLOG-CHAIN-002.md`

## Files in scope

- `internal/chain/service/**`
- `cmd/chain/**`
- `cmd/local-lifecycle/**`
- `scripts/dev-up.sh`
- `docs/operations/local-offchain-lifecycle.md`
- narrowly related chain/bootstrap docs

## Inputs from other threads

- `/admin` and the lifecycle proof are in place, but the current deposit step is still simulated through `ApplyConfirmedDeposit(...)`
- claim-lane payload validation is now hardened, so the next truth gap is the actual wallet funding path
- the default local env currently lacks a live vault/listener-ready proof path

## Outputs back to commander

- changed files
- the exact proof environment used
- exact commands to reproduce the deposit proof
- tx/deposit/balance evidence
- any remaining blockers for a fully honest lifecycle run

## Blockers

- do not widen into first-liquidity design or admin auth in this task
- if a real listener proof cannot be completed, record the exact missing env/config blocker instead of silently falling back to simulation

## Status

- complete

## Handoff notes

- `cmd/local-lifecycle` now proves the deposit step through an embedded go-ethereum simulated chain plus a real `DepositListener`, instead of calling `ApplyConfirmedDeposit(...)` directly.
- the successful proof environment is:
  - local product services from `scripts/dev-up.sh`
  - in-process proof chain with `chain_id=1337`, `chain_name=simulated`, `network_name=local-proof`
  - listener config `confirmations=0`, `start_block=2`
- local proof only succeeded after normalizing stored tx hashes to 64-char hex without `0x` and shortening deterministic `deposit_id` values; the worker observed that the running local DB still enforced a legacy `VARCHAR(64)` shape even though repo migrations now declare `VARCHAR(128)` for `chain_deposits.tx_hash`
