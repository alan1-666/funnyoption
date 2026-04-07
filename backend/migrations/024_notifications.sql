CREATE TABLE IF NOT EXISTS notifications (
    notification_id  BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL,
    type             VARCHAR(64) NOT NULL,
    title            VARCHAR(512) NOT NULL,
    body             TEXT NOT NULL DEFAULT '',
    metadata         JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id, is_read, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);
