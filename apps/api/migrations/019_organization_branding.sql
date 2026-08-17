ALTER TABLE organizations
  ADD COLUMN filing_number VARCHAR(80) NOT NULL DEFAULT '' AFTER contact_email,
  ADD COLUMN logo_asset_id CHAR(36) NOT NULL DEFAULT '' AFTER filing_number,
  ADD INDEX idx_organizations_logo_asset_id (logo_asset_id)
