BEGIN;

CREATE TABLE IF NOT EXISTS rollup_accepted_batches (
    batch_id BIGINT PRIMARY KEY,
    submission_id TEXT NOT NULL UNIQUE,
    encoding_version TEXT NOT NULL,
    first_sequence_no BIGINT NOT NULL,
    last_sequence_no BIGINT NOT NULL,
    entry_count INT NOT NULL,
    batch_data_hash TEXT NOT NULL,
    prev_state_root TEXT NOT NULL,
    balances_root TEXT NOT NULL,
    orders_root TEXT NOT NULL,
    positions_funding_root TEXT NOT NULL,
    withdrawals_root TEXT NOT NULL,
    next_state_root TEXT NOT NULL,
    record_tx_hash TEXT NOT NULL DEFAULT '',
    accept_tx_hash TEXT NOT NULL DEFAULT '',
    accepted_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS rollup_accepted_withdrawals (
    withdrawal_id TEXT PRIMARY KEY,
    batch_id BIGINT NOT NULL REFERENCES rollup_accepted_batches(batch_id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    wallet_address TEXT NOT NULL,
    recipient_address TEXT NOT NULL,
    vault_address TEXT NOT NULL,
    asset TEXT NOT NULL,
    amount BIGINT NOT NULL,
    lane TEXT NOT NULL,
    chain_name TEXT NOT NULL,
    network_name TEXT NOT NULL,
    request_sequence BIGINT NOT NULL,
    claim_id TEXT NOT NULL,
    claim_status TEXT NOT NULL,
    claim_tx_hash TEXT NOT NULL DEFAULT '',
    claim_submitted_at BIGINT NOT NULL DEFAULT 0,
    claimed_at BIGINT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    last_error_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_rollup_accepted_withdrawals_claim
    ON rollup_accepted_withdrawals(claim_id);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_withdrawals_batch
    ON rollup_accepted_withdrawals(batch_id);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_withdrawals_claim_status
    ON rollup_accepted_withdrawals(claim_status, updated_at DESC);

COMMIT;
