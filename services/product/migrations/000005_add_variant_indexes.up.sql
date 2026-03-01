BEGIN;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_product_variants_active_created
    ON product_variants (product_id, is_active, created_at DESC);

COMMIT;
