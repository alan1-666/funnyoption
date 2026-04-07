CREATE TABLE IF NOT EXISTS rollup_accepted_withdrawal_roots (
    batch_id       BIGINT PRIMARY KEY,
    merkle_root    TEXT NOT NULL DEFAULT '',
    leaf_count     BIGINT NOT NULL DEFAULT 0,
    created_at     BIGINT NOT NULL DEFAULT 0,
    updated_at     BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS rollup_accepted_withdrawal_leaves (
    batch_id          BIGINT NOT NULL,
    withdrawal_id     TEXT NOT NULL,
    account_id        BIGINT NOT NULL DEFAULT 0,
    wallet_address    TEXT NOT NULL DEFAULT '',
    recipient_address TEXT NOT NULL DEFAULT '',
    amount            BIGINT NOT NULL DEFAULT 0,
    leaf_index        BIGINT NOT NULL DEFAULT 0,
    leaf_hash         TEXT NOT NULL DEFAULT '',
    proof_hashes      JSONB NOT NULL DEFAULT '[]',
    claim_id          TEXT NOT NULL DEFAULT '',
    claim_status      TEXT NOT NULL DEFAULT 'CLAIMABLE',
    claim_tx_hash     TEXT NOT NULL DEFAULT '',
    claim_submitted_at BIGINT NOT NULL DEFAULT 0,
    claimed_at        BIGINT NOT NULL DEFAULT 0,
    last_error        TEXT NOT NULL DEFAULT '',
    last_error_at     BIGINT NOT NULL DEFAULT 0,
    created_at        BIGINT NOT NULL DEFAULT 0,
    updated_at        BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (batch_id, leaf_index)
);

CREATE INDEX IF NOT EXISTS idx_withdrawal_leaves_claim_status
    ON rollup_accepted_withdrawal_leaves (claim_status);
CREATE INDEX IF NOT EXISTS idx_withdrawal_leaves_wallet
    ON rollup_accepted_withdrawal_leaves (wallet_address);
