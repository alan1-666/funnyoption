BEGIN;

CREATE TABLE IF NOT EXISTS market_categories (
    category_id     BIGSERIAL PRIMARY KEY,
    category_key    VARCHAR(32) NOT NULL UNIQUE,
    display_name    VARCHAR(64) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    sort_order      INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0
);

INSERT INTO market_categories (
    category_key, display_name, description, status, sort_order, metadata, created_at, updated_at
)
VALUES
    ('CRYPTO', '加密', '加密资产与链上生态相关市场', 'ACTIVE', 10, '{}'::jsonb, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('SPORTS', '体育', '体育赛事与比分相关市场', 'ACTIVE', 20, '{}'::jsonb, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (category_key) DO UPDATE
SET display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    sort_order = EXCLUDED.sort_order,
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;

ALTER TABLE markets
    ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES market_categories(category_id);

CREATE INDEX IF NOT EXISTS idx_markets_category_id_status_open_at
    ON markets(category_id, status, open_at);

WITH normalized_categories AS (
    SELECT
        m.market_id,
        CASE
            WHEN UPPER(COALESCE(m.metadata->>'categoryKey', m.metadata->>'category_key', '')) = 'SPORTS' THEN 'SPORTS'
            WHEN UPPER(COALESCE(m.metadata->>'category', '')) IN ('SPORTS', 'SPORT', '体育') THEN 'SPORTS'
            ELSE 'CRYPTO'
        END AS category_key
    FROM markets m
)
UPDATE markets AS m
SET category_id = c.category_id
FROM normalized_categories nc
INNER JOIN market_categories c ON c.category_key = nc.category_key
WHERE m.market_id = nc.market_id
  AND m.category_id IS NULL;

CREATE TABLE IF NOT EXISTS market_option_sets (
    market_id        BIGINT PRIMARY KEY REFERENCES markets(market_id) ON DELETE CASCADE,
    option_schema    JSONB NOT NULL DEFAULT '[]'::jsonb,
    version          INT NOT NULL DEFAULT 1,
    created_at       BIGINT NOT NULL DEFAULT 0,
    updated_at       BIGINT NOT NULL DEFAULT 0
);

INSERT INTO market_option_sets (
    market_id, option_schema, version, created_at, updated_at
)
SELECT
    m.market_id,
    '[
      {"key":"YES","label":"是","short_label":"是","sort_order":10,"is_active":true},
      {"key":"NO","label":"否","short_label":"否","sort_order":20,"is_active":true}
    ]'::jsonb,
    1,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
FROM markets m
ON CONFLICT (market_id) DO NOTHING;

COMMIT;
