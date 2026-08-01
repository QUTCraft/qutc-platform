-- Ordered migration; legacy AutoMigrate volumes are registered as the 001-008 baseline.
ALTER TABLE contents
  ADD COLUMN knowledge_directory_id CHAR(36) NULL AFTER category,
  ADD KEY contents_knowledge_directory_id (knowledge_directory_id);
