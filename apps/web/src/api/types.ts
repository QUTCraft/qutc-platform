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

export interface AdminDashboard {
  organization_name: string
  updated_at: string
  metrics: Array<{ label: string; value: number; change?: string; tone: 'primary' | 'secondary' | 'warning' | 'neutral' }>
  pending_applications: AdminApplication[]
  recent_content: AdminContent[]
  server: AdminServerStatus
}

export interface AdminContent {
  id: string
  title: string
  type: 'news' | 'resource' | 'knowledge'
  status: 'draft' | 'published' | 'review' | 'archived'
  author: string
  excerpt?: string
  body?: string
  published_at?: string | null
  updated_at: string
}

export interface AdminUser {
  id: string
  name: string
  email: string
  role: 'owner' | 'administrator' | 'editor' | 'member'
  state: 'active' | 'invited' | 'disabled'
  joined_at: string
}

export interface AdminApplication {
  id: string
  applicant: string
  type: 'whitelist' | 'membership'
  submitted_at: string
  note: string
  status: 'pending' | 'approved' | 'rejected'
}

export interface AdminServerStatus {
  enabled: boolean
  label: string
  state: 'online' | 'maintenance' | 'offline'
  online_players: number
  max_players: number
  last_command_at?: string
}

export interface ServerCommandResult {
  accepted: boolean
  message: string
  executed_at: string
}

export interface AuthUser {
  id: string
  email: string
  display_name: string
  organization_id: string
  roles: Array<'owner' | 'administrator' | 'editor' | 'member'>
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: 'Bearer'
  expires_in: number
  user: AuthUser
}

export interface ApplicationPayload {
  class_name: string
  name: string
  game_id: string
  qq_number: string
  email: string
}

export interface SmtpSettings {
  host: string
  port: number
  sender_email: string
  recipient_email: string
  auth_code: string
  enable_notification: boolean
}
