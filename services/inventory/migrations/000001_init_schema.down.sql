-- =============================================================================
-- Inventory Service - Rollback Initial Schema Migration
-- =============================================================================

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS stock;
