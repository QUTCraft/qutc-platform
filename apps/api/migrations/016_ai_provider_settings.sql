ALTER TABLE agent_configurations
  ADD COLUMN provider VARCHAR(32) NOT NULL DEFAULT '' AFTER organization_id,
  ADD COLUMN provider_base_url VARCHAR(500) NOT NULL DEFAULT '' AFTER provider,
  ADD COLUMN provider_api_key_encrypted TEXT NULL AFTER provider_base_url,
  ADD COLUMN provider_model VARCHAR(120) NOT NULL DEFAULT '' AFTER provider_api_key_encrypted;

UPDATE agent_configurations
SET provider_api_key_encrypted = ''
WHERE provider_api_key_encrypted IS NULL;

ALTER TABLE agent_configurations
  MODIFY COLUMN provider_api_key_encrypted TEXT NOT NULL;
