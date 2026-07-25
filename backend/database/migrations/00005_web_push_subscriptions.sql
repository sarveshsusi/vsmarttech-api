-- +goose Up
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_push_subscriptions_endpoint ON push_subscriptions (endpoint);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions (user_id);

ALTER TABLE notification_preferences
    ADD COLUMN IF NOT EXISTS push_notifications BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN IF EXISTS push_notifications;
DROP TABLE IF EXISTS push_subscriptions;
