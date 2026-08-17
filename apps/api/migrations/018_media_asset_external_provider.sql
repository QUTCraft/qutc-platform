ALTER TABLE media_assets
  ADD COLUMN provider VARCHAR(24) NOT NULL DEFAULT 'local' AFTER storage_path,
  ADD COLUMN external_url VARCHAR(500) NOT NULL DEFAULT '' AFTER provider;
