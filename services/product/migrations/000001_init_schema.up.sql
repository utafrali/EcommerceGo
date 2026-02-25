-- =============================================================================
-- Product Service - Initial Schema Migration
-- =============================================================================

-- Enable the UUID extension if not already enabled.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- Brands
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS brands (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    logo_url   TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_brands_slug ON brands (slug);

-- -----------------------------------------------------------------------------
-- Categories
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS categories (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(255) NOT NULL UNIQUE,
    parent_id  UUID REFERENCES categories(id) ON DELETE SET NULL,
    sort_order INT          NOT NULL DEFAULT 0,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_slug      ON categories (slug);
CREATE INDEX idx_categories_parent_id ON categories (parent_id);

-- -----------------------------------------------------------------------------
-- Products
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS products (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(500) NOT NULL,
    slug        VARCHAR(500) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    brand_id    UUID REFERENCES brands(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'published', 'archived')),
    base_price  BIGINT       NOT NULL DEFAULT 0
                CHECK (base_price >= 0),
    currency    CHAR(3)      NOT NULL DEFAULT 'USD',
    metadata    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_slug        ON products (slug);
CREATE INDEX idx_products_status      ON products (status);
CREATE INDEX idx_products_brand_id    ON products (brand_id);
CREATE INDEX idx_products_category_id ON products (category_id);
CREATE INDEX idx_products_base_price  ON products (base_price);
CREATE INDEX idx_products_created_at  ON products (created_at DESC);

-- GIN index for full-text search on name and description.
CREATE INDEX idx_products_search ON products USING GIN (
    to_tsvector('english', name || ' ' || description)
);

-- GIN index for JSONB metadata queries.
CREATE INDEX idx_products_metadata ON products USING GIN (metadata);

-- -----------------------------------------------------------------------------
-- Product Variants
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS product_variants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku         VARCHAR(100) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    price       BIGINT       CHECK (price IS NULL OR price >= 0),
    attributes  JSONB        NOT NULL DEFAULT '{}',
    weight_grams INT,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_variants_product_id ON product_variants (product_id);
CREATE INDEX idx_product_variants_sku        ON product_variants (sku);
CREATE INDEX idx_product_variants_active     ON product_variants (product_id) WHERE is_active = TRUE;

-- -----------------------------------------------------------------------------
-- Product Images
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS product_images (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url        TEXT         NOT NULL,
    alt_text   VARCHAR(500) NOT NULL DEFAULT '',
    sort_order INT          NOT NULL DEFAULT 0,
    is_primary BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_images_product_id ON product_images (product_id);
CREATE INDEX idx_product_images_primary    ON product_images (product_id) WHERE is_primary = TRUE;

-- -----------------------------------------------------------------------------
-- Trigger: auto-update updated_at on row changes
-- -----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_brands
    BEFORE UPDATE ON brands
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_categories
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_products
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_product_variants
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
