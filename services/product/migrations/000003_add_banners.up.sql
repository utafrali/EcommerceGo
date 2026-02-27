-- =============================================================================
-- Product Service - Add Banners
-- =============================================================================

CREATE TABLE IF NOT EXISTS banners (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title      VARCHAR(255)  NOT NULL,
    subtitle   VARCHAR(500),
    image_url  TEXT          NOT NULL,
    link_url   TEXT          NOT NULL,
    link_type  VARCHAR(20)   NOT NULL DEFAULT 'internal'
               CHECK (link_type IN ('internal', 'external')),
    position   VARCHAR(50)   NOT NULL
               CHECK (position IN ('hero_slider', 'mid_banner', 'category_banner')),
    sort_order INT           NOT NULL DEFAULT 0,
    is_active  BOOLEAN       NOT NULL DEFAULT TRUE,
    starts_at  TIMESTAMPTZ,
    ends_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_banners_position_active ON banners (position, is_active, sort_order);

CREATE TRIGGER set_updated_at_banners
    BEFORE UPDATE ON banners
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
