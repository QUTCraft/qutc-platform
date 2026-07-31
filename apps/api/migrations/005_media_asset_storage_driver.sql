ALTER TABLE media_assets
    ADD COLUMN storage_driver VARCHAR(16) NOT NULL DEFAULT 'local' AFTER size_bytes;
