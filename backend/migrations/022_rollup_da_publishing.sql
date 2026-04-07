ALTER TABLE rollup_shadow_submissions
    ADD COLUMN IF NOT EXISTS publish_calldata TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS publish_tx_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS publish_submitted_at BIGINT NOT NULL DEFAULT 0;
