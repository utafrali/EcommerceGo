CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    discount_value BIGINT NOT NULL,
    min_order_amount BIGINT NOT NULL DEFAULT 0,
    max_discount_amount BIGINT NOT NULL DEFAULT 0,
    code VARCHAR(50) UNIQUE,
    max_usage_count INT NOT NULL DEFAULT 0,
    current_usage_count INT NOT NULL DEFAULT 0,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    applicable_categories JSONB DEFAULT '[]',
    applicable_products JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_campaigns_code ON campaigns(code) WHERE code IS NOT NULL;
CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_dates ON campaigns(start_date, end_date);

CREATE TABLE IF NOT EXISTS campaign_usages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    user_id UUID NOT NULL,
    order_id UUID NOT NULL,
    discount_applied BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_campaign_usages_campaign_id ON campaign_usages(campaign_id);
CREATE INDEX idx_campaign_usages_user_id ON campaign_usages(user_id);
