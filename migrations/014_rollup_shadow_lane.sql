BEGIN;

CREATE TABLE IF NOT EXISTS rollup_shadow_journal_entries (
    sequence_no            BIGSERIAL PRIMARY KEY,
    entry_id               VARCHAR(64) NOT NULL UNIQUE,
    entry_type             VARCHAR(64) NOT NULL,
    source_type            VARCHAR(64) NOT NULL,
    source_ref             VARCHAR(128) NOT NULL,
    occurred_at_millis     BIGINT NOT NULL DEFAULT 0,
    payload                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at             BIGINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_rollup_shadow_journal_source
    ON rollup_shadow_journal_entries(entry_type, source_type, source_ref);

CREATE INDEX IF NOT EXISTS idx_rollup_shadow_journal_sequence
    ON rollup_shadow_journal_entries(sequence_no);

CREATE INDEX IF NOT EXISTS idx_rollup_shadow_journal_type_sequence
    ON rollup_shadow_journal_entries(entry_type, sequence_no);

CREATE TABLE IF NOT EXISTS rollup_shadow_batches (
    batch_id                BIGSERIAL PRIMARY KEY,
    encoding_version        VARCHAR(64) NOT NULL DEFAULT 'shadow-batch-v1',
    first_sequence_no       BIGINT NOT NULL,
    last_sequence_no        BIGINT NOT NULL,
    entry_count             INTEGER NOT NULL DEFAULT 0,
    input_data              TEXT NOT NULL,
    input_hash              VARCHAR(64) NOT NULL,
    prev_state_root         VARCHAR(64) NOT NULL DEFAULT '',
    balances_root           VARCHAR(64) NOT NULL DEFAULT '',
    orders_root             VARCHAR(64) NOT NULL DEFAULT '',
    positions_funding_root  VARCHAR(64) NOT NULL DEFAULT '',
    withdrawals_root        VARCHAR(64) NOT NULL DEFAULT '',
    state_root              VARCHAR(64) NOT NULL DEFAULT '',
    created_at              BIGINT NOT NULL DEFAULT 0,
    UNIQUE (first_sequence_no, last_sequence_no)
);

CREATE INDEX IF NOT EXISTS idx_rollup_shadow_batches_last_sequence
    ON rollup_shadow_batches(last_sequence_no);

COMMIT;
