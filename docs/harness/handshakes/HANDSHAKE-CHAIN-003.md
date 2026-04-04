# HANDSHAKE-CHAIN-003

## Task

- [TASK-CHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-003.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `WORKLOG-CHAIN-002.md`
- this handshake
- `WORKLOG-CHAIN-003.md`

## Files in scope

- `migrations/**`
- `docs/sql/**`
- `docs/operations/**` for narrowly related runbook updates
- `docs/harness/worklogs/WORKLOG-CHAIN-003.md`
- `internal/chain/service/**` only if schema repair and tests prove a compatibility simplification is safe

## Inputs from other threads

- `TASK-CHAIN-002` restored a truthful listener-driven local deposit proof
- that worker observed legacy local schema drift:
  - old local databases may still enforce narrow `chain_deposits.tx_hash` / `deposit_id` widths
  - current repo migrations expect wider or normalized storage than some reused local DBs actually have
- this follow-up should clean the schema/runbook story without touching bootstrap order semantics or session/order API code

## Outputs back to commander

- exact drift diagnosis
- changed files
- dry-run / apply commands and test results
- any remaining compatibility workaround that must intentionally stay in code

## Blockers

- do not touch files owned by `TASK-OFFCHAIN-010`
- do not widen into withdrawals, claims, or user-session semantics
- if no legacy local DB is available to prove an automated repair safely, document that blocker and ship a conservative docs-only recommendation rather than guessing

## Status

- completed

## Handoff notes

- repo truth and legacy drift are now documented explicitly:
  - expected: `chain_deposits.deposit_id = VARCHAR(64)`, `chain_deposits.tx_hash = VARCHAR(128)`
  - observed legacy reused-local drift: `deposit_id = VARCHAR(64)`, `tx_hash = VARCHAR(64)`
- narrow repair path shipped as:
  - `migrations/010_chain_deposits_tx_hash_width_repair.sql`
  - `docs/operations/local-chain-deposits-schema-repair.md`
  - supporting doc updates in `docs/sql/schema.md` and `docs/operations/local-persistent-chain.md`
- worker validated the repair path with synthetic drift reproduction, dry-run rollback, real apply, re-apply idempotency, and `go test ./internal/chain/...`
- no product-runtime chain code change was required; residual evidence boundary is documented explicitly because no preserved legacy local DB volume existed in this workspace
