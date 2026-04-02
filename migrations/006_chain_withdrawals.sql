BEGIN;

CREATE TABLE IF NOT EXISTS chain_withdrawals (
    withdrawal_id       VARCHAR(66) PRIMARY KEY,
    user_id             BIGINT NOT NULL DEFAULT 0,
    wallet_address      VARCHAR(64) NOT NULL,
    recipient_address   VARCHAR(64) NOT NULL,
    vault_address       VARCHAR(128) NOT NULL DEFAULT '',
    asset               VARCHAR(32) NOT NULL,
    amount              BIGINT NOT NULL DEFAULT 0,
    chain_name          VARCHAR(32) NOT NULL DEFAULT 'bsc',
    network_name        VARCHAR(32) NOT NULL DEFAULT 'testnet',
    tx_hash             VARCHAR(128) NOT NULL,
    log_index           BIGINT NOT NULL DEFAULT 0,
    block_number        BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(32) NOT NULL DEFAULT 'QUEUED',
    debited_at          BIGINT NOT NULL DEFAULT 0,
    created_at          BIGINT NOT NULL DEFAULT 0,
    updated_at          BIGINT NOT NULL DEFAULT 0,
    UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_chain_withdrawals_user_created_at
    ON chain_withdrawals(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chain_withdrawals_wallet_created_at
    ON chain_withdrawals(wallet_address, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chain_withdrawals_status_created_at
    ON chain_withdrawals(status, created_at DESC);

COMMIT;
