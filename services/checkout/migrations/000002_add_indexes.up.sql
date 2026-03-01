BEGIN;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_checkout_sessions_user_status_created
    ON checkout_sessions (user_id, status, created_at DESC);

COMMIT;
