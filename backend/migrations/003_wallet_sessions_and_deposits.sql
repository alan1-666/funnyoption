BEGIN;

CREATE TABLE IF NOT EXISTS wallet_sessions (
    session_id          VARCHAR(64) PRIMARY KEY,
    user_id             BIGINT NOT NULL DEFAULT 0,
    wallet_address      VARCHAR(64) NOT NULL,
    session_public_key  VARCHAR(256) NOT NULL,
    scope               VARCHAR(32) NOT NULL DEFAULT 'TRADE',
    chain_id            BIGINT NOT NULL DEFAULT 0,
    session_nonce       VARCHAR(64) NOT NULL DEFAULT '',
    last_order_nonce    BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    issued_at           BIGINT NOT NULL DEFAULT 0,
    expires_at          BIGINT NOT NULL DEFAULT 0,
    revoked_at          BIGINT NOT NULL DEFAULT 0,
    created_at          BIGINT NOT NULL DEFAULT 0,
    updated_at          BIGINT NOT NULL DEFAULT 0,
    UNIQUE (wallet_address, session_public_key)
);

CREATE INDEX IF NOT EXISTS idx_wallet_sessions_wallet_status
    ON wallet_sessions(wallet_address, status);

CREATE INDEX IF NOT EXISTS idx_wallet_sessions_user_status
    ON wallet_sessions(user_id, status);

CREATE TABLE IF NOT EXISTS chain_deposits (
    deposit_id          VARCHAR(64) PRIMARY KEY,
    user_id             BIGINT NOT NULL DEFAULT 0,
    wallet_address      VARCHAR(64) NOT NULL,
    vault_address       VARCHAR(128) NOT NULL DEFAULT '',
    asset               VARCHAR(32) NOT NULL,
    amount              BIGINT NOT NULL DEFAULT 0,
    chain_name          VARCHAR(32) NOT NULL DEFAULT 'bsc',
    network_name        VARCHAR(32) NOT NULL DEFAULT 'testnet',
    tx_hash             VARCHAR(128) NOT NULL,
    log_index           BIGINT NOT NULL DEFAULT 0,
    block_number        BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(32) NOT NULL DEFAULT 'CONFIRMED',
    credited_at         BIGINT NOT NULL DEFAULT 0,
    created_at          BIGINT NOT NULL DEFAULT 0,
    updated_at          BIGINT NOT NULL DEFAULT 0,
    UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_chain_deposits_user_created_at
    ON chain_deposits(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chain_deposits_wallet_created_at
    ON chain_deposits(wallet_address, created_at DESC);

COMMIT;
