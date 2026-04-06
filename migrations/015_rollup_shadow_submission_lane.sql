BEGIN;

CREATE TABLE IF NOT EXISTS rollup_shadow_submissions (
    submission_id        VARCHAR(64) PRIMARY KEY,
    batch_id             BIGINT NOT NULL UNIQUE REFERENCES rollup_shadow_batches(batch_id) ON DELETE CASCADE,
    encoding_version     VARCHAR(64) NOT NULL DEFAULT 'shadow-submit-v1',
    status               VARCHAR(32) NOT NULL DEFAULT 'READY',
    batch_data_hash      VARCHAR(66) NOT NULL DEFAULT '',
    next_state_root      VARCHAR(66) NOT NULL DEFAULT '',
    auth_proof_hash      VARCHAR(66) NOT NULL DEFAULT '',
    verifier_gate_hash   VARCHAR(66) NOT NULL DEFAULT '',
    record_calldata      TEXT NOT NULL,
    accept_calldata      TEXT NOT NULL,
    submission_data      JSONB NOT NULL DEFAULT '{}'::jsonb,
    submission_hash      VARCHAR(64) NOT NULL DEFAULT '',
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_rollup_shadow_submissions_status_batch
    ON rollup_shadow_submissions(status, batch_id);

COMMIT;
