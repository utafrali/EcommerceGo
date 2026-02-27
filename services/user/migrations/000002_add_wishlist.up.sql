-- =============================================================================
-- User Service - Add Wishlist Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS wishlists (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, product_id)
);

CREATE INDEX idx_wishlists_user ON wishlists(user_id, created_at DESC);
