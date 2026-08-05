CREATE TABLE IF NOT EXISTS activity_plan_evaluations (
  id CHAR(36) PRIMARY KEY,
  organization_id CHAR(36) NOT NULL,
  plan_id CHAR(36) NOT NULL,
  reviewer_user_id CHAR(36) NOT NULL,
  accuracy INT NOT NULL,
  feasibility INT NOT NULL,
  campus_fit INT NOT NULL,
  clarity INT NOT NULL,
  adoptability INT NOT NULL,
  notes VARCHAR(1000) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  UNIQUE KEY idx_activity_plan_evaluation_reviewer (plan_id, reviewer_user_id),
  KEY idx_activity_plan_evaluations_organization_id (organization_id),
  KEY idx_activity_plan_evaluations_plan_id (plan_id),
  KEY idx_activity_plan_evaluations_reviewer_user_id (reviewer_user_id),
  KEY idx_activity_plan_evaluations_created_at (created_at)
)
 ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
