#!/bin/bash
# =============================================================================
# init-databases.sh
# Runs once on first PostgreSQL container start (when data volume is empty).
# Creates one database per microservice.
# =============================================================================
set -e

for db in product_db user_db order_db checkout_db payment_db inventory_db campaign_db notification_db media_db cart_db search_db; do
    echo "Creating database: $db"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
        SELECT 'CREATE DATABASE $db'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db') \gexec
EOSQL
done

echo "All databases created successfully."
