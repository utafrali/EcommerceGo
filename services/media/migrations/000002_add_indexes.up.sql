BEGIN;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_files_owner_created
    ON media_files (owner_id, owner_type, created_at DESC);

COMMIT;
