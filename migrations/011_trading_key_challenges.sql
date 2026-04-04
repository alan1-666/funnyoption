BEGIN;

CREATE TABLE IF NOT EXISTS trading_key_challenges (
    challenge_id    VARCHAR(64) PRIMARY KEY,
    wallet_address  VARCHAR(64) NOT NULL,
    chain_id        BIGINT NOT NULL DEFAULT 0,
    vault_address   VARCHAR(128) NOT NULL DEFAULT '',
    challenge       VARCHAR(64) NOT NULL,
    expires_at      BIGINT NOT NULL DEFAULT 0,
    consumed_at     BIGINT NOT NULL DEFAULT 0,
    created_at      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0,
    UNIQUE (wallet_address, chain_id, vault_address, challenge)
);

CREATE INDEX IF NOT EXISTS idx_trading_key_challenges_wallet_scope
    ON trading_key_challenges(wallet_address, chain_id, vault_address, consumed_at, expires_at DESC);

COMMIT;
