BEGIN;

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS matched_withdrawal_id TEXT NOT NULL DEFAULT '';

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS matched_claim_id TEXT NOT NULL DEFAULT '';

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS satisfaction_status TEXT NOT NULL DEFAULT 'NONE';

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS satisfaction_tx_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS satisfaction_submitted_at BIGINT NOT NULL DEFAULT 0;

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS satisfaction_last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE rollup_forced_withdrawal_requests
    ADD COLUMN IF NOT EXISTS satisfaction_last_error_at BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_rollup_forced_withdrawal_requests_satisfaction
    ON rollup_forced_withdrawal_requests (status, satisfaction_status, request_id ASC);

COMMIT;
