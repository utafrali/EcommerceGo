-- =============================================================================
-- User Service - Initial Schema Migration
-- =============================================================================

-- Enable the UUID extension if not already enabled.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- Users
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255),
    first_name      VARCHAR(100),
    last_name       VARCHAR(100),
    phone           VARCHAR(20),
    role            VARCHAR(20) DEFAULT 'customer'
                    CHECK (role IN ('customer', 'admin', 'seller')),
    is_active       BOOLEAN DEFAULT true,
    email_verified  BOOLEAN DEFAULT false,
    oauth_provider  VARCHAR(50),
    oauth_id        VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email      ON users (email);
CREATE INDEX idx_users_role       ON users (role);
CREATE INDEX idx_users_active     ON users (is_active) WHERE is_active = true;
CREATE INDEX idx_users_created_at ON users (created_at DESC);

-- -----------------------------------------------------------------------------
-- Addresses
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label           VARCHAR(50),
    first_name      VARCHAR(100) NOT NULL,
    last_name       VARCHAR(100) NOT NULL,
    address_line1   VARCHAR(500) NOT NULL,
    address_line2   VARCHAR(500),
    city            VARCHAR(100) NOT NULL,
    state           VARCHAR(100),
    postal_code     VARCHAR(20) NOT NULL,
    country_code    VARCHAR(2) NOT NULL,
    phone           VARCHAR(20),
    is_default      BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_addresses_user_id  ON addresses (user_id);
CREATE INDEX idx_addresses_default  ON addresses (user_id) WHERE is_default = true;

-- -----------------------------------------------------------------------------
-- Refresh Tokens
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX idx_refresh_tokens_active     ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

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

CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
