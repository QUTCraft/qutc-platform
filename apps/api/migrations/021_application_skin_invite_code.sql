ALTER TABLE applications
  ADD COLUMN skin_invite_code VARCHAR(500) NOT NULL DEFAULT '' AFTER decision_reason;
