ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS is_stackable BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 0;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS exclusion_group VARCHAR(100);

CREATE TABLE IF NOT EXISTS campaign_stacking_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_a_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    campaign_b_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    rule_type VARCHAR(20) NOT NULL CHECK (rule_type IN ('compatible', 'exclusive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(campaign_a_id, campaign_b_id)
);

CREATE INDEX idx_stacking_rules_campaigns ON campaign_stacking_rules(campaign_a_id, campaign_b_id);
