CREATE TABLE IF NOT EXISTS custody_address_mapping (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT NOT NULL,
  tenant_id   TEXT NOT NULL DEFAULT 'funnyoption',
  chain       TEXT NOT NULL,
  network     TEXT NOT NULL,
  coin        TEXT NOT NULL,
  address     TEXT NOT NULL,
  created_at  BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  UNIQUE (tenant_id, chain, network, coin, address)
);

CREATE INDEX IF NOT EXISTS idx_custody_addr_user ON custody_address_mapping (user_id);
CREATE INDEX IF NOT EXISTS idx_custody_addr_lookup ON custody_address_mapping (address, chain, network);
