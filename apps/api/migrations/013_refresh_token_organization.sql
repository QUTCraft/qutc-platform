ALTER TABLE refresh_tokens
  ADD COLUMN organization_id CHAR(36) NOT NULL DEFAULT '' AFTER user_id,
  ADD KEY idx_refresh_tokens_organization_id (organization_id);
