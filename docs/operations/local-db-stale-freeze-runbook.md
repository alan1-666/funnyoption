# Local DB Stale Freeze Runbook

This runbook is for reused local developer databases only.
It is not a production reconciliation design and must not be applied to shared or live environments.

## When to use it

Run this audit when a local PostgreSQL volume has been reused across older off-chain builds and BUY-side quote freezes may have survived after the recent account fix.

Typical symptom:

- `account_balances.frozen` stays too high for `USDT`
- the user has `FILLED` or `CANCELLED` BUY orders whose `freeze_records.status` is still `ACTIVE`

## Audit command

From the repo root:

```bash
./scripts/audit_stale_freezes.sh
```

The helper reads `FUNNYOPTION_POSTGRES_DSN` from `.env.local` by default and executes [`docs/sql/local_stale_freeze_audit.sql`](/Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_audit.sql).

## How to read the output

The audit prints five sections:

1. `Active freeze totals versus account_balances.frozen`
2. `Suspicious terminal orders that still hold ACTIVE freezes`
3. `Pre-fix BUY quote freezes: per-user release totals`
4. `Pre-fix BUY quote freezes: detailed rows`
5. `Live non-terminal BUY freezes that should remain untouched`

Treat a row as stale local residue only when all of these are true:

- the order is `FILLED`, `CANCELLED`, or otherwise has `remaining_quantity = 0`
- the freeze row is still `ACTIVE`
- `remaining_amount > 0`
- the asset is `USDT` for the pre-fix BUY reserve bug

Do not clean rows from the `Live non-terminal BUY freezes` section. Those are still backing an open order.

## Current local evidence

Audit performed on `2026-04-01` against the current local developer DB found:

- `account_balances`: user `1001` / `USDT` has `frozen = 5100`
- legitimate open reserve: `ord_1775043615826_a90f870e15ef` keeps `4640` active for a `NEW` BUY order
- stale residue: two terminal BUY orders still keep `ACTIVE` quote freezes:
  - `ord_1775043614954_a16c576344c5` / `frz_1775043614954_105f757f1cf4` with `250`
  - `ord_1775043615335_47037c23808b` / `frz_1775043615335_4502d5cd30bb` with `210`
- suggested local release total: `460`

That means the current reused local DB does contain stale pre-fix BUY freezes.

## Safe local-only cleanup flow

1. Stop the local stack so nothing writes while you repair the reused DB.

```bash
./scripts/dev-down.sh
```

2. Re-run the audit and confirm the affected asset has `balance_minus_active = 0`.

Why this matters:
The cleanup SQL assumes `account_balances.frozen` is fully explained by currently active freezes.
If the delta is non-zero, stop and open a separate reconciliation task instead of using this runbook.

3. Dry-run the cleanup inside a rollback and inspect the candidate rows.

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 \
  -c "BEGIN" \
  -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql \
  -c "ROLLBACK"
```

4. If the dry-run only shows stale terminal BUY quote freezes, execute the same file inside a real transaction.

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 \
  -c "BEGIN" \
  -f /Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql \
  -c "COMMIT"
```

5. Run the audit again.

Expected post-cleanup result:

- the stale BUY rows disappear from the suspicious sections
- `1001 / USDT frozen` drops from `5100` to `4640`
- the remaining `4640` is still backed by the single open BUY order
- each released `freeze_records` row ends as `status = RELEASED` and `remaining_amount = 0`

## What the cleanup SQL does

[`docs/sql/local_stale_freeze_cleanup.sql`](/Users/zhangza/code/funnyoption/docs/sql/local_stale_freeze_cleanup.sql):

- selects only terminal BUY orders with `ACTIVE` `USDT` freezes
- sums those `remaining_amount` values by `user_id + asset`
- moves that amount from `account_balances.frozen` back to `account_balances.available`
- marks the matching `freeze_records` as `RELEASED`
- zeroes `freeze_records.remaining_amount` on those released rows

It does not touch runtime code, trades, positions, or ledger rows.

## When not to use this runbook

Do not use this runbook when:

- the environment is shared, staging, or production
- the audit shows non-zero `balance_minus_active`
- the suspicious rows are not limited to stale terminal BUY quote freezes
- you need immutable-evidence reconciliation against ledger or chain state

Those cases need a separate productized reconciliation task.
