-- =============================================================================
-- init-db.sql
-- Runs once on first PostgreSQL container start (data volume empty).
-- Creates one database per microservice.
-- =============================================================================

-- Product Service
SELECT 'CREATE DATABASE product_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'product_db') \gexec

-- Cart Service
SELECT 'CREATE DATABASE cart_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'cart_db') \gexec

-- Order Service
SELECT 'CREATE DATABASE order_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'order_db') \gexec

-- Checkout Service
SELECT 'CREATE DATABASE checkout_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'checkout_db') \gexec

-- Payment Service
SELECT 'CREATE DATABASE payment_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'payment_db') \gexec

-- User Service
SELECT 'CREATE DATABASE user_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'user_db') \gexec

-- Inventory Service
SELECT 'CREATE DATABASE inventory_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'inventory_db') \gexec

-- Campaign Service
SELECT 'CREATE DATABASE campaign_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'campaign_db') \gexec

-- Notification Service
SELECT 'CREATE DATABASE notification_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notification_db') \gexec

-- Media Service
SELECT 'CREATE DATABASE media_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'media_db') \gexec

-- Search Service (uses Elasticsearch but may need a small config/audit table)
SELECT 'CREATE DATABASE search_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'search_db') \gexec
