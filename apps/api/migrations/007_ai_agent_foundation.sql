CREATE TABLE IF NOT EXISTS agent_definitions (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  `key` VARCHAR(64) NOT NULL,
  name VARCHAR(120) NOT NULL,
  purpose VARCHAR(500) NOT NULL,
  system_policy_version VARCHAR(64) NOT NULL,
  allowed_tool_keys TEXT NOT NULL,
  model_profile VARCHAR(64) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_agent_definition_org_key (organization_id, `key`),
  KEY idx_agent_definitions_enabled (enabled),
  CONSTRAINT fk_agent_definitions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
)
 ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS agent_runs (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  actor_user_id CHAR(36) NOT NULL,
  agent_definition_id CHAR(36) NOT NULL,
  status VARCHAR(24) NOT NULL,
  task TEXT NOT NULL,
  output_title VARCHAR(160) NOT NULL DEFAULT '',
  output_excerpt VARCHAR(500) NOT NULL DEFAULT '',
  output_markdown LONGTEXT NOT NULL,
  provider VARCHAR(32) NOT NULL,
  mode VARCHAR(24) NOT NULL,
  model VARCHAR(120) NOT NULL,
  prompt_version VARCHAR(64) NOT NULL,
  input_tokens INT NOT NULL DEFAULT 0,
  output_tokens INT NOT NULL DEFAULT 0,
  failure_code VARCHAR(64) NOT NULL DEFAULT '',
  failure_message VARCHAR(500) NOT NULL DEFAULT '',
  request_id VARCHAR(64) NOT NULL,
  started_at DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  KEY idx_agent_runs_organization_id (organization_id),
  KEY idx_agent_runs_actor_user_id (actor_user_id),
  KEY idx_agent_runs_agent_definition_id (agent_definition_id),
  KEY idx_agent_runs_status (status),
  KEY idx_agent_runs_request_id (request_id),
  KEY idx_agent_runs_expires_at (expires_at),
  KEY idx_agent_runs_created_at (created_at),
  CONSTRAINT fk_agent_runs_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_runs_actor FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_agent_runs_definition FOREIGN KEY (agent_definition_id) REFERENCES agent_definitions(id) ON DELETE RESTRICT
)
 ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS agent_citations (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  organization_id CHAR(36) NOT NULL,
  source_type VARCHAR(32) NOT NULL,
  source_id VARCHAR(64) NOT NULL,
  title VARCHAR(160) NOT NULL,
  excerpt VARCHAR(500) NOT NULL,
  source_updated_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_agent_citation_run_source (run_id, source_type, source_id),
  KEY idx_agent_citations_organization_id (organization_id),
  CONSTRAINT fk_agent_citations_run FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_citations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
)
 ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
