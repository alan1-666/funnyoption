BEGIN;

CREATE TABLE IF NOT EXISTS account_balance_events (
    id              BIGSERIAL PRIMARY KEY,
    event_type      VARCHAR(32) NOT NULL,
    ref_id          VARCHAR(64) NOT NULL,
    user_id         BIGINT NOT NULL,
    asset           VARCHAR(128) NOT NULL,
    direction       VARCHAR(16) NOT NULL,
    amount          BIGINT NOT NULL,
    created_at      BIGINT NOT NULL DEFAULT 0,
    UNIQUE (event_type, ref_id)
);

CREATE INDEX IF NOT EXISTS idx_account_balance_events_user_asset
    ON account_balance_events(user_id, asset, created_at DESC);

COMMIT;
