-- Drop the unique index
DROP INDEX IF EXISTS idx_payments_idempotency_key;

-- Drop the idempotency_key column
ALTER TABLE payments DROP COLUMN IF EXISTS idempotency_key;
