\pset pager off
\pset null '(null)'

\echo ''
\echo '== Active freeze totals versus account_balances.frozen =='
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
    ab.frozen AS balance_frozen,
    COALESCE(aft.active_freeze_total, 0) AS active_freeze_total,
    ab.frozen - COALESCE(aft.active_freeze_total, 0) AS balance_minus_active
FROM account_balances ab
LEFT JOIN active_freeze_totals aft
    ON aft.user_id = ab.user_id
   AND aft.asset = ab.asset
WHERE ab.frozen <> 0
   OR COALESCE(aft.active_freeze_total, 0) <> 0
ORDER BY ab.user_id, ab.asset;

\echo ''
\echo '== Suspicious terminal orders that still hold ACTIVE freezes =='
SELECT
    o.order_id,
    o.user_id,
    o.market_id,
    o.side,
    o.status AS order_status,
    o.remaining_quantity,
    fr.freeze_id,
    fr.asset AS freeze_asset,
    fr.original_amount,
    fr.remaining_amount AS stuck_amount,
    fr.status AS freeze_status,
    o.updated_at
FROM orders o
JOIN freeze_records fr
    ON fr.freeze_id = o.freeze_id
WHERE fr.status = 'ACTIVE'
  AND fr.remaining_amount > 0
  AND (
      o.status IN ('FILLED', 'CANCELLED', 'REJECTED')
      OR o.remaining_quantity = 0
  )
ORDER BY o.user_id, o.created_at, o.order_id;

\echo ''
\echo '== Pre-fix BUY quote freezes: per-user release totals =='
WITH stale_buy_quote AS (
    SELECT
        o.order_id,
        o.user_id,
        o.market_id,
        o.price,
        o.quantity,
        o.filled_quantity,
        o.remaining_quantity,
        o.status AS order_status,
        fr.freeze_id,
        fr.asset,
        fr.remaining_amount AS suggested_release_amount
    FROM orders o
    JOIN freeze_records fr
        ON fr.freeze_id = o.freeze_id
    WHERE o.side = 'BUY'
      AND fr.asset = 'USDT'
      AND fr.status = 'ACTIVE'
      AND fr.remaining_amount > 0
      AND (
          o.status IN ('FILLED', 'CANCELLED', 'REJECTED')
          OR o.remaining_quantity = 0
      )
)
SELECT
    user_id,
    asset,
    COUNT(*) AS stale_order_count,
    SUM(suggested_release_amount) AS suggested_release_total
FROM stale_buy_quote
GROUP BY user_id, asset
ORDER BY user_id, asset;

\echo ''
\echo '== Pre-fix BUY quote freezes: detailed rows =='
WITH stale_buy_quote AS (
    SELECT
        o.order_id,
        o.user_id,
        o.market_id,
        o.price,
        o.quantity,
        o.filled_quantity,
        o.remaining_quantity,
        o.status AS order_status,
        fr.freeze_id,
        fr.remaining_amount AS suggested_release_amount
    FROM orders o
    JOIN freeze_records fr
        ON fr.freeze_id = o.freeze_id
    WHERE o.side = 'BUY'
      AND fr.asset = 'USDT'
      AND fr.status = 'ACTIVE'
      AND fr.remaining_amount > 0
      AND (
          o.status IN ('FILLED', 'CANCELLED', 'REJECTED')
          OR o.remaining_quantity = 0
      )
)
SELECT
    order_id,
    freeze_id,
    user_id,
    market_id,
    price,
    quantity,
    filled_quantity,
    remaining_quantity,
    order_status,
    suggested_release_amount
FROM stale_buy_quote
ORDER BY user_id, market_id, order_id;

\echo ''
\echo '== Live non-terminal BUY freezes that should remain untouched =='
SELECT
    o.order_id,
    o.user_id,
    o.market_id,
    o.price,
    o.quantity,
    o.filled_quantity,
    o.remaining_quantity,
    o.status AS order_status,
    fr.freeze_id,
    fr.remaining_amount AS expected_live_reserve
FROM orders o
JOIN freeze_records fr
    ON fr.freeze_id = o.freeze_id
WHERE o.side = 'BUY'
  AND fr.asset = 'USDT'
  AND fr.status = 'ACTIVE'
  AND fr.remaining_amount > 0
  AND o.remaining_quantity > 0
  AND o.status IN ('NEW', 'PARTIALLY_FILLED')
ORDER BY o.user_id, o.created_at, o.order_id;
