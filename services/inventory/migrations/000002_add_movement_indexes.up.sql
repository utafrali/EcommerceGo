BEGIN;

-- Composite index for time-ordered stock movement queries per product.
-- Supports: "show recent stock movements for product X" (dashboard, audit).
CREATE INDEX IF NOT EXISTS idx_movements_product_created
    ON stock_movements(product_id, created_at DESC);

COMMIT;
