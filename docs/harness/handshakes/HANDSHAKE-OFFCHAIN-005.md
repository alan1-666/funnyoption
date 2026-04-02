# HANDSHAKE-OFFCHAIN-005

## Task

- [TASK-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-005.md)

## Thread owner

- implementation or docs-sql worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/sql/schema.md`
- `docs/architecture/ledger-service.md`
- `TASK-OFFCHAIN-002.md`
- `HANDSHAKE-OFFCHAIN-002.md`
- `WORKLOG-OFFCHAIN-002.md`
- this handshake
- `WORKLOG-OFFCHAIN-005.md`

## Files in scope

- `scripts/**`
- `docs/sql/**`
- `docs/deploy/**`
- `docs/operations/**`

## Inputs from other threads

- `TASK-OFFCHAIN-002` observed a reused local DB with historical stale BUY-side quote freezes that were created before the recent account fix
- `TASK-OFFCHAIN-004` is fixing resolved-market finality and should not be blocked or expanded by this task

## Outputs back to commander

- changed files
- audit steps and evidence
- recommendation on whether a later code-level reconciliation task is still needed

## Handoff notes back to commander

- changed files:
  - `scripts/audit_stale_freezes.sh`
  - `docs/sql/local_stale_freeze_audit.sql`
  - `docs/sql/local_stale_freeze_cleanup.sql`
  - `docs/operations/local-db-stale-freeze-runbook.md`
- current local DB evidence:
  - `user_id=1001 asset=USDT` still has `frozen=5100`
  - two stale terminal BUY freezes are still `ACTIVE` and sum to `460`
  - one open BUY reserve for `4640` is still legitimate and should remain
- validation:
  - the audit helper runs against the current local DB and cleanly isolates stale terminal BUY quote freezes from live reserves
  - the cleanup SQL was executed inside `BEGIN ... ROLLBACK`, proving it matches the current schema and would reduce `1001/USDT frozen` from `5100` to `4640` without touching runtime code
- recommendation:
  - immediate next action: `local scriptable cleanup`
  - future productized reconciliation: only if this stale-freeze pattern is discovered in shared/non-local data or if a future audit shows `balance_minus_active <> 0`
- commander review found one cleanup-semantic follow-up:
  - `docs/sql/local_stale_freeze_cleanup.sql` updates `freeze_records.status` to `RELEASED` but does not also zero `remaining_amount`, which would preserve the same released-row inconsistency that newer runtime release paths now avoid

## Blockers

- do not change runtime service code or widen into a general reconciliation engine
- the original cleanup-semantic follow-up was closed by `TASK-OFFCHAIN-007`

## Status

- completed
