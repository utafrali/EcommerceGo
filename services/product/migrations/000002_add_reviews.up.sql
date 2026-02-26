-- =============================================================================
-- Product Service - Add Product Reviews
-- =============================================================================

CREATE TABLE IF NOT EXISTS product_reviews (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id    UUID         NOT NULL,
    rating     INT          NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title      VARCHAR(255) NOT NULL DEFAULT '',
    body       TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_reviews_product_id ON product_reviews (product_id);
CREATE INDEX idx_product_reviews_user_id    ON product_reviews (user_id);
CREATE INDEX idx_product_reviews_rating     ON product_reviews (product_id, rating);

CREATE TRIGGER set_updated_at_product_reviews
    BEFORE UPDATE ON product_reviews
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
