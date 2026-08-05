INSERT INTO content_revisions (
  id,
  organization_id,
  content_id,
  version,
  created_by,
  reason,
  title,
  type,
  category,
  knowledge_directory_id,
  status,
  excerpt,
  body,
  published_at,
  created_at
)
SELECT
  UUID(),
  c.organization_id,
  c.id,
  1,
  c.author_user_id,
  'create',
  c.title,
  c.type,
  c.category,
  COALESCE(c.knowledge_directory_id, ''),
  c.status,
  c.excerpt,
  c.body,
  c.published_at,
  c.created_at
FROM contents c
WHERE NOT EXISTS (
  SELECT 1
  FROM content_revisions r
  WHERE r.content_id = c.id
);
