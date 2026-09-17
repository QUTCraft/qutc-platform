CREATE TABLE IF NOT EXISTS site_feedback (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  reporter_user_id CHAR(36) NOT NULL DEFAULT '',
  kind VARCHAR(24) NOT NULL,
  title VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  contact_name VARCHAR(80) NOT NULL,
  contact_email VARCHAR(254) NOT NULL,
  page_url VARCHAR(500) NOT NULL DEFAULT '',
  status VARCHAR(24) NOT NULL DEFAULT 'open',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY site_feedback_organization_created (organization_id, created_at),
  KEY site_feedback_organization_kind (organization_id, kind),
  CONSTRAINT fk_site_feedback_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
