BEGIN;

-- Composite index for user notification inbox filtered by status.
-- Supports: "show unread/pending notifications for user X" (notification center).
CREATE INDEX IF NOT EXISTS idx_notifications_user_status_created
    ON notifications(user_id, status, created_at DESC);

COMMIT;
