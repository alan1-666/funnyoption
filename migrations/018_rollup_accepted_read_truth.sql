BEGIN;

CREATE TABLE IF NOT EXISTS rollup_accepted_balances (
    batch_id    BIGINT NOT NULL,
    account_id  BIGINT NOT NULL,
    asset       VARCHAR(64) NOT NULL,
    available   BIGINT NOT NULL DEFAULT 0,
    frozen      BIGINT NOT NULL DEFAULT 0,
    sequence_no BIGINT NOT NULL DEFAULT 0,
    created_at  BIGINT NOT NULL DEFAULT 0,
    updated_at  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, asset)
);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_balances_batch
    ON rollup_accepted_balances (batch_id);

CREATE TABLE IF NOT EXISTS rollup_accepted_positions (
    batch_id          BIGINT NOT NULL,
    account_id        BIGINT NOT NULL,
    market_id         BIGINT NOT NULL,
    outcome           VARCHAR(16) NOT NULL,
    position_asset    VARCHAR(128) NOT NULL,
    quantity          BIGINT NOT NULL DEFAULT 0,
    settled_quantity  BIGINT NOT NULL DEFAULT 0,
    settlement_status VARCHAR(32) NOT NULL DEFAULT 'OPEN',
    sequence_no       BIGINT NOT NULL DEFAULT 0,
    created_at        BIGINT NOT NULL DEFAULT 0,
    updated_at        BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, market_id, outcome)
);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_positions_batch
    ON rollup_accepted_positions (batch_id);

CREATE TABLE IF NOT EXISTS rollup_accepted_payouts (
    event_id          VARCHAR(96) PRIMARY KEY,
    batch_id          BIGINT NOT NULL,
    market_id         BIGINT NOT NULL,
    user_id           BIGINT NOT NULL,
    winning_outcome   VARCHAR(16) NOT NULL,
    position_asset    VARCHAR(128) NOT NULL,
    settled_quantity  BIGINT NOT NULL DEFAULT 0,
    payout_asset      VARCHAR(64) NOT NULL,
    payout_amount     BIGINT NOT NULL DEFAULT 0,
    status            VARCHAR(32) NOT NULL DEFAULT 'COMPLETED',
    created_at        BIGINT NOT NULL DEFAULT 0,
    updated_at        BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_payouts_user
    ON rollup_accepted_payouts (user_id, batch_id DESC);

CREATE INDEX IF NOT EXISTS idx_rollup_accepted_payouts_market
    ON rollup_accepted_payouts (market_id, batch_id DESC);

COMMIT;
