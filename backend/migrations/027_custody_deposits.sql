CREATE TABLE IF NOT EXISTS custody_deposits (
  id          BIGSERIAL PRIMARY KEY,
  biz_id      TEXT NOT NULL UNIQUE,
  user_id     BIGINT NOT NULL,
  address     TEXT NOT NULL,
  asset       TEXT NOT NULL,
  chain_amount TEXT NOT NULL,
  credit_amount BIGINT NOT NULL,
  chain_id    BIGINT NOT NULL DEFAULT 0,
  tx_hash     TEXT NOT NULL DEFAULT '',
  tx_index    INT NOT NULL DEFAULT 0,
  status      TEXT NOT NULL DEFAULT 'CREDITED',
  created_at  BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);
CREATE INDEX IF NOT EXISTS idx_custody_deposits_user ON custody_deposits (user_id);
