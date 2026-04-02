# WORKLOG-OFFCHAIN-007

### 2026-04-01 21:55 Asia/Shanghai

- read:
  - `WORKLOG-OFFCHAIN-005.md`
  - `HANDSHAKE-OFFCHAIN-005.md`
  - `docs/sql/local_stale_freeze_cleanup.sql`
- changed:
  - created a narrow docs/sql follow-up to align local cleanup SQL with current release semantics
- validated:
  - the issue is isolated to local cleanup artifacts and does not require runtime service changes
- blockers:
  - none yet
- next:
  - launch a worker against `TASK-OFFCHAIN-007`

### 2026-04-01 22:05 Asia/Shanghai

- read:
  - `TASK-OFFCHAIN-007.md`
  - `HANDSHAKE-OFFCHAIN-007.md`
  - `docs/operations/local-db-stale-freeze-runbook.md`
  - `docs/sql/local_stale_freeze_cleanup.sql`
- changed:
  - updated `docs/sql/local_stale_freeze_cleanup.sql` so released stale freezes also set `remaining_amount = 0`
  - updated `docs/operations/local-db-stale-freeze-runbook.md` so the documented post-cleanup row shape matches the SQL and the audit section list matches the current helper output
- validated:
  - `bash -n /Users/zhangza/code/funnyoption/scripts/audit_stale_freezes.sh`
  - `psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -c 'BEGIN' -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql -c 'ROLLBACK'`
  - rollback output confirms the two stale rows (`frz_1775043614954_105f757f1cf4`, `frz_1775043615335_4502d5cd30bb`) are returned as `RELEASED` with `remaining_amount = 0`
  - rollback output confirms `user_id=1001 asset=USDT` would move from `frozen=5100` to `frozen=4640`, with `balance_minus_active = 0` after cleanup
- blockers:
  - none
- next:
  - hand back the corrected local cleanup SQL and runbook update to commander
  - no further local DB tooling changes are required unless a new stale-freeze pattern is found outside this local-only flow
