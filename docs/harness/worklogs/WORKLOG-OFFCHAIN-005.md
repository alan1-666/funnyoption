# WORKLOG-OFFCHAIN-005

### 2026-04-01 21:08 Asia/Shanghai

- read:
  - `WORKLOG-OFFCHAIN-002.md`
  - `docs/sql/schema.md`
  - `migrations/001_init.sql`
- changed:
  - created a parallel task for stale-freeze audit and local cleanup guidance
- validated:
  - the task is disjoint from `TASK-OFFCHAIN-004` runtime ownership
- blockers:
  - none yet; worker should inspect the current local DB and repo schema first
- next:
  - launch a worker against `TASK-OFFCHAIN-005`

### 2026-04-01 21:14 Asia/Shanghai

- read:
  - `docs/architecture/ledger-service.md`
  - `TASK-OFFCHAIN-002.md`
  - `HANDSHAKE-OFFCHAIN-002.md`
  - `WORKLOG-OFFCHAIN-002.md`
  - `.env.local`
  - `migrations/001_init.sql`
  - `deploy/postgres/docker-compose.yml`
  - `scripts/dev-up.sh`
  - `scripts/apply_migrations.sh`
- changed:
  - added `scripts/audit_stale_freezes.sh`
  - added `docs/sql/local_stale_freeze_audit.sql`
  - added `docs/sql/local_stale_freeze_cleanup.sql`
  - added `docs/operations/local-db-stale-freeze-runbook.md`
- validated:
  - `bash -n scripts/audit_stale_freezes.sh`
  - `/Users/zhangza/code/funnyoption/scripts/audit_stale_freezes.sh`
  - `psql 'postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable' -v ON_ERROR_STOP=1 -c 'BEGIN' -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql -c 'ROLLBACK'`
  - audit evidence from the current local DB:
    - `account_balances` shows `user_id=1001 asset=USDT frozen=5100`
    - `freeze_records` shows two stale terminal BUY quote freezes:
      - `frz_1775043614954_105f757f1cf4` for `ord_1775043614954_a16c576344c5` with `remaining_amount=250`
      - `frz_1775043615335_4502d5cd30bb` for `ord_1775043615335_47037c23808b` with `remaining_amount=210`
    - one live BUY reserve remains valid and must stay untouched:
      - `frz_1775043615827_5b71a699dee4` for `ord_1775043615826_a90f870e15ef` with `remaining_amount=4640`
    - the stale local release total is `460`, which would move `1001/USDT` from `frozen=5100` to `frozen=4640`
  - cleanup SQL matches the current schema and behaves as expected inside a rolled-back transaction
- blockers:
  - no blocker for local developer cleanup guidance
  - if the same stale pattern is ever found in non-local or shared data, this doc-only task is not enough and a separate productized reconciliation task will be needed
- next:
  - hand back the audit helper and runbook to commander
  - recommend `local scriptable cleanup` for reused developer DBs
  - recommend a later reconciliation task only if the stale-freeze pattern appears outside disposable local environments or if future audits show `balance_minus_active <> 0`
