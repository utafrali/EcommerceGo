-- =============================================================================
-- Product Service - Rollback Initial Schema Migration
-- =============================================================================

-- Drop triggers first.
DROP TRIGGER IF EXISTS set_updated_at_product_variants ON product_variants;
DROP TRIGGER IF EXISTS set_updated_at_products ON products;
DROP TRIGGER IF EXISTS set_updated_at_categories ON categories;
DROP TRIGGER IF EXISTS set_updated_at_brands ON brands;

-- Drop the trigger function.
DROP FUNCTION IF EXISTS trigger_set_updated_at();

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS brands;
