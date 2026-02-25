-- =============================================================================
-- Inventory Service - Initial Schema Migration
-- =============================================================================

CREATE TABLE stock (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL,
    variant_id      UUID NOT NULL,
    warehouse_id    UUID DEFAULT '00000000-0000-0000-0000-000000000001',
    quantity        INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved        INTEGER NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    low_stock_threshold INTEGER DEFAULT 10,
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(product_id, variant_id, warehouse_id)
);

CREATE TABLE stock_reservations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL,
    variant_id      UUID NOT NULL,
    quantity        INTEGER NOT NULL CHECK (quantity > 0),
    checkout_id     UUID NOT NULL,
    status          VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'confirmed', 'released', 'expired')),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE stock_movements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL,
    variant_id      UUID NOT NULL,
    quantity_change  INTEGER NOT NULL,
    reason          VARCHAR(50) NOT NULL CHECK (reason IN ('order', 'return', 'adjustment', 'reservation')),
    reference_id    UUID,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_stock_product_variant ON stock(product_id, variant_id);
CREATE INDEX idx_reservations_checkout ON stock_reservations(checkout_id);
CREATE INDEX idx_reservations_status_expires ON stock_reservations(status, expires_at) WHERE status = 'active';
CREATE INDEX idx_movements_product ON stock_movements(product_id, variant_id);
