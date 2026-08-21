CREATE TABLE IF NOT EXISTS content_review_requests (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  content_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  revision_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  requester_user_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  type VARCHAR(24) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'pending',
  note VARCHAR(1000) NOT NULL DEFAULT '',
  feedback VARCHAR(1000) NOT NULL DEFAULT '',
  reviewer_user_id CHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  reviewed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  KEY idx_content_review_requests_organization_id (organization_id),
  KEY idx_content_review_requests_content_id (content_id),
  KEY idx_content_review_requests_revision_id (revision_id),
  KEY idx_content_review_requests_requester_user_id (requester_user_id),
  KEY idx_content_review_requests_reviewer_user_id (reviewer_user_id),
  KEY idx_content_review_requests_type (type),
  KEY idx_content_review_requests_status (status),
  KEY idx_content_review_requests_reviewed_at (reviewed_at),
  KEY idx_content_review_requests_created_at (created_at),
  KEY idx_content_review_lookup (organization_id, content_id, status, created_at),
  CONSTRAINT fk_content_review_requests_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  CONSTRAINT fk_content_review_requests_requester FOREIGN KEY (requester_user_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE notification_outboxes
  DROP INDEX idx_notification_outbox_event_target,
  ADD UNIQUE KEY idx_notification_outbox_event_target_recipient (event_type, target_type, target_id, recipient_email);
