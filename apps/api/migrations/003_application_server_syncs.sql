-- Application decisions and external server synchronization are separate facts.
-- Ordered migration; legacy AutoMigrate volumes are registered as the 001-008 baseline.
CREATE TABLE IF NOT EXISTS application_server_syncs (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  application_id CHAR(36) NOT NULL,
  operation VARCHAR(64) NOT NULL,
  adapter VARCHAR(64) NOT NULL,
  mode VARCHAR(24) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  message VARCHAR(500) NOT NULL DEFAULT '',
  last_error VARCHAR(500) NOT NULL DEFAULT '',
  requested_at DATETIME(3) NOT NULL,
  completed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY application_server_syncs_organization_id (organization_id),
  KEY application_server_syncs_application_id (application_id),
  KEY application_server_syncs_status (status),
  CONSTRAINT fk_application_server_syncs_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_application_server_syncs_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
