BEGIN;

ALTER TABLE chain_transactions
    ADD COLUMN IF NOT EXISTS error_message VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE chain_transactions
    ADD COLUMN IF NOT EXISTS attempt_count BIGINT NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS uk_chain_transactions_biz_ref
    ON chain_transactions(biz_type, ref_id);

COMMIT;
