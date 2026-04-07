BEGIN;

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id         BIGINT PRIMARY KEY,
    wallet_address  VARCHAR(128) NOT NULL UNIQUE,
    display_name    VARCHAR(64) NOT NULL DEFAULT '',
    avatar_preset   VARCHAR(32) NOT NULL DEFAULT 'aurora',
    created_at      BIGINT NOT NULL DEFAULT 0,
    updated_at      BIGINT NOT NULL DEFAULT 0
);

INSERT INTO user_profiles (
    user_id, wallet_address, display_name, avatar_preset, created_at, updated_at
)
SELECT DISTINCT ON (ws.user_id)
    ws.user_id,
    LOWER(ws.wallet_address),
    '',
    'aurora',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
FROM wallet_sessions ws
WHERE ws.user_id > 0
  AND TRIM(COALESCE(ws.wallet_address, '')) <> ''
ORDER BY ws.user_id, ws.created_at DESC, ws.updated_at DESC
ON CONFLICT (user_id) DO UPDATE
SET wallet_address = EXCLUDED.wallet_address,
    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT;

COMMIT;
