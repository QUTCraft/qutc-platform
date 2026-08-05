CREATE TABLE IF NOT EXISTS agent_configurations (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  enabled BOOLEAN NOT NULL,
  run_limit_per_hour INT NOT NULL DEFAULT 20,
  request_timeout_seconds INT NOT NULL DEFAULT 30,
  max_sources INT NOT NULL DEFAULT 10,
  max_context_characters INT NOT NULL DEFAULT 30000,
  updated_by CHAR(36) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_agent_configurations_organization_id (organization_id),
  KEY idx_agent_configurations_enabled (enabled),
  KEY idx_agent_configurations_updated_by (updated_by),
  CONSTRAINT fk_agent_configurations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_configurations_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE RESTRICT
)
 ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
