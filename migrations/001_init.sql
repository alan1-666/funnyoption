-- funnyoption core schema
-- database: PostgreSQL

BEGIN;

CREATE TABLE IF NOT EXISTS markets (
    market_id            BIGINT PRIMARY KEY,
    title                VARCHAR(255) NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    collateral_asset     VARCHAR(32) NOT NULL DEFAULT 'USDT',
    status               VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    open_at              BIGINT NOT NULL DEFAULT 0,
    close_at             BIGINT NOT NULL DEFAULT 0,
    resolve_at           BIGINT NOT NULL DEFAULT 0,
    resolved_outcome     VARCHAR(32) NOT NULL DEFAULT '',
    created_by           BIGINT NOT NULL DEFAULT 0,
    metadata             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_markets_status_open_at
    ON markets(status, open_at);

CREATE TABLE IF NOT EXISTS market_resolutions (
    id                   BIGSERIAL PRIMARY KEY,
    market_id            BIGINT NOT NULL REFERENCES markets(market_id),
    status               VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    resolved_outcome     VARCHAR(32) NOT NULL DEFAULT '',
    resolver_type        VARCHAR(32) NOT NULL DEFAULT 'ADMIN',
    resolver_ref         VARCHAR(128) NOT NULL DEFAULT '',
    evidence             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0,
    UNIQUE (market_id)
);

CREATE TABLE IF NOT EXISTS orders (
    order_id             VARCHAR(64) PRIMARY KEY,
    client_order_id      VARCHAR(64) NOT NULL DEFAULT '',
    command_id           VARCHAR(64) NOT NULL DEFAULT '',
    user_id              BIGINT NOT NULL,
    market_id            BIGINT NOT NULL REFERENCES markets(market_id),
    outcome              VARCHAR(32) NOT NULL,
    side                 VARCHAR(16) NOT NULL,
    order_type           VARCHAR(16) NOT NULL,
    time_in_force        VARCHAR(16) NOT NULL,
    collateral_asset     VARCHAR(32) NOT NULL DEFAULT 'USDT',
    freeze_id            VARCHAR(64) NOT NULL DEFAULT '',
    freeze_asset         VARCHAR(128) NOT NULL DEFAULT '',
    freeze_amount        BIGINT NOT NULL DEFAULT 0,
    price                BIGINT NOT NULL DEFAULT 0,
    quantity             BIGINT NOT NULL DEFAULT 0,
    filled_quantity      BIGINT NOT NULL DEFAULT 0,
    remaining_quantity   BIGINT NOT NULL DEFAULT 0,
    status               VARCHAR(32) NOT NULL DEFAULT 'NEW',
    cancel_reason        VARCHAR(64) NOT NULL DEFAULT '',
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id_created_at
    ON orders(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_orders_market_id_outcome_status
    ON orders(market_id, outcome, status);

CREATE INDEX IF NOT EXISTS idx_orders_freeze_id
    ON orders(freeze_id);

CREATE TABLE IF NOT EXISTS trades (
    trade_id             VARCHAR(64) PRIMARY KEY,
    sequence_no          BIGINT NOT NULL UNIQUE,
    market_id            BIGINT NOT NULL REFERENCES markets(market_id),
    outcome              VARCHAR(32) NOT NULL,
    collateral_asset     VARCHAR(32) NOT NULL DEFAULT 'USDT',
    price                BIGINT NOT NULL,
    quantity             BIGINT NOT NULL,
    taker_order_id       VARCHAR(64) NOT NULL REFERENCES orders(order_id),
    maker_order_id       VARCHAR(64) NOT NULL REFERENCES orders(order_id),
    taker_user_id        BIGINT NOT NULL,
    maker_user_id        BIGINT NOT NULL,
    taker_side           VARCHAR(16) NOT NULL,
    maker_side           VARCHAR(16) NOT NULL,
    occurred_at          BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_trades_market_id_sequence_no
    ON trades(market_id, sequence_no);

CREATE INDEX IF NOT EXISTS idx_trades_taker_user_id
    ON trades(taker_user_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_trades_maker_user_id
    ON trades(maker_user_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS positions (
    market_id            BIGINT NOT NULL REFERENCES markets(market_id),
    user_id              BIGINT NOT NULL,
    outcome              VARCHAR(32) NOT NULL,
    position_asset       VARCHAR(128) NOT NULL,
    quantity             BIGINT NOT NULL DEFAULT 0,
    settled_quantity     BIGINT NOT NULL DEFAULT 0,
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (market_id, user_id, outcome)
);

CREATE INDEX IF NOT EXISTS idx_positions_user_id
    ON positions(user_id, market_id);

CREATE TABLE IF NOT EXISTS account_balances (
    user_id              BIGINT NOT NULL,
    asset                VARCHAR(128) NOT NULL,
    available            BIGINT NOT NULL DEFAULT 0,
    frozen               BIGINT NOT NULL DEFAULT 0,
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, asset)
);

CREATE INDEX IF NOT EXISTS idx_account_balances_asset
    ON account_balances(asset, user_id);

CREATE TABLE IF NOT EXISTS freeze_records (
    freeze_id            VARCHAR(64) PRIMARY KEY,
    user_id              BIGINT NOT NULL,
    asset                VARCHAR(128) NOT NULL,
    ref_type             VARCHAR(32) NOT NULL,
    ref_id               VARCHAR(64) NOT NULL,
    original_amount      BIGINT NOT NULL,
    remaining_amount     BIGINT NOT NULL,
    status               VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_freeze_records_user_id_status
    ON freeze_records(user_id, status);

CREATE INDEX IF NOT EXISTS idx_freeze_records_ref
    ON freeze_records(ref_type, ref_id);

CREATE TABLE IF NOT EXISTS ledger_entries (
    entry_id             VARCHAR(64) PRIMARY KEY,
    biz_type             VARCHAR(32) NOT NULL,
    ref_id               VARCHAR(64) NOT NULL,
    status               VARCHAR(32) NOT NULL DEFAULT 'CONFIRMED',
    created_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_ref_id
    ON ledger_entries(ref_id);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_biz_type_created_at
    ON ledger_entries(biz_type, created_at DESC);

CREATE TABLE IF NOT EXISTS ledger_postings (
    id                   BIGSERIAL PRIMARY KEY,
    entry_id             VARCHAR(64) NOT NULL REFERENCES ledger_entries(entry_id),
    account_ref          VARCHAR(128) NOT NULL,
    asset                VARCHAR(128) NOT NULL,
    direction            VARCHAR(16) NOT NULL,
    amount               BIGINT NOT NULL,
    created_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ledger_postings_entry_id
    ON ledger_postings(entry_id);

CREATE INDEX IF NOT EXISTS idx_ledger_postings_account_asset
    ON ledger_postings(account_ref, asset);

CREATE TABLE IF NOT EXISTS settlement_payouts (
    event_id             VARCHAR(64) PRIMARY KEY,
    market_id            BIGINT NOT NULL REFERENCES markets(market_id),
    user_id              BIGINT NOT NULL,
    winning_outcome      VARCHAR(32) NOT NULL,
    position_asset       VARCHAR(128) NOT NULL,
    settled_quantity     BIGINT NOT NULL,
    payout_asset         VARCHAR(32) NOT NULL,
    payout_amount        BIGINT NOT NULL,
    status               VARCHAR(32) NOT NULL DEFAULT 'COMPLETED',
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_settlement_payouts_market_id_user_id
    ON settlement_payouts(market_id, user_id);

CREATE TABLE IF NOT EXISTS chain_transactions (
    id                   BIGSERIAL PRIMARY KEY,
    biz_type             VARCHAR(32) NOT NULL,
    ref_id               VARCHAR(64) NOT NULL,
    chain_name           VARCHAR(32) NOT NULL DEFAULT 'bsc',
    network_name         VARCHAR(32) NOT NULL DEFAULT 'testnet',
    wallet_address       VARCHAR(128) NOT NULL DEFAULT '',
    tx_hash              VARCHAR(128) NOT NULL DEFAULT '',
    status               VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    payload              JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           BIGINT NOT NULL DEFAULT 0,
    updated_at           BIGINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_chain_transactions_tx_hash
    ON chain_transactions(tx_hash)
    WHERE tx_hash <> '';

CREATE INDEX IF NOT EXISTS idx_chain_transactions_ref
    ON chain_transactions(biz_type, ref_id);

COMMIT;
