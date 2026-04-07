BEGIN;

CREATE TABLE IF NOT EXISTS chain_listener_cursors (
    chain_name      VARCHAR(32) NOT NULL DEFAULT 'bsc',
    network_name    VARCHAR(32) NOT NULL DEFAULT 'testnet',
    vault_address   VARCHAR(128) NOT NULL DEFAULT '',
    next_block      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chain_name, network_name, vault_address)
);

CREATE INDEX IF NOT EXISTS idx_chain_listener_cursors_updated_at
    ON chain_listener_cursors(updated_at DESC);

COMMIT;
