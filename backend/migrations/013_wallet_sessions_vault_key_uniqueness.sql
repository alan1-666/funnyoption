BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'wallet_sessions'::regclass
          AND conname = 'wallet_sessions_wallet_address_session_public_key_key'
    ) THEN
        ALTER TABLE wallet_sessions
            DROP CONSTRAINT wallet_sessions_wallet_address_session_public_key_key;
    END IF;
END $$;

DROP INDEX IF EXISTS wallet_sessions_wallet_address_session_public_key_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'wallet_sessions'::regclass
          AND conname = 'wallet_sessions_wallet_chain_vault_public_key_key'
    ) THEN
        ALTER TABLE wallet_sessions
            ADD CONSTRAINT wallet_sessions_wallet_chain_vault_public_key_key
            UNIQUE (wallet_address, chain_id, vault_address, session_public_key);
    END IF;
END $$;

COMMIT;
