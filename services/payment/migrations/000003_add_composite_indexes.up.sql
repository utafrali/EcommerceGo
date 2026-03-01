BEGIN;

-- Composite index for user payment history filtered by status.
-- Supports: "show completed/pending payments for user X" (order history, admin).
CREATE INDEX IF NOT EXISTS idx_payments_user_status
    ON payments(user_id, status);

COMMIT;
