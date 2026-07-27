-- The API also applies this change through GORM AutoMigrate for existing volumes.
-- This file keeps a fresh Compose database aligned with the runtime model.
ALTER TABLE contents
  ADD COLUMN knowledge_directory_id CHAR(36) NULL AFTER category,
  ADD KEY contents_knowledge_directory_id (knowledge_directory_id);
