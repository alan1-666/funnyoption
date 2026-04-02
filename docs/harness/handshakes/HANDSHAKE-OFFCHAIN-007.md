# HANDSHAKE-OFFCHAIN-007

## Task

- [TASK-OFFCHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-007.md)

## Thread owner

- docs-sql worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-OFFCHAIN-005.md`
- `HANDSHAKE-OFFCHAIN-005.md`
- `WORKLOG-OFFCHAIN-005.md`
- this handshake
- `WORKLOG-OFFCHAIN-007.md`

## Files in scope

- `scripts/audit_stale_freezes.sh`
- `docs/sql/local_stale_freeze_audit.sql`
- `docs/sql/local_stale_freeze_cleanup.sql`
- `docs/operations/local-db-stale-freeze-runbook.md`

## Inputs from other threads

- `TASK-OFFCHAIN-005` already produced the audit helper and current local DB evidence
- commander review found one semantic bug in the cleanup SQL: released rows keep non-zero `remaining_amount`

## Outputs back to commander

- changed files
- updated rollback validation result
- explicit statement that released rows now end at `remaining_amount = 0`

## Handoff notes back to commander

- `docs/sql/local_stale_freeze_cleanup.sql` now marks stale rows `RELEASED` and sets `remaining_amount = 0` in the same cleanup step.
- `docs/operations/local-db-stale-freeze-runbook.md` now documents the expected post-cleanup row shape and the aligned rollback outcome.
- validation summary:
  - `bash -n /Users/zhangza/code/funnyoption/scripts/audit_stale_freezes.sh`: PASS
  - `psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -c 'BEGIN' -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql -c 'ROLLBACK'`: PASS
  - rollback output confirms the previously stale rows return as `RELEASED` with `remaining_amount = 0`

## Blockers

- do not widen into runtime reconciliation or `internal/**` code

## Status

- completed
