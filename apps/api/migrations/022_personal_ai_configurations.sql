CREATE TABLE IF NOT EXISTS personal_ai_configurations (
  user_id CHAR(36) COLLATE utf8mb4_unicode_ci PRIMARY KEY,
  base_url VARCHAR(500) NOT NULL,
  model VARCHAR(120) NOT NULL,
  api_key_encrypted TEXT NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  CONSTRAINT fk_personal_ai_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
