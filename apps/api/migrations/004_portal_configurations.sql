-- Draft and active manifests are stored separately so preview/save never
-- changes the currently active public portal.
CREATE TABLE IF NOT EXISTS portal_configurations (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  draft_manifest_json LONGTEXT NOT NULL,
  active_manifest_json LONGTEXT NOT NULL,
  updated_by CHAR(36) NOT NULL,
  activated_by CHAR(36) NULL,
  activated_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY portal_configurations_organization_id (organization_id),
  KEY portal_configurations_updated_by (updated_by),
  KEY portal_configurations_activated_by (activated_by),
  KEY portal_configurations_activated_at (activated_at),
  CONSTRAINT fk_portal_configurations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_portal_configurations_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
  CONSTRAINT fk_portal_configurations_activated_by FOREIGN KEY (activated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
