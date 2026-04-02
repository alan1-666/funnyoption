\pset pager off

\echo ''
\echo '== Candidate stale BUY quote freezes =='
DROP TABLE IF EXISTS stale_buy_quote_candidates;
CREATE TEMP TABLE stale_buy_quote_candidates ON COMMIT DROP AS
SELECT
    fr.freeze_id,
    fr.user_id,
    fr.asset,
    fr.remaining_amount
FROM freeze_records fr
JOIN orders o
    ON o.freeze_id = fr.freeze_id
WHERE o.side = 'BUY'
  AND fr.asset = 'USDT'
  AND fr.status = 'ACTIVE'
  AND fr.remaining_amount > 0
  AND (
      o.status IN ('FILLED', 'CANCELLED', 'REJECTED')
      OR o.remaining_quantity = 0
  );

TABLE stale_buy_quote_candidates;

\echo ''
\echo '== Releasing stale amounts back into account_balances =='
WITH release_totals AS (
    SELECT
        user_id,
        asset,
        SUM(remaining_amount) AS release_total
    FROM stale_buy_quote_candidates
    GROUP BY user_id, asset
)
UPDATE account_balances ab
SET available = ab.available + rt.release_total,
    frozen = ab.frozen - rt.release_total,
    updated_at = EXTRACT(EPOCH FROM clock_timestamp())::BIGINT
FROM release_totals rt
WHERE ab.user_id = rt.user_id
  AND ab.asset = rt.asset
RETURNING ab.user_id, ab.asset, rt.release_total, ab.available, ab.frozen, ab.updated_at;

\echo ''
\echo '== Marking stale freeze_records as RELEASED with remaining_amount = 0 =='
UPDATE freeze_records fr
SET status = 'RELEASED',
    remaining_amount = 0,
    updated_at = EXTRACT(EPOCH FROM clock_timestamp())::BIGINT
FROM stale_buy_quote_candidates c
WHERE fr.freeze_id = c.freeze_id
RETURNING fr.freeze_id, fr.user_id, fr.asset, fr.remaining_amount, fr.status, fr.updated_at;

\echo ''
\echo '== Post-cleanup balance check =='
WITH active_freeze_totals AS (
    SELECT
        user_id,
        asset,
        SUM(remaining_amount) AS active_freeze_total
    FROM freeze_records
    WHERE status = 'ACTIVE'
    GROUP BY user_id, asset
)
SELECT
    ab.user_id,
    ab.asset,
    ab.available,
    ab.frozen,
    COALESCE(aft.active_freeze_total, 0) AS active_freeze_total,
    ab.frozen - COALESCE(aft.active_freeze_total, 0) AS balance_minus_active
FROM account_balances ab
LEFT JOIN active_freeze_totals aft
    ON aft.user_id = ab.user_id
   AND aft.asset = ab.asset
WHERE ab.frozen <> 0
   OR COALESCE(aft.active_freeze_total, 0) <> 0
ORDER BY ab.user_id, ab.asset;
