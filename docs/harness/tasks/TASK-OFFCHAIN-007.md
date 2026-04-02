# TASK-OFFCHAIN-007

## Summary

Correct the local stale-freeze cleanup artifacts so the scripted local repair flow zeroes `remaining_amount` when a freeze is released and the runbook matches that behavior.

## Scope

- fix the local cleanup SQL to align with current runtime release semantics
- update the runbook and any validation notes so released rows are expected to carry `remaining_amount = 0`
- keep the task strictly in local audit / docs / SQL scope

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-007.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-007.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-007.md)

## Owned files

- `scripts/audit_stale_freezes.sh`
- `docs/sql/local_stale_freeze_audit.sql`
- `docs/sql/local_stale_freeze_cleanup.sql`
- `docs/operations/local-db-stale-freeze-runbook.md`

## Acceptance criteria

- local cleanup SQL marks stale freezes `RELEASED` and also zeroes `remaining_amount`
- runbook explicitly states the expected post-cleanup row shape
- worker validates the cleanup SQL again inside `BEGIN ... ROLLBACK`

## Validation

- `bash -n scripts/audit_stale_freezes.sh`
- `psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -c 'BEGIN' -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql -c 'ROLLBACK'`

## Dependencies

- `TASK-OFFCHAIN-005` output is the baseline
- this task can run in parallel with `TASK-OFFCHAIN-006`

## Handoff

- return the corrected local cleanup flow and the updated runbook path
- note whether any further local DB cleanup work is still needed
