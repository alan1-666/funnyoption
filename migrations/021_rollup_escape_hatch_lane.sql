BEGIN;

CREATE TABLE IF NOT EXISTS rollup_accepted_escape_roots (
    batch_id BIGINT PRIMARY KEY REFERENCES rollup_accepted_batches(batch_id) ON DELETE CASCADE,
    state_root TEXT NOT NULL,
    collateral_asset TEXT NOT NULL,
    merkle_root TEXT NOT NULL,
    leaf_count BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    anchor_status TEXT NOT NULL DEFAULT 'READY',
    anchor_tx_hash TEXT NOT NULL DEFAULT '',
    anchor_submitted_at BIGINT NOT NULL DEFAULT 0,
    anchored_at BIGINT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    last_error_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_escape_roots_status
    ON rollup_accepted_escape_roots(anchor_status, batch_id DESC);

CREATE TABLE IF NOT EXISTS rollup_accepted_escape_leaves (
    batch_id BIGINT NOT NULL REFERENCES rollup_accepted_escape_roots(batch_id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    wallet_address TEXT NOT NULL,
    collateral_asset TEXT NOT NULL,
    claim_amount BIGINT NOT NULL DEFAULT 0,
    leaf_index BIGINT NOT NULL,
    leaf_hash TEXT NOT NULL,
    proof_hashes JSONB NOT NULL DEFAULT '[]'::jsonb,
    claim_id TEXT NOT NULL,
    claim_status TEXT NOT NULL DEFAULT 'CLAIMABLE',
    claim_tx_hash TEXT NOT NULL DEFAULT '',
    claim_submitted_at BIGINT NOT NULL DEFAULT 0,
    claimed_at BIGINT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    last_error_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (batch_id, account_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_rollup_accepted_escape_leaves_claim
    ON rollup_accepted_escape_leaves(claim_id);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_escape_leaves_wallet
    ON rollup_accepted_escape_leaves(wallet_address, claim_status, batch_id DESC);

COMMIT;
