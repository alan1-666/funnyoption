# Local `chain_deposits` Schema Repair

This runbook is only for reused local PostgreSQL databases.

It does **not** change order flow, session semantics, or listener behavior.
It exists to reconcile one legacy schema drift in `chain_deposits` without
depending on runtime compatibility assumptions.

## Repo truth

Current repo DDL expects:

- `chain_deposits.deposit_id = VARCHAR(64)`
- `chain_deposits.tx_hash = VARCHAR(128)`

That shape comes from:

- [`migrations/003_wallet_sessions_and_deposits.sql`](/Users/zhangza/code/funnyoption/backend/migrations/003_wallet_sessions_and_deposits.sql)
- [`migrations/010_chain_deposits_tx_hash_width_repair.sql`](/Users/zhangza/code/funnyoption/backend/migrations/010_chain_deposits_tx_hash_width_repair.sql)

## Observed local drift

`TASK-CHAIN-002` observed one reused local database that still enforced:

- `chain_deposits.deposit_id = VARCHAR(64)`
- `chain_deposits.tx_hash = VARCHAR(64)`

That is why the listener-driven local proof stayed alive only after storing:

- deterministic `deposit_id` values that fit inside `VARCHAR(64)`
- normalized transaction hashes as lowercase hex without the `0x` prefix, which
  also fit inside `VARCHAR(64)`

The only repo-vs-schema drift that needs repair is `tx_hash`.
`deposit_id` is intentionally still `VARCHAR(64)` today.

## Scope and non-goals

Use this runbook only when:

- the database is a reused local dev database
- `information_schema` still reports `chain_deposits.tx_hash = 64`
- `chain_deposits.deposit_id` already reports `64`

Stop and do **not** apply this runbook if:

- `deposit_id` is not `64`
- `tx_hash` is neither `64` nor `128`
- you are operating on staging or production
- the table was manually customized beyond the width drift above

## Preflight

Load the local DSN first:

```bash
cd /Users/zhangza/code/funnyoption
set -a
source .env.local
set +a
```

Inspect the live widths:

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -At -F $'\t' -c "
SELECT
  column_name,
  character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'chain_deposits'
  AND column_name IN ('deposit_id', 'tx_hash')
ORDER BY column_name;
"
```

Expected results:

- healthy repo shape:
  - `deposit_id    64`
  - `tx_hash       128`
- legacy local drift this runbook repairs:
  - `deposit_id    64`
  - `tx_hash       64`

Take a schema-only snapshot before changing anything:

```bash
pg_dump "$FUNNYOPTION_POSTGRES_DSN" -s -t chain_deposits > /tmp/funnyoption-chain_deposits-before.sql
```

## Dry run

This proves the width change is valid and rollback-safe before the real apply:

```bash
cat <<'SQL' | psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1
BEGIN;

SELECT
  column_name,
  character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'chain_deposits'
  AND column_name IN ('deposit_id', 'tx_hash')
ORDER BY column_name;

ALTER TABLE chain_deposits
  ALTER COLUMN tx_hash TYPE VARCHAR(128);

SELECT
  column_name,
  character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'chain_deposits'
  AND column_name IN ('deposit_id', 'tx_hash')
ORDER BY column_name;

ROLLBACK;
SQL
```

The first query should show `tx_hash = 64`, the second should show
`tx_hash = 128`, and the rollback should leave the live schema unchanged.

## Apply

Apply the repo migration once the dry run looks correct:

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1 -f \
  /Users/zhangza/code/funnyoption/backend/migrations/010_chain_deposits_tx_hash_width_repair.sql
```

This migration is intentionally narrow and idempotent:

- on a drifted local DB, it widens `tx_hash` from `64` to `128`
- on a healthy local DB, re-running it is a no-op with the same final shape

## Verify after apply

Check the live schema again:

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -At -F $'\t' -c "
SELECT
  column_name,
  character_maximum_length
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'chain_deposits'
  AND column_name IN ('deposit_id', 'tx_hash')
ORDER BY column_name;
"
```

Then re-run chain validation:

```bash
cd /Users/zhangza/code/funnyoption
go test ./internal/chain/...
```

Optional local proof rerun:

```bash
cd /Users/zhangza/code/funnyoption
set -a
source .env.local
set +a
go run ./cmd/local-lifecycle
```

Because the listener still stores normalized 64-character transaction hashes,
this repair should not break the listener-driven local deposit proof.

## Rollback-safe boundary

The safe rollback condition is strict: only shrink `tx_hash` back to `64` if
every stored value still fits inside `64` characters.

Check that boundary first:

```bash
psql "$FUNNYOPTION_POSTGRES_DSN" -At -c "
SELECT COALESCE(MAX(length(tx_hash)), 0)
FROM chain_deposits;
"
```

If the result is greater than `64`, stop. A shrink rollback is no longer safe.

If the result is `64` or lower and you are still on a local dev database, the
rollback-safe command is:

```bash
cat <<'SQL' | psql "$FUNNYOPTION_POSTGRES_DSN" -v ON_ERROR_STOP=1
BEGIN;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM chain_deposits
    WHERE length(tx_hash) > 64
  ) THEN
    RAISE EXCEPTION 'rollback unsafe: tx_hash values longer than 64 exist';
  END IF;
END
$$;

ALTER TABLE chain_deposits
  ALTER COLUMN tx_hash TYPE VARCHAR(64);

COMMIT;
SQL
```

Do not treat that rollback as a normal workflow. The preferred fix is to keep
the repo truth of `VARCHAR(128)` in local databases too.
