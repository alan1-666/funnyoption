BEGIN;

-- Reused local databases can carry a legacy chain_deposits.tx_hash width of
-- VARCHAR(64). The repo truth is VARCHAR(128), so widen the column without
-- touching deposit_id semantics.
ALTER TABLE IF EXISTS chain_deposits
    ALTER COLUMN tx_hash TYPE VARCHAR(128);

COMMIT;
