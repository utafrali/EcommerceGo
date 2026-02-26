-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    subtotal_amount BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    shipping_amount BIGINT NOT NULL DEFAULT 0,
    total_amount    BIGINT NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'USD',
    shipping_address JSONB,
    billing_address  JSONB,
    notes           TEXT NOT NULL DEFAULT '',
    canceled_reason TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at DESC);

-- Order items table
CREATE TABLE IF NOT EXISTS order_items (
    id         TEXT PRIMARY KEY,
    order_id   TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    variant_id TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL,
    sku        TEXT NOT NULL DEFAULT '',
    price      BIGINT NOT NULL DEFAULT 0,
    quantity   INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
