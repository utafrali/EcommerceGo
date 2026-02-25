CREATE TABLE IF NOT EXISTS checkout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'initiated',
    items JSONB NOT NULL DEFAULT '[]',
    subtotal_amount BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    shipping_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    shipping_address JSONB,
    billing_address JSONB,
    payment_method VARCHAR(50),
    payment_id UUID,
    order_id UUID,
    failure_reason TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkout_sessions_user_id ON checkout_sessions(user_id);
CREATE INDEX idx_checkout_sessions_status ON checkout_sessions(status);
CREATE INDEX idx_checkout_sessions_expires_at ON checkout_sessions(expires_at) WHERE status NOT IN ('completed', 'failed', 'expired');
