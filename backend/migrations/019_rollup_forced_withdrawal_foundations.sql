BEGIN;

CREATE TABLE IF NOT EXISTS rollup_forced_withdrawal_requests (
    request_id BIGINT PRIMARY KEY,
    wallet_address TEXT NOT NULL,
    recipient_address TEXT NOT NULL,
    amount BIGINT NOT NULL,
    requested_at BIGINT NOT NULL,
    deadline_at BIGINT NOT NULL,
    satisfied_claim_id TEXT NOT NULL DEFAULT '',
    satisfied_at BIGINT NOT NULL DEFAULT 0,
    frozen_at BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rollup_forced_withdrawal_requests_wallet
    ON rollup_forced_withdrawal_requests (wallet_address, request_id DESC);

CREATE INDEX IF NOT EXISTS idx_rollup_forced_withdrawal_requests_status
    ON rollup_forced_withdrawal_requests (status, deadline_at ASC);

CREATE TABLE IF NOT EXISTS rollup_freeze_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    frozen BOOLEAN NOT NULL DEFAULT FALSE,
    frozen_at BIGINT NOT NULL DEFAULT 0,
    request_id BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
);

INSERT INTO rollup_freeze_state (id, frozen, frozen_at, request_id, updated_at)
VALUES (TRUE, FALSE, 0, 0, 0)
ON CONFLICT (id) DO NOTHING;

COMMIT;
