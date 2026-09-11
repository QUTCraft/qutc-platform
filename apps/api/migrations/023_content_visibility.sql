ALTER TABLE contents
  ADD COLUMN is_public TINYINT(1) NOT NULL DEFAULT 1 AFTER status,
  ADD KEY contents_organization_status_public (organization_id, status, is_public);
