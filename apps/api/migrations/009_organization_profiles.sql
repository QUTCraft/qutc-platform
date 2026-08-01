ALTER TABLE organizations
    ADD COLUMN short_name VARCHAR(40) NOT NULL DEFAULT '' AFTER name,
    ADD COLUMN tagline VARCHAR(160) NOT NULL DEFAULT '' AFTER short_name,
    ADD COLUMN introduction VARCHAR(2000) NOT NULL DEFAULT '' AFTER tagline,
    ADD COLUMN contact_email VARCHAR(254) NOT NULL DEFAULT '' AFTER introduction,
	ADD COLUMN social_links_json TEXT NOT NULL DEFAULT ('[]') AFTER contact_email,
    ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT TRUE AFTER social_links_json,
    ADD INDEX idx_organizations_is_public (is_public);

UPDATE organizations
SET short_name = CASE WHEN slug = 'qutcraft' THEN 'QUTCraft' ELSE name END,
    tagline = CASE WHEN slug = 'qutcraft' THEN '把社团正在发生的事，认真地呈现出来。' ELSE '' END,
    introduction = CASE WHEN slug = 'qutcraft' THEN 'QUTCraft 是青岛理工大学的 Minecraft 社团，持续建设内容、项目与公共知识资产。' ELSE '' END,
    contact_email = CASE WHEN slug = 'qutcraft' THEN 'contact@qutcraft.example' ELSE '' END,
    social_links_json = CASE WHEN slug = 'qutcraft' THEN '[{"label":"GitHub","href":"https://github.com/QUTCraft/qutc-platform"}]' ELSE '[]' END
WHERE short_name = '';
