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
	is_public: boolean
	updated_at: string
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
  download_count: number
  last_downloaded_at?: string | null
  created_at?: string
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
  revision_count?: number
  asset?: MediaAsset | null
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

export interface AuditEvent {
  id: string
  actor_user_id: string
  actor_name: string
  action: string
  target_type: string
  target_id: string
  result: string
  request_id: string
  created_at: string
}

export interface AuditEventFilters {
  page?: number
  page_size?: number
  action?: string
  target_type?: string
  result?: string
  actor_user_id?: string
  request_id?: string
  date_from?: string
  date_to?: string
}

export interface AIProviderStatus {
  provider: 'disabled' | 'mock' | 'openai_compatible'
  mode: 'disabled' | 'mock' | 'real'
  model: string
  enabled: boolean
  configured: boolean
}

export interface AIProviderConfiguration {
  driver: AIProviderStatus['provider']
  base_url: string
  model: string
  api_key_configured: boolean
  api_key_hint?: string
  source: 'server' | 'organization'
}

export interface AIAgentDefinition {
  id: string
  key: 'content-copilot' | 'activity-planner'
  name: string
  purpose: string
  system_policy_version: string
  allowed_tool_keys: string[]
  model_profile: string
  enabled: boolean
}

export interface AIAgentCatalog {
  agents: AIAgentDefinition[]
  provider: AIProviderStatus
}

export interface AIConfiguration {
  id?: string
  enabled: boolean
  run_limit_per_hour: number
  request_timeout_seconds: number
  max_sources: number
  max_context_characters: number
  provider: AIProviderStatus
  provider_config: AIProviderConfiguration
  updated_by?: string
  updated_at?: string
}

export type AIConfigurationUpdate = Pick<AIConfiguration, 'enabled' | 'run_limit_per_hour' | 'request_timeout_seconds' | 'max_sources' | 'max_context_characters'> & {
  provider: Exclude<AIProviderStatus['provider'], 'mock'>
  base_url: string
  api_key: string
  model: string
}

export interface AIKnowledgeResult {
  source_type: 'content'
  id: string
  title: string
  excerpt: string
  status: AdminContent['status']
  updated_at: string
}

export interface AISourceReference {
  type: 'content'
  id: string
}

export interface AICitation {
  id: string
  source_type: 'content'
  source_id: string
  title: string
  excerpt: string
  source_updated_at: string
}

export interface AIAgentRun {
  id: string
  agent_key: string
  agent_name: string
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'
  task: string
  output_title: string
  output_excerpt: string
  output_markdown: string
  provider: 'mock' | 'openai_compatible'
  mode: 'mock' | 'real'
  model: string
  prompt_version: string
  input_tokens: number
  output_tokens: number
  failure_code: string
  failure_message: string
  request_id: string
  citations: AICitation[]
  started_at?: string | null
  completed_at?: string | null
  expires_at: string
  created_at: string
  updated_at: string
}

export type ActivityPlanStatus = 'generating' | 'ready' | 'failed' | 'canceled' | 'applied'

export interface ActivityActionProposal {
  key: string
  kind: 'project' | 'milestone' | 'content'
  title: string
  description: string
  due_at?: string | null
  requires: string[]
}

export interface ActivityPlan {
  id: string
  title: string
  objective: string
  audience: string
  venue: string
  starts_at?: string | null
  ends_at?: string | null
  expected_participants: number
  budget: string
  constraints: string
  status: ActivityPlanStatus
  run: AIAgentRun
  proposed_actions: ActivityActionProposal[]
  approved_actions: string[]
  project_id?: string | null
  announcement_content_id?: string | null
  approved_by?: string | null
  approved_at?: string | null
  created_at: string
  updated_at: string
}

export interface ContentRevision {
  id: string
  content_id: string
  version: number
  created_by: string
  reason: 'create' | 'update' | 'published' | 'archived' | 'restore'
  title: string
  type: AdminContent['type']
  category: string
  knowledge_directory_id?: string | null
  status: AdminContent['status']
  excerpt: string
  body?: string
  published_at?: string | null
  created_at: string
}

export interface ActivityPlanSummary {
  id: string
  title: string
  status: ActivityPlanStatus
  starts_at?: string | null
  ends_at?: string | null
  provider: string
  mode: 'mock' | 'real' | 'disabled'
  model: string
  prompt_version: string
  project_id?: string | null
  announcement_content_id?: string | null
  has_my_evaluation: boolean
  my_evaluation_score?: number | null
  created_at: string
  updated_at: string
}

export interface ActivityPlanEvaluation {
  id: string
  plan_id: string
  reviewer_user_id: string
  accuracy: number
  feasibility: number
  campus_fit: number
  clarity: number
  adoptability: number
  overall_score: number
  notes: string
  created_at: string
  updated_at: string
}

export interface ActivityPlanEvaluationSummary {
  total_evaluations: number
  evaluated_plans: number
  average_score: number
  dimension_averages: {
    accuracy: number
    feasibility: number
    campus_fit: number
    clarity: number
    adoptability: number
  }
  by_model: Array<{
    provider: string
    mode: 'mock' | 'real' | 'disabled'
    model: string
    prompt_version: string
    evaluations: number
    evaluated_plans: number
    average_score: number
  }>
  updated_at?: string | null
}

export interface ActivityPlanApprovalResult extends ActivityPlan {
  created_project_id?: string | null
  created_milestone_ids: string[]
  created_content_id?: string | null
}

export type AdminMembershipWriteState = 'active' | 'disabled'
export type InvitationRole = 'member' | 'editor' | 'administrator'
export type InvitationStatus = 'pending' | 'accepted' | 'expired' | 'revoked'
export type EmailDeliveryStatus = 'disabled' | 'pending' | 'sent' | 'failed'

export interface EmailDelivery {
  status: EmailDeliveryStatus
  adapter: 'disabled' | 'smtp'
  attempts: number
  last_error?: string
  last_attempt_at?: string
  sent_at?: string
}

export interface EmailAdapterStatus {
  driver: 'disabled' | 'smtp'
  enabled: boolean
  configured: boolean
  from_address?: string
  from_name?: string
  security?: 'starttls' | 'tls' | 'none'
}

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

export interface AdminInvitationSummary extends Invitation {
  delivery: EmailDelivery
}

export interface ManagedRuntimeItem {
  key: 'database' | 'cache' | 'security' | 'server'
  label: string
  state: 'deployment' | 'deferred'
  description: string
}

export interface IntegrationSettings {
  public_web_base_url: string
  source: 'deployment' | 'web'
  email: {
    driver: 'disabled' | 'smtp'
    source: 'deployment' | 'web'
    enabled: boolean
    configured: boolean
    host: string
    port: number
    username: string
    password_configured: boolean
    password_hint?: string
    from_address: string
    from_name: string
    security: 'starttls' | 'tls' | 'none'
    timeout_seconds: number
  }
  storage: {
    driver: 'local' | 's3'
    source: 'deployment' | 'web'
    configured: boolean
    endpoint: string
    access_key_configured: boolean
    access_key_hint?: string
    secret_key_configured: boolean
    secret_key_hint?: string
    bucket: string
    region: string
    use_ssl: boolean
  }
  managed_runtime: ManagedRuntimeItem[]
  updated_at?: string
}

export interface IntegrationSettingsUpdate {
  public_web_base_url: string
  email: {
    driver: 'disabled' | 'smtp'
    host: string
    port: number
    username: string
    password: string
    clear_password: boolean
    from_address: string
    from_name: string
    security: 'starttls' | 'tls' | 'none'
    timeout_seconds: number
  }
  storage: {
    driver: 'local' | 's3'
    endpoint: string
    access_key: string
    secret_key: string
    clear_access_key: boolean
    clear_secret_key: boolean
    bucket: string
    region: string
    use_ssl: boolean
  }
}

export interface IntegrationTestResult {
  section: 'email' | 'storage'
  reachable: boolean
  checked_at: string
}

export interface InvitationTemplate {
  subject_template: string
  body_template: string
  variables: string[]
}

export interface NotificationOutbox {
  id: string
  organization_id: string
  event_type: string
  target_type: string
  target_id: string
  recipient_email: string
  status: 'pending' | 'sending' | 'sent' | 'failed' | 'disabled'
  attempts: number
  last_error?: string
  available_at: string
  last_attempt_at?: string | null
  sent_at?: string | null
  created_at: string
  updated_at: string
}

export interface AdminInvitation extends AdminInvitationSummary {
  invite_url: string
}

export interface BatchInvitationResult {
  index: number
  email: string
  succeeded: boolean
  invitation?: AdminInvitation
  error?: {
    code: string
    message: string
  }
}

export interface BatchInvitationResponse {
  total: number
  succeeded: number
  failed: number
  results: BatchInvitationResult[]
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
  mode: 'disabled' | 'mock' | 'rcon'
  status: 'pending' | 'succeeded' | 'failed'
  attempts: number
  message: string
  last_error: string
  requested_at: string
  completed_at?: string | null
}

export interface AdminServerStatus {
  enabled: boolean
  adapter: string
  mode: 'disabled' | 'mock' | 'rcon'
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
  mode: 'disabled' | 'mock' | 'rcon'
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
  default_organization_id?: string
  roles: Array<'owner' | 'administrator' | 'editor' | 'member'>
}

export interface OrganizationMembership {
  id: string
  slug: string
  name: string
  short_name: string
  roles: Array<'owner' | 'administrator' | 'editor' | 'member'>
  current: boolean
}

export interface TokenPair {
  access_token: string
  token_type: 'Bearer'
  expires_in: number
  user: AuthUser
}

export interface ApplicationPayload {
  type?: 'whitelist' | 'membership'
  class_name?: string
  name: string
  game_id?: string
  qq_number?: string
  email: string
  note?: string
}
