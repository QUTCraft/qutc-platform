ALTER TABLE users
    ADD COLUMN default_organization_id CHAR(36) NOT NULL DEFAULT '' AFTER state,
    ADD KEY idx_users_default_organization_id (default_organization_id);

ALTER TABLE organizations
    ADD COLUMN invitation_subject_template VARCHAR(255) NOT NULL DEFAULT '' AFTER is_public,
    ADD COLUMN invitation_body_template TEXT NOT NULL AFTER invitation_subject_template;

ALTER TABLE media_assets
    ADD COLUMN download_count BIGINT NOT NULL DEFAULT 0 AFTER storage_path,
    ADD COLUMN last_downloaded_at DATETIME(3) NULL AFTER download_count,
    ADD KEY idx_media_assets_last_downloaded_at (last_downloaded_at);

ALTER TABLE agent_citations
    ADD COLUMN source_body LONGTEXT NOT NULL AFTER excerpt;

CREATE TABLE IF NOT EXISTS content_revisions (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  content_id CHAR(36) COLLATE utf8mb4_0900_ai_ci NOT NULL,
  version INT NOT NULL,
  created_by CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  reason VARCHAR(32) NOT NULL,
  title VARCHAR(160) NOT NULL,
  type VARCHAR(24) NOT NULL,
  category VARCHAR(64) NOT NULL DEFAULT '',
  knowledge_directory_id CHAR(36) NOT NULL DEFAULT '',
  status VARCHAR(24) NOT NULL,
  excerpt VARCHAR(500) NOT NULL DEFAULT '',
  body LONGTEXT NOT NULL,
  published_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_content_revision_version (content_id, version),
  KEY idx_content_revisions_organization_id (organization_id),
  KEY idx_content_revisions_content_id (content_id),
  KEY idx_content_revisions_created_by (created_by),
  KEY idx_content_revisions_created_at (created_at),
  CONSTRAINT fk_content_revisions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_content_revisions_content FOREIGN KEY (content_id) REFERENCES contents(id) ON DELETE CASCADE,
  CONSTRAINT fk_content_revisions_author FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notification_outboxes (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id CHAR(36) NOT NULL,
  recipient_email VARCHAR(254) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0,
  last_error VARCHAR(500) NOT NULL DEFAULT '',
  available_at DATETIME(3) NOT NULL,
  last_attempt_at DATETIME(3) NULL,
  sent_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_notification_outbox_event_target (event_type, target_type, target_id),
  KEY idx_notification_outboxes_organization_id (organization_id),
  KEY idx_notification_outboxes_status_available (status, available_at),
  KEY idx_notification_outboxes_recipient_email (recipient_email),
  CONSTRAINT fk_notification_outboxes_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
