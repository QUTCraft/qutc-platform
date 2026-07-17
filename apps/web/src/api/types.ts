export interface ApiMeta {
  request_id: string
  page?: number
  page_size?: number
  total?: number
}

export interface ApiEnvelope<T> {
  data: T
  meta: ApiMeta
}

export interface Page<T> {
  items: T[]
  page: number
  page_size: number
  total: number
}

export interface Organization {
  id: string
  slug: string
  name: string
  short_name: string
  tagline: string
  introduction: string
  contact_email: string
  social_links: Array<{ label: string; href: string }>
}

export interface PublicPost {
  id: string
  title: string
  excerpt: string
  cover_url?: string
  category: string
  published_at: string
  reading_minutes: number
}

export interface Project {
  id: string
  title: string
  summary: string
  status: 'active' | 'research' | 'completed'
  tags: string[]
  updated_at: string
  public_url?: string
}

export interface Resource {
  id: string
  title: string
  description: string
  kind: 'document' | 'template' | 'package' | 'video'
  size_bytes: number
  updated_at: string
  download_url: string
}

export interface KnowledgeArticle {
  id: string
  title: string
  summary: string
  category: string
  updated_at: string
  reading_minutes: number
}

export interface ServerStatus {
  enabled: boolean
  label: string
  state: 'online' | 'maintenance' | 'offline'
  version?: string
  online_players?: number
  max_players?: number
  updated_at: string
  apply_url?: string
}
