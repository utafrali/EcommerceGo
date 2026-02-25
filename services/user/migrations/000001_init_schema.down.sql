-- =============================================================================
-- User Service - Rollback Initial Schema Migration
-- =============================================================================

-- Drop triggers first.
DROP TRIGGER IF EXISTS set_updated_at_users ON users;

-- Drop the trigger function.
DROP FUNCTION IF EXISTS trigger_set_updated_at();

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS users;
