BEGIN;

ALTER TABLE wallet_sessions
    ADD COLUMN IF NOT EXISTS vault_address VARCHAR(128);

UPDATE wallet_sessions
SET vault_address = ''
WHERE vault_address IS NULL;

ALTER TABLE wallet_sessions
    ALTER COLUMN vault_address SET DEFAULT '';

ALTER TABLE wallet_sessions
    ALTER COLUMN vault_address SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wallet_sessions_wallet_chain_vault_status_created_at
    ON wallet_sessions(wallet_address, chain_id, vault_address, status, created_at DESC);

COMMIT;
