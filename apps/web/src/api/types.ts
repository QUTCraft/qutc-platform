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

export interface PublicContentDetail {
  id: string
  title: string
  type: 'news' | 'resource' | 'knowledge'
  category: string
  excerpt: string
  body: string
  published_at?: string | null
  updated_at: string
  reading_minutes: number
  asset?: { id: string; original_name: string; mime_type: string; size_bytes: number } | null
  download_url?: string | null
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
  download_url: string | null
}

export interface MediaAsset {
  id: string
  content_id?: string | null
  original_name: string
  mime_type: string
  size_bytes: number
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

export interface KnowledgeDirectory {
  id: string
  name: string
  slug: string
  description: string
  article_count: number
  updated_at: string
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
  category?: string
  knowledge_directory_id?: string | null
  status: 'draft' | 'published' | 'review' | 'archived'
  author: string
  excerpt?: string
  body?: string
  published_at?: string | null
  updated_at: string
}

export interface AdminKnowledgeDirectory {
  id: string
  parent_id: string
  name: string
  slug: string
  description: string
  sort_order: number
  is_public: boolean
  updated_at: string
}

export interface AdminProject {
  id: string
  title: string
  summary: string
  status: 'active' | 'research' | 'completed'
  tags: string[]
  is_public: boolean
  owner: string
  member_count?: number
  milestone_count?: number
  updated_at: string
}

export interface AdminProjectMember {
  user_id: string
  name: string
  email: string
  state: 'active' | 'invited' | 'disabled'
  role: 'member' | 'contributor' | 'lead' | 'owner'
  assigned_at: string
}

export interface AdminProjectMilestone {
  id: string
  project_id: string
  title: string
  status: 'planned' | 'active' | 'completed'
  due_at?: string | null
  completed_at?: string | null
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

export type InvitationRole = 'member' | 'editor' | 'administrator'
export type InvitationStatus = 'pending' | 'accepted' | 'expired' | 'revoked'

export interface Invitation {
  id: string
  organization_id: string
  organization_name: string
  email: string
  role: InvitationRole
  status: InvitationStatus
  expires_at: string
  created_at: string
}

export interface AdminInvitation extends Invitation {
  invite_url: string
}

export interface InvitationAcceptance extends Invitation {
  membership_id: string
}

export interface AdminApplication {
  id: string
  applicant: string
  type: 'whitelist' | 'membership'
  submitted_at: string
  note: string
  status: 'pending' | 'approved' | 'rejected'
  class_name?: string
  game_id?: string
  qq_number?: string
  email?: string
  decided_at?: string | null
  decided_by?: string
  decision_reason: string
  server_sync?: ApplicationServerSync | null
}

export interface AdminApplicationFilters {
  page?: number
  page_size?: number
  status?: '' | AdminApplication['status']
  type?: '' | AdminApplication['type']
  server_sync_status?: '' | 'none' | ApplicationServerSync['status']
  query?: string
}

export interface ApplicationServerSync {
  id: string
  operation: 'whitelist.add'
  adapter: string
  mode: 'mock' | 'rcon'
  status: 'pending' | 'succeeded' | 'failed'
  attempts: number
  message: string
  last_error: string
  requested_at: string
  completed_at?: string | null
}

export interface AdminAuditEvent {
  id: string
  action: string
  target_type: string
  target_id: string
  result: 'success' | 'accepted' | 'failed'
  request_id: string
  actor_user_id: string
  actor_name: string
  created_at: string
}

export interface AdminAuditFilters {
  page?: number
  page_size?: number
  action?: string
  target_type?: string
  result?: '' | AdminAuditEvent['result']
  request_id?: string
  actor_user_id?: string
}

export interface AdminServerStatus {
  enabled: boolean
  adapter: string
  mode: 'mock' | 'rcon'
  label: string
  state: 'online' | 'maintenance' | 'offline'
  online_players: number
  max_players: number
  last_command_at?: string
  updated_at: string
  last_error?: string
}

export interface ServerCommandResult {
  accepted: boolean
  executed: boolean
  mode: 'mock' | 'rcon'
  message: string
  executed_at: string
}

export interface PortalManifest {
  schema: 'qutc.portal/v1'
  id: string
  version: string
  display_name: string
  entry: string
  theme: {
    mode: 'md3' | 'custom'
    tokens?: string
  }
  capabilities: Array<'organization.read' | 'public_content.read' | 'projects.read' | 'assets.read' | 'knowledge.read' | 'server.status.read'>
  fallback: 'md3'
  integrity?: string
}

export interface PortalConfiguration {
  id?: string
  draft_manifest: PortalManifest | null
  active_manifest: PortalManifest | null
  active: boolean
  updated_by?: string
  updated_at?: string
  activated_by?: string
  activated_at?: string
}

export interface PortalRuntimeConfiguration {
  manifest: PortalManifest
  source: 'default' | 'active'
  activated_at?: string
}

export interface AuthUser {
  id: string
  email: string
  display_name: string
  bio?: string
  avatar_url?: string
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
  type?: 'whitelist' | 'membership'
  class_name: string
  name: string
  game_id: string
  qq_number: string
  email: string
  note?: string
}

export interface SmtpSettings {
  host: string
  port: number
  sender_email: string
  recipient_email: string
  auth_code: string
  enable_notification: boolean
}
