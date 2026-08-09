import type {
  ActivityPlan,
  ActivityPlanEvaluation,
  ActivityPlanEvaluationSummary,
  ActivityPlanSummary,
  AdminApplication,
  AdminContent,
  AdminDashboard,
  AdminInvitation,
  AdminKnowledgeDirectory,
  AdminProject,
  AdminProjectMember,
  AdminProjectMilestone,
  AdminServerStatus,
  AdminUser,
  AIAgentCatalog,
  AIAgentRun,
  AIConfiguration,
  AIKnowledgeResult,
  AuditEvent,
  AuthUser,
  BatchInvitationResponse,
  ContentRevision,
  EmailAdapterStatus,
  KnowledgeArticle,
  KnowledgeDirectory,
  Organization,
  OrganizationMembership,
  Page,
  PortalConfiguration,
  PortalManifest,
  PortalRuntimeConfiguration,
  Project,
  PublicPost,
  PublicContentDetail,
  Resource,
  ServerStatus,
  TokenPair,
  Invitation,
  InvitationTemplate,
  IntegrationSettings,
  IntegrationSettingsUpdate,
  NotificationOutbox,
  MediaAsset,
} from '@/api/types'

const organization: Organization = {
  id: 'org_qutcraft',
  slug: 'qutcraft',
  name: 'QUTCraft Commons',
  short_name: 'QUTCraft',
  tagline: '把社团正在发生的事，认真地呈现出来。',
  introduction: 'QUTCraft 是青岛理工大学的 Minecraft 社团。我们围绕 Minecraft、创作和技术协作，持续建设属于成员的公共项目与知识资产。',
  contact_email: 'contact@qutcraft.example',
  social_links: [
    { label: 'GitHub', href: 'https://github.com/QUTCraft/qutc-platform' },
	{ label: '加入我们', href: 'https://qutcraft.example/join' },
  ],
	is_public: true,
	updated_at: new Date().toISOString(),
}

const campusOrganization: Organization = {
  id: 'org_campus_commons',
  slug: 'campus-commons',
  name: 'Campus Commons',
  short_name: 'Commons',
  tagline: '让组织信息、协作与公共内容持续流动。',
  introduction: '面向校园社团与民间组织的公共门户、内容分发和协作平台。',
  contact_email: 'hello@campus-commons.example',
  social_links: [],
  is_public: true,
  updated_at: new Date().toISOString(),
}

const mockOrganizations: OrganizationMembership[] = [
  { id: 'org_qutcraft', slug: 'qutcraft', name: 'QUTCraft Commons', short_name: 'QUTCraft', roles: ['owner'], current: true },
  { id: 'org_campus_commons', slug: 'campus-commons', name: '校园社团协作中心', short_name: '校园协作中心', roles: ['administrator'], current: false },
]

const posts: PublicPost[] = [
  { id: 'post_cms', title: 'QUTCraft CMS 项目正式启动', excerpt: '从官网、资源分发到服务器适配，我们开始把社团长期积累的内容整理成可持续的公共入口。', category: '社团动态', published_at: '2026-07-14T12:00:00Z', reading_minutes: 4 },
  { id: 'post_build', title: '主城公共区域设计征集', excerpt: '建筑组开放第一轮概念征集：请用一张草图、一段说明，提出你想在主城里留下的公共空间。', category: '活动', published_at: '2026-07-12T08:00:00Z', reading_minutes: 3 },
  { id: 'post_rules', title: '服务器行为准则更新', excerpt: '新版本明确了公共建筑、资源共享与成员协作的边界，欢迎阅读后提出修订建议。', category: '公告', published_at: '2026-07-09T09:30:00Z', reading_minutes: 6 },
]

const projects: Project[] = [
  { id: 'project_cms', title: 'QUTCraft CMS', summary: '面向校园社团与民间组织的公开门户与内容分发系统，QUTCraft 是首个落地案例。', status: 'active', tags: ['Vue 3', 'Go', 'API-first'], updated_at: '2026-07-17T03:00:00Z' },
  { id: 'project_spawn', title: '主城公共区域计划', summary: '把成员作品、活动路线与社区服务设施组织成一个可以长期生长的起点。', status: 'active', tags: ['建筑', '社区共建'], updated_at: '2026-07-15T05:00:00Z' },
  { id: 'project_wiki', title: '社团知识库迁移', summary: '将散落的经验、规则、活动资料和技术笔记逐步整理为可检索的公共知识库。', status: 'research', tags: ['知识库', '信息架构'], updated_at: '2026-07-12T02:00:00Z' },
]

const resources: Resource[] = [
  { id: 'resource_overview', title: 'QUTCraft CMS 产品说明', description: '项目目标、公开门户范围与 MVP 路线。', kind: 'document', size_bytes: 0, updated_at: '2026-07-17T01:00:00Z', download_url: null },
  { id: 'resource_event-kit', title: '社团活动策划模板', description: '用于活动立项、分工和复盘的基础模板。', kind: 'template', size_bytes: 0, updated_at: '2026-07-15T01:00:00Z', download_url: null },
  { id: 'resource_portal-api', title: 'Portal API 快速开始', description: '为自定义门户开发者准备的接口与 Manifest 示例。', kind: 'package', size_bytes: 0, updated_at: '2026-07-12T01:00:00Z', download_url: null },
]

const knowledge: KnowledgeArticle[] = [
  { id: 'knowledge_handoff', title: '如何让社团项目可交接', summary: '从目标、角色、决策记录到发布节奏，建立不依赖个人记忆的项目协作方式。', category: '项目协作', updated_at: '2026-07-16T02:00:00Z', reading_minutes: 8 },
  { id: 'knowledge_portal', title: 'Portal API 的公开能力边界', summary: '哪些内容能被主题门户消费，哪些数据必须只留在后台。', category: '开发规范', updated_at: '2026-07-14T02:00:00Z', reading_minutes: 6 },
  { id: 'knowledge_server', title: '服务器公开状态的设计原则', summary: '门户展示状态、申请入口；后台处理审核与命令执行。', category: '服务器', updated_at: '2026-07-11T02:00:00Z', reading_minutes: 5 },
]

const knowledgeDirectories: KnowledgeDirectory[] = [
  { id: 'knowledge_directory_collaboration', name: '项目协作', slug: 'collaboration', description: '项目目标、角色与交接记录。', article_count: 1, updated_at: '2026-07-16T02:00:00Z' },
  { id: 'knowledge_directory_technology', name: '技术规范', slug: 'technology', description: '接口、前端和部署规范。', article_count: 0, updated_at: '2026-07-14T02:00:00Z' },
  { id: 'knowledge_directory_community', name: '社团实践', slug: 'community', description: '适用于组织日常协作的经验。', article_count: 0, updated_at: '2026-07-11T02:00:00Z' },
]

let adminKnowledgeDirectories: AdminKnowledgeDirectory[] = knowledgeDirectories.map((directory, index) => ({
  ...directory,
  parent_id: '',
  sort_order: (index + 1) * 10,
  is_public: true,
}))

const contentDetails: Record<string, PublicContentDetail> = {
  post_cms: { id: 'post_cms', title: 'QUTCraft CMS 项目正式启动', type: 'news', category: '社团动态', excerpt: posts[0].excerpt, body: 'QUTCraft CMS 内容闭环演示内容。我们将官网、资源分发与可选的服务器适配能力拆成清晰边界，让社团长期积累的内容能够被持续维护和公开访问。', published_at: posts[0].published_at, updated_at: posts[0].published_at, reading_minutes: 4 },
  knowledge_handoff: { id: 'knowledge_handoff', title: '如何让社团项目可交接', type: 'knowledge', category: '项目协作', excerpt: knowledge[0].summary, body: '从目标、角色、决策记录到发布节奏，建立不依赖个人记忆的知识库。每次交接都应该留下可复用的背景、当前状态和下一步行动。', published_at: knowledge[0].updated_at, updated_at: knowledge[0].updated_at, reading_minutes: 8 },
  resource_overview: { id: 'resource_overview', title: 'QUTCraft CMS 产品说明', type: 'resource', category: 'document', excerpt: resources[0].description, body: 'QUTCraft CMS 的公开产品说明与接入资料。当前资源条目没有关联可下载文件，请以管理端上传资源文件后生成受控下载地址。', published_at: resources[0].updated_at, updated_at: resources[0].updated_at, reading_minutes: 3, asset: null, download_url: null },
}

const serverStatus: ServerStatus = {
  enabled: true,
  label: 'QUTCraft Java 生存服',
  state: 'online',
  version: 'Java 1.21.x',
  online_players: 18,
  max_players: 60,
  updated_at: '2026-07-17T04:10:00Z',
  apply_url: '#join',
}

let adminContent: AdminContent[] = [
  { id: 'content_001', title: 'QUTCraft CMS 项目正式启动', type: 'news', status: 'published', author: 'BBKarasu', updated_at: '2026-07-17T03:00:00Z' },
  { id: 'content_002', title: '暑期建筑活动资源包', type: 'resource', status: 'review', author: 'Lin', updated_at: '2026-07-16T08:00:00Z' },
  { id: 'content_003', title: '自定义门户接入约定', type: 'knowledge', status: 'draft', author: 'Mori', updated_at: '2026-07-15T03:00:00Z' },
]

const contentRevisions: Record<string, ContentRevision[]> = {}
const invitationTemplate: InvitationTemplate = {
  subject_template: '【{{organization}}】邀请加入组织',
  body_template: '你好，\n\n你收到了加入 {{organization}} 的邀请。\n角色：{{role}}\n邀请链接：{{invite_url}}\n有效期至：{{expires_at}}\n',
  variables: ['organization', 'role', 'invite_url', 'expires_at'],
}
let notificationOutbox: NotificationOutbox[] = []
let integrationSettings: IntegrationSettings = {
  public_web_base_url: window.location.origin,
  source: 'deployment',
  email: {
    driver: 'disabled', source: 'deployment', enabled: false, configured: false,
    host: '', port: 587, username: '', password_configured: false,
    from_address: '', from_name: 'QUTCraft', security: 'starttls', timeout_seconds: 8,
  },
  storage: {
    driver: 'local', source: 'deployment', configured: true, endpoint: '',
    access_key_configured: false, secret_key_configured: false,
    bucket: '', region: '', use_ssl: false,
  },
  managed_runtime: [
    { key: 'database', label: 'MySQL 数据库', state: 'deployment', description: '启动根基，由部署维护；网页仅使用当前连接。' },
    { key: 'cache', label: 'Redis 缓存', state: 'deployment', description: '启动根基，由部署维护；可通过健康检查确认状态。' },
    { key: 'security', label: 'JWT、CORS 与限流', state: 'deployment', description: '安全边界，由部署维护，修改后需要重启 API。' },
    { key: 'server', label: '服务器命令适配器', state: 'deferred', description: 'RCON 已按项目计划延期，当前保持安全 Mock。' },
  ],
}
const assetDownloadStats: Record<string, { download_count: number; last_downloaded_at: string | null }> = {}
let adminAssets: MediaAsset[] = []

function recordContentRevision(content: AdminContent, reason: ContentRevision['reason']) {
  const revisions = contentRevisions[content.id] ?? []
  const revision: ContentRevision = {
    id: `revision_${content.id}_${Date.now()}_${revisions.length + 1}`,
    content_id: content.id,
    version: revisions.length + 1,
    created_by: mockUser?.id ?? 'user_bk',
    reason,
    title: content.title,
    type: content.type,
    category: content.category ?? '',
    knowledge_directory_id: content.knowledge_directory_id ?? null,
    status: content.status,
    excerpt: content.excerpt ?? '',
    body: content.body ?? '',
    published_at: content.published_at ?? null,
    created_at: new Date().toISOString(),
  }
  contentRevisions[content.id] = [revision, ...revisions]
  content.revision_count = contentRevisions[content.id].length
  return revision
}

function ensureContentRevisions(content: AdminContent) {
  if (!contentRevisions[content.id]?.length) recordContentRevision(content, 'create')
  else content.revision_count = contentRevisions[content.id].length
}

const adminUsers: AdminUser[] = [
  { id: 'user_bk', name: 'BBKarasu', email: 'gdd233@qq.com', role: 'owner', state: 'active', joined_at: '2026-07-14T01:00:00Z' },
  { id: 'user_lin', name: 'Lin', email: 'lin@qutcraft.example', role: 'editor', state: 'active', joined_at: '2026-07-15T01:00:00Z' },
  { id: 'user_mori', name: 'Mori', email: 'mori@qutcraft.example', role: 'administrator', state: 'active', joined_at: '2026-07-15T01:00:00Z' },
  { id: 'user_nova', name: 'Nova', email: 'nova@qutcraft.example', role: 'member', state: 'invited', joined_at: '2026-07-16T01:00:00Z' },
]

const auditEvents: AuditEvent[] = [
  { id: 'audit_001', actor_user_id: 'user_bk', actor_name: 'BBKarasu', action: 'content.published', target_type: 'content', target_id: 'content_001', result: 'success', request_id: 'mock-request-content-001', created_at: '2026-07-30T07:30:00Z' },
  { id: 'audit_002', actor_user_id: 'user_mori', actor_name: 'Mori', action: 'membership.invite', target_type: 'invitation', target_id: 'invite_001', result: 'success', request_id: 'mock-request-invite-001', created_at: '2026-07-30T06:10:00Z' },
  { id: 'audit_003', actor_user_id: 'user_bk', actor_name: 'BBKarasu', action: 'server.command', target_type: 'server', target_id: '', result: 'accepted', request_id: 'mock-request-server-001', created_at: '2026-07-29T13:20:00Z' },
]

const aiAgentCatalog: AIAgentCatalog = {
  agents: [
    {
      id: 'agent_content_copilot', key: 'content-copilot', name: '内容协作智能体',
      purpose: '根据当前组织内已授权的知识资料生成带引用的 Markdown 内容提案；结果必须由人工确认。',
      system_policy_version: 'content-copilot/v1', allowed_tool_keys: ['knowledge.search', 'knowledge.read'],
      model_profile: 'content-generation', enabled: true,
    },
    {
      id: 'agent_activity_planner', key: 'activity-planner', name: '校园活动策划智能体',
      purpose: '生成带引用的活动方案，并提出须由人工批准的项目、里程碑和公告草稿。',
        system_policy_version: 'activity-planner/v2',
      allowed_tool_keys: ['knowledge.search', 'knowledge.read', 'project.create_proposal', 'milestone.create_proposal', 'content.create_draft_proposal'],
      model_profile: 'activity-planning', enabled: true,
    },
  ],
  provider: {
    provider: 'mock',
    mode: 'mock',
    model: 'mock-content-v1',
    enabled: true,
    configured: true,
  },
}
let aiConfiguration: AIConfiguration = {
  enabled: true,
  run_limit_per_hour: 20,
  request_timeout_seconds: 30,
  max_sources: 10,
  max_context_characters: 30000,
  provider: structuredClone(aiAgentCatalog.provider),
  provider_config: {
    driver: 'mock',
    base_url: '',
    model: 'mock-content-v1',
    api_key_configured: true,
    api_key_hint: '••••••mock',
    source: 'server',
  },
}
const aiRuns: Record<string, AIAgentRun> = {}
const activityPlans: Record<string, ActivityPlan> = {}
const activityPlanEvaluations: Record<string, ActivityPlanEvaluation> = {}

let adminInvitations: AdminInvitation[] = []

let adminProjects: AdminProject[] = projects.map((project) => ({ ...project, is_public: true, owner: 'BBKarasu', member_count: 1, milestone_count: project.id === 'project_cms' ? 2 : 0 }))

const adminProjectMembers: Record<string, AdminProjectMember[]> = Object.fromEntries(adminProjects.map((project) => [project.id, [{ user_id: 'user_bk', name: 'BBKarasu', email: 'gdd233@qq.com', state: 'active', role: 'owner', assigned_at: '2026-07-14T01:00:00Z' }]]))
const adminProjectMilestones: Record<string, AdminProjectMilestone[]> = {
  project_cms: [
    { id: 'milestone_cms_api', project_id: 'project_cms', title: '完成 API 合同与后台项目闭环', status: 'active', due_at: '2026-08-09T00:00:00Z', completed_at: null, updated_at: '2026-07-26T04:00:00Z' },
    { id: 'milestone_cms_review', project_id: 'project_cms', title: '完成前端联调与回归检查', status: 'planned', due_at: '2026-08-16T00:00:00Z', completed_at: null, updated_at: '2026-07-26T04:00:00Z' },
  ],
}

let applications: AdminApplication[] = [
  { id: 'application_001', applicant: 'Yukino', type: 'whitelist', submitted_at: '2026-07-17T02:30:00Z', note: '希望参与周末建筑测试。', status: 'pending', decision_reason: '' },
  { id: 'application_002', applicant: 'Dawn', type: 'membership', submitted_at: '2026-07-16T10:00:00Z', note: '想加入资源整理与 Wiki 维护。', status: 'pending', decision_reason: '' },
  { id: 'application_003', applicant: 'Kite', type: 'whitelist', submitted_at: '2026-07-15T08:00:00Z', note: '已参加过新生联机活动。', status: 'approved', decision_reason: '资料符合要求。' },
]

const adminServer: AdminServerStatus = {
  enabled: true,
  adapter: 'minecraft-mock',
  mode: 'mock',
  label: 'QUTCraft Minecraft Mock',
  state: 'maintenance',
  online_players: 0,
  max_players: 60,
  updated_at: '2026-07-28T04:10:00Z',
}

const defaultPortalManifest: PortalManifest = {
  schema: 'qutc.portal/v1',
  id: 'qutcraft-md3',
  version: '0.1.0',
  display_name: 'QUTCraft MD3 Portal',
  entry: '/index.html',
  theme: { mode: 'md3' },
  capabilities: ['organization.read', 'public_content.read', 'projects.read', 'assets.read', 'knowledge.read', 'server.status.read'],
  fallback: 'md3',
}
let portalConfiguration: PortalConfiguration = {
  draft_manifest: null,
  active_manifest: null,
  active: false,
}

const mockUserKey = 'qutc.mock_user'
const savedMockUser = () => { try { return JSON.parse(window.localStorage.getItem(mockUserKey) ?? 'null') as AuthUser | null } catch { return null } }
let mockUser: AuthUser | null = savedMockUser()
const saveMockUser = (user: AuthUser | null) => { mockUser = user; if (user) window.localStorage.setItem(mockUserKey, JSON.stringify(user)); else window.localStorage.removeItem(mockUserKey) }
const authPair = (user: AuthUser): TokenPair => ({ access_token: 'mock-access-token', token_type: 'Bearer', expires_in: 900, user })
const requireMockAdmin = () => { if (!mockUser) throw new Error('请先登录后再访问管理工作台。') }

const page = <T>(items: T[]): Page<T> => ({ items, page: 1, page_size: 20, total: items.length })
const wait = () => new Promise((resolve) => window.setTimeout(resolve, 160))

export async function mockGet<T>(path: string): Promise<T> {
  await wait()
  const requestUrl = new URL(path, 'http://mock.local')
  if (path.endsWith('/auth/me')) { if (!mockUser) throw new Error('当前会话已失效。'); return mockUser as T }
  if (path.endsWith('/auth/organizations')) {
    requireMockAdmin()
    return mockOrganizations.map((item) => ({ ...item, current: item.id === mockUser?.organization_id })) as T
  }
  if (path.includes('/admin/')) requireMockAdmin()
  if (path.endsWith('/admin/dashboard')) {
    const currentOrganization = mockOrganizations.find((item) => item.id === mockUser?.organization_id)
    const isQutcraftOrganization = currentOrganization?.slug === 'qutcraft'
    const dashboard: AdminDashboard = {
      organization_name: currentOrganization?.name ?? organization.name,
      updated_at: '2026-07-17T04:10:00Z',
      metrics: [
        { label: '活跃成员', value: 24, change: '较上周 +3', tone: 'primary' },
        { label: '已发布内容', value: 38, change: '本周 +5', tone: 'secondary' },
        { label: '待处理申请', value: applications.filter((item) => item.status === 'pending').length, change: '需要你的处理', tone: 'warning' },
        isQutcraftOrganization
          ? { label: '在线玩家', value: adminServer.online_players, change: '服务器状态正常', tone: 'neutral' }
          : { label: '进行中项目', value: 2, change: '当前组织项目', tone: 'neutral' },
      ],
      pending_applications: applications.filter((item) => item.status === 'pending'),
      recent_content: adminContent,
      server: adminServer,
    }
    return dashboard as T
  }
  const revisionDetailMatch = path.match(/\/admin\/content\/([^/]+)\/revisions\/([^/]+)$/)
  if (revisionDetailMatch) {
    const content = adminContent.find((item) => item.id === revisionDetailMatch[1])
    if (!content) throw new Error('内容不存在。')
    ensureContentRevisions(content)
    const revision = contentRevisions[revisionDetailMatch[1]]?.find((item) => item.id === revisionDetailMatch[2])
    if (!revision) throw new Error('修订版本不存在。')
    return structuredClone(revision) as T
  }
  const revisionListMatch = path.match(/\/admin\/content\/([^/]+)\/revisions(?:\?.*)?$/)
  if (revisionListMatch) {
    const content = adminContent.find((item) => item.id === revisionListMatch[1])
    if (!content) throw new Error('内容不存在。')
    ensureContentRevisions(content)
    const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
    const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
    const items = contentRevisions[content.id] ?? []
    const start = (pageNumber - 1) * pageSize
    return { items: structuredClone(items.slice(start, start + pageSize)), page: pageNumber, page_size: pageSize, total: items.length } as T
  }
  const contentDetailMatch = path.match(/\/admin\/content\/([^/]+)$/)
  if (contentDetailMatch) {
    const content = adminContent.find((item) => item.id === contentDetailMatch[1])
    if (!content) throw new Error('内容不存在。')
    ensureContentRevisions(content)
    return structuredClone(content) as T
  }
	if (requestUrl.pathname.endsWith('/admin/content')) return page(adminContent) as T
	if (requestUrl.pathname.endsWith('/admin/assets')) {
		const search = requestUrl.searchParams.get('query')?.trim().toLowerCase() ?? ''
		const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
		const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
		const filtered = adminAssets.filter((item) => !search || item.original_name.toLowerCase().includes(search))
		const start = (pageNumber - 1) * pageSize
		return { items: structuredClone(filtered.slice(start, start + pageSize)), page: pageNumber, page_size: pageSize, total: filtered.length } as T
	}
	if (requestUrl.pathname.endsWith('/admin/knowledge/directories')) return page(adminKnowledgeDirectories) as T
  if (requestUrl.pathname.endsWith('/admin/users')) return page(adminUsers) as T
  if (requestUrl.pathname.endsWith('/admin/invitations')) {
    const status = requestUrl.searchParams.get('status')
    const invitations = status ? adminInvitations.filter((item) => item.status === status) : adminInvitations
    return page(structuredClone(invitations)) as T
  }
  if (requestUrl.pathname.endsWith('/admin/audit-events')) {
    const action = requestUrl.searchParams.get('action')
    const targetType = requestUrl.searchParams.get('target_type')
    const result = requestUrl.searchParams.get('result')
    const actorUserID = requestUrl.searchParams.get('actor_user_id')
    const requestID = requestUrl.searchParams.get('request_id')
    const dateFrom = requestUrl.searchParams.get('date_from')
    const dateTo = requestUrl.searchParams.get('date_to')
    const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
    const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
    const filtered = auditEvents.filter((item) => {
      const date = item.created_at.slice(0, 10)
      return (!action || item.action === action)
        && (!targetType || item.target_type === targetType)
        && (!result || item.result === result)
        && (!actorUserID || item.actor_user_id === actorUserID)
        && (!requestID || item.request_id === requestID)
        && (!dateFrom || date >= dateFrom)
        && (!dateTo || date <= dateTo)
    })
    const start = (pageNumber - 1) * pageSize
    return { items: filtered.slice(start, start + pageSize), page: pageNumber, page_size: pageSize, total: filtered.length } as T
  }
  if (path.endsWith('/admin/ai/config')) return structuredClone(aiConfiguration) as T
  if (path.endsWith('/admin/ai/agents')) return structuredClone(aiAgentCatalog) as T
  if (requestUrl.pathname.endsWith('/admin/ai/activity-plans/evaluation-summary')) {
    const evaluations = Object.values(activityPlanEvaluations)
    const average = (field: keyof Pick<ActivityPlanEvaluation, 'accuracy' | 'feasibility' | 'campus_fit' | 'clarity' | 'adoptability' | 'overall_score'>) => evaluations.length
      ? evaluations.reduce((sum, item) => sum + item[field], 0) / evaluations.length
      : 0
    const groups = new Map<string, ActivityPlanEvaluationSummary['by_model'][number]>()
    for (const evaluation of evaluations) {
      const plan = activityPlans[evaluation.plan_id]
      if (!plan) continue
      const key = [plan.run.provider, plan.run.mode, plan.run.model, plan.run.prompt_version].join(':')
      const existing = groups.get(key)
      if (existing) {
        existing.evaluations += 1
        existing.evaluated_plans += 1
        existing.average_score = ((existing.average_score * (existing.evaluations - 1)) + evaluation.overall_score) / existing.evaluations
      } else {
        groups.set(key, { provider: plan.run.provider, mode: plan.run.mode, model: plan.run.model, prompt_version: plan.run.prompt_version, evaluations: 1, evaluated_plans: 1, average_score: evaluation.overall_score })
      }
    }
    return {
      total_evaluations: evaluations.length,
      evaluated_plans: new Set(evaluations.map((item) => item.plan_id)).size,
      average_score: average('overall_score'),
      dimension_averages: { accuracy: average('accuracy'), feasibility: average('feasibility'), campus_fit: average('campus_fit'), clarity: average('clarity'), adoptability: average('adoptability') },
      by_model: [...groups.values()],
      updated_at: evaluations.sort((left, right) => right.updated_at.localeCompare(left.updated_at))[0]?.updated_at ?? null,
    } as T
  }
  if (requestUrl.pathname.endsWith('/admin/ai/activity-plans')) {
    const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
    const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
    const items: ActivityPlanSummary[] = Object.values(activityPlans)
      .sort((left, right) => right.created_at.localeCompare(left.created_at))
      .map((plan) => ({
        id: plan.id, title: plan.title, status: plan.status, starts_at: plan.starts_at, ends_at: plan.ends_at,
        provider: plan.run.provider, mode: plan.run.mode, model: plan.run.model, prompt_version: plan.run.prompt_version,
        project_id: plan.project_id, announcement_content_id: plan.announcement_content_id,
        has_my_evaluation: Boolean(activityPlanEvaluations[plan.id]),
        my_evaluation_score: activityPlanEvaluations[plan.id]?.overall_score ?? null,
        created_at: plan.created_at, updated_at: plan.updated_at,
      }))
    const start = (pageNumber - 1) * pageSize
    return { items: items.slice(start, start + pageSize), page: pageNumber, page_size: pageSize, total: items.length } as T
  }
  const activityEvaluationMatch = path.match(/\/admin\/ai\/activity-plans\/([^/]+)\/evaluation$/)
  if (activityEvaluationMatch) {
    if (!activityPlans[activityEvaluationMatch[1]]) throw new Error('活动策划不存在。')
    return structuredClone(activityPlanEvaluations[activityEvaluationMatch[1]] ?? null) as T
  }
  const activityPlanMatch = path.match(/\/admin\/ai\/activity-plans\/([^/]+)$/)
  if (activityPlanMatch) {
    const plan = activityPlans[activityPlanMatch[1]]
    if (!plan) throw new Error('活动策划不存在。')
    return structuredClone(plan) as T
  }
  const aiRunMatch = path.match(/\/admin\/ai\/runs\/([^/]+)$/)
  if (aiRunMatch) {
    const run = aiRuns[aiRunMatch[1]]
    if (!run) throw new Error('智能体运行不存在。')
    return structuredClone(run) as T
  }
  const invitationMatch = path.match(/\/api\/v1\/invitations\/([^/]+)$/)
  if (invitationMatch) {
    const invitation = adminInvitations.find((item) => item.invite_url.endsWith(invitationMatch[1]))
    if (!invitation || invitation.status !== 'pending') throw new Error('邀请链接不存在或已失效。')
    const { invite_url: _inviteUrl, ...preview } = invitation
    return preview as Invitation as T
  }
  const projectMembersMatch = path.match(/\/admin\/projects\/([^/]+)\/members$/)
  if (projectMembersMatch) return page(adminProjectMembers[projectMembersMatch[1]] ?? []) as T
  const projectMilestonesMatch = path.match(/\/admin\/projects\/([^/]+)\/milestones$/)
  if (projectMilestonesMatch) return page(adminProjectMilestones[projectMilestonesMatch[1]] ?? []) as T
  if (path.endsWith('/admin/projects')) return page(adminProjects) as T
  if (new URL(path, 'http://mock.local').pathname.endsWith('/admin/applications')) {
    const status = requestUrl.searchParams.get('status')
    const applicationType = requestUrl.searchParams.get('type')
    const syncStatus = requestUrl.searchParams.get('server_sync_status')
    const query = requestUrl.searchParams.get('query')?.trim().toLowerCase() ?? ''
    const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
    const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
    const filtered = applications.filter((item) => {
      if (status && item.status !== status) return false
      if (applicationType && item.type !== applicationType) return false
      if (syncStatus === 'none' && item.server_sync) return false
      if (syncStatus && syncStatus !== 'none' && item.server_sync?.status !== syncStatus) return false
      if (query && ![item.applicant, item.game_id, item.email, item.qq_number].some((value) => value?.toLowerCase().includes(query))) return false
      return true
    })
    const start = (pageNumber - 1) * pageSize
    return { items: filtered.slice(start, start + pageSize), page: pageNumber, page_size: pageSize, total: filtered.length } as T
  }
  if (path.endsWith('/admin/server/status')) return adminServer as T
  if (path.endsWith('/admin/notifications/email/status')) {
    return { driver: integrationSettings.email.driver, enabled: integrationSettings.email.enabled, configured: integrationSettings.email.configured, from_address: integrationSettings.email.from_address, from_name: integrationSettings.email.from_name, security: integrationSettings.email.security } as EmailAdapterStatus as T
  }
	if (path.endsWith('/admin/integrations')) return structuredClone(integrationSettings) as T
	if (path.endsWith('/admin/notifications/invitation-template')) return structuredClone(invitationTemplate) as T
	if (requestUrl.pathname.endsWith('/admin/notifications/outbox')) {
		const status = requestUrl.searchParams.get('status')
		const pageNumber = Math.max(1, Number(requestUrl.searchParams.get('page') ?? 1))
		const pageSize = Math.min(100, Math.max(1, Number(requestUrl.searchParams.get('page_size') ?? 20)))
		const items = status ? notificationOutbox.filter((item) => item.status === status) : notificationOutbox
		const start = (pageNumber - 1) * pageSize
		return { items: structuredClone(items.slice(start, start + pageSize)), page: pageNumber, page_size: pageSize, total: items.length } as T
	}
	const assetStatsMatch = path.match(/\/admin\/assets\/([^/]+)\/stats$/)
	if (assetStatsMatch) {
		const stats = assetDownloadStats[assetStatsMatch[1]] ?? { download_count: 0, last_downloaded_at: null }
		return { id: assetStatsMatch[1], content_id: null, ...stats } as T
	}
	if (path.endsWith('/admin/organization')) return structuredClone(mockUser?.organization_id === campusOrganization.id ? campusOrganization : organization) as T
  if (path.endsWith('/admin/portal/config')) return structuredClone(portalConfiguration) as T
  if (path.endsWith('/configuration')) {
    const runtime: PortalRuntimeConfiguration = {
      manifest: structuredClone(portalConfiguration.active_manifest ?? defaultPortalManifest),
      source: portalConfiguration.active_manifest ? 'active' : 'default',
      activated_at: portalConfiguration.activated_at,
    }
    return runtime as T
  }
  const contentMatch = path.match(/\/organizations\/[^/]+\/content\/([^/]+)$/)
  if (contentMatch) {
    const detail = contentDetails[contentMatch[1]]
    if (!detail) throw new Error('公开内容不存在或尚未发布。')
    return detail as T
  }
  const organizationMatch = requestUrl.pathname.match(/\/organizations\/([^/]+)$/)
  if (organizationMatch) return structuredClone(organizationMatch[1] === campusOrganization.slug ? campusOrganization : organization) as T
  if (requestUrl.pathname.endsWith('/posts')) return page(posts) as T
  if (requestUrl.pathname.endsWith('/projects')) return page(projects) as T
  if (requestUrl.pathname.endsWith('/resources')) return page(resources) as T
  if (requestUrl.pathname.endsWith('/knowledge/articles')) return page(knowledge) as T
  if (requestUrl.pathname.endsWith('/knowledge/directories')) return page(knowledgeDirectories) as T
  if (requestUrl.pathname.endsWith('/server-status')) return serverStatus as T
  throw new Error(`Mock endpoint not implemented: ${path}`)
}

export async function mockPost<T>(path: string, body?: unknown): Promise<T> {
  await wait()
  if (path.endsWith('/auth/login')) {
    const payload = body as { email: string; password: string }
    if (payload.email !== 'admin@qutcraft.local' || payload.password !== 'demo-admin-pass') throw new Error('演示账号或密码错误。')
    const user: AuthUser = { id: 'user_bk', email: payload.email, display_name: 'BBKarasu', organization_id: 'org_qutcraft', roles: ['owner'] }
    saveMockUser(user)
    return authPair(user) as T
  }
  if (path.endsWith('/auth/register')) {
    const payload = body as { email: string; display_name: string; invitation_token?: string }
    const invitation = payload.invitation_token ? adminInvitations.find((item) => item.invite_url.endsWith(payload.invitation_token ?? '')) : undefined
    if (payload.invitation_token && !invitation) throw new Error('邀请链接不存在或已失效。')
    if (invitation) invitation.status = 'accepted'
    const user: AuthUser = { id: `user_${Date.now()}`, email: payload.email, display_name: payload.display_name, organization_id: 'org_qutcraft', roles: [invitation?.role ?? 'member'] }
    saveMockUser(user)
    return authPair(user) as T
  }
  if (path.endsWith('/auth/refresh')) { if (!mockUser) throw new Error('刷新令牌已失效。'); return authPair(mockUser) as T }
  if (path.endsWith('/auth/logout')) { saveMockUser(null); return { revoked: true } as T }
  if (path.endsWith('/auth/switch-organization')) {
    requireMockAdmin()
    const payload = body as { organization_id?: string }
    const target = mockOrganizations.find((item) => item.id === payload.organization_id)
    if (!target || !mockUser) throw new Error('当前账户不是该组织的有效成员。')
    const user: AuthUser = { ...mockUser, organization_id: target.id, roles: [...target.roles] }
    saveMockUser(user)
    return authPair(user) as T
  }
	if (path.endsWith('/apply')) return { id: `application_${Date.now()}`, status: 'pending', submitted_at: new Date().toISOString() } as T
	if (path.includes('/admin/')) requireMockAdmin()
	if (path.endsWith('/admin/integrations/test')) {
		const section = (body as { section?: string }).section
		if (section !== 'email' && section !== 'storage') throw new Error('未知的服务接入类型。')
		if (section === 'email' && !integrationSettings.email.configured) throw new Error('请先启用并保存完整的 SMTP 配置。')
		if (section === 'storage' && !integrationSettings.storage.configured) throw new Error('请先保存完整的存储配置。')
		return { section, reachable: true, checked_at: new Date().toISOString() } as T
	}
	const notificationRetryMatch = path.match(/\/admin\/notifications\/outbox\/([^/]+)\/retry$/)
	if (notificationRetryMatch) {
		const item = notificationOutbox.find((notification) => notification.id === notificationRetryMatch[1])
		if (!item) throw new Error('通知记录不存在。')
		if (!['failed', 'disabled'].includes(item.status)) throw new Error('当前通知状态不可重试。')
		Object.assign(item, { status: 'pending', attempts: 0, last_error: '', sent_at: null, available_at: new Date().toISOString(), updated_at: new Date().toISOString() })
		return structuredClone(item) as T
	}
	const restoreRevisionMatch = path.match(/\/admin\/content\/([^/]+)\/revisions\/([^/]+)\/restore$/)
	if (restoreRevisionMatch) {
		const content = adminContent.find((item) => item.id === restoreRevisionMatch[1])
		if (!content) throw new Error('内容不存在。')
		ensureContentRevisions(content)
		const revision = contentRevisions[content.id]?.find((item) => item.id === restoreRevisionMatch[2])
		if (!revision) throw new Error('修订版本不存在。')
		Object.assign(content, {
			title: revision.title,
			type: revision.type,
			category: revision.category,
			knowledge_directory_id: revision.knowledge_directory_id ?? null,
			excerpt: revision.excerpt,
			body: revision.body ?? '',
			status: 'draft',
			published_at: null,
			updated_at: new Date().toISOString(),
		})
		recordContentRevision(content, 'restore')
		return structuredClone(content) as T
	}
	if (path.endsWith('/admin/ai/knowledge/search')) {
    const payload = body as { query: string; limit?: number }
    const query = payload.query.trim().toLowerCase()
    const limit = Math.min(20, Math.max(1, payload.limit ?? 10))
    const results: AIKnowledgeResult[] = adminContent
      .filter((item) => item.type === 'knowledge')
      .filter((item) => [item.title, item.category, item.excerpt, item.body].some((value) => value?.toLowerCase().includes(query)))
      .slice(0, limit)
      .map((item) => ({
        source_type: 'content',
        id: item.id,
        title: item.title,
        excerpt: item.excerpt ?? item.body?.slice(0, 180) ?? '',
        status: item.status,
        updated_at: item.updated_at,
      }))
    return results as T
  }
  if (path.endsWith('/admin/ai/activity-plans')) {
    const payload = body as {
      title: string; objective: string; audience: string; venue: string; starts_at?: string; ends_at?: string
      expected_participants: number; budget: string; constraints: string; context_refs: Array<{ type: 'content'; id: string }>
    }
    const sourceItems = payload.context_refs.map((reference) => adminContent.find((item) => item.id === reference.id && item.type === 'knowledge'))
    if (!payload.title.trim() || !payload.objective.trim() || !payload.audience.trim() || sourceItems.some((item) => !item)) throw new Error('活动需求或引用资料不符合约束。')
    const now = new Date().toISOString()
    const runID = `ai_run_${Date.now()}`
    const citations = sourceItems.map((item, index) => ({ id: `ai_citation_${Date.now()}_${index}`, source_type: 'content' as const, source_id: item!.id, title: item!.title, excerpt: item!.excerpt ?? '', source_updated_at: item!.updated_at }))
    const markdown = `# ${payload.title}\n\n> 此方案由开发 Mock 生成，仅用于验证活动策划与人工批准闭环。\n\n## 活动目标与服务价值\n\n${payload.objective}\n\n## 时间流程\n\n1. 完成活动立项和场地确认。\n2. 完成人员分工与宣传。\n3. 执行活动并记录关键事实。\n4. 完成总结与知识沉淀。\n\n## 风险与应急\n\n${payload.constraints || '活动前核对审批、安全、设备和天气风险。'}\n\n## 引用资料\n${sourceItems.map((item) => `- [${item!.title}](qutc://knowledge/${item!.id})`).join('\n')}`
    const run: AIAgentRun = {
      id: runID, agent_key: 'activity-planner', agent_name: '校园活动策划智能体', status: 'succeeded', task: payload.title,
      output_title: payload.title, output_excerpt: payload.objective.slice(0, 180), output_markdown: markdown,
      provider: 'mock', mode: 'mock', model: 'mock-content-v1', prompt_version: 'activity-planner/v2', input_tokens: 64, output_tokens: 128,
      failure_code: '', failure_message: '', request_id: `mock-request-${runID}`, citations, started_at: now, completed_at: now,
      expires_at: new Date(Date.now() + 24 * 3600 * 1000).toISOString(), created_at: now, updated_at: now,
    }
    aiRuns[runID] = run
    const planID = `activity_plan_${Date.now()}`
    const start = payload.starts_at ? new Date(payload.starts_at) : null
    const end = payload.ends_at ? new Date(payload.ends_at) : null
    const due = (base: Date | null, days: number) => base ? new Date(base.getTime() + days * 86400000).toISOString() : null
    const plan: ActivityPlan = {
      id: planID, ...payload, status: 'ready', run,
      proposed_actions: [
        { key: 'create_project', kind: 'project', title: payload.title, description: '创建非公开项目。', requires: [] },
        { key: 'create_preparation_milestone', kind: 'milestone', title: '完成活动方案与审批确认', description: '核对场地、规则与负责人。', due_at: due(start, -14), requires: ['create_project'] },
        { key: 'create_promotion_milestone', kind: 'milestone', title: '完成宣传与人员确认', description: '准备宣传和执行分工。', due_at: due(start, -3), requires: ['create_project'] },
        { key: 'create_execution_milestone', kind: 'milestone', title: '活动执行与现场保障', description: '执行活动并记录异常。', due_at: due(start, 0), requires: ['create_project'] },
        { key: 'create_retrospective_milestone', kind: 'milestone', title: '完成活动总结与知识沉淀', description: '完成总结与改进项。', due_at: due(end, 3), requires: ['create_project'] },
        { key: 'create_announcement_draft', kind: 'content', title: `活动预告｜${payload.title}`, description: '创建 CMS 草稿，不自动发布。', requires: [] },
      ],
      approved_actions: [], project_id: null, announcement_content_id: null, approved_by: null, approved_at: null, created_at: now, updated_at: now,
    }
    activityPlans[planID] = plan
    return structuredClone(plan) as T
  }
  const activityApproveMatch = path.match(/\/admin\/ai\/activity-plans\/([^/]+)\/approve$/)
  if (activityApproveMatch) {
    const plan = activityPlans[activityApproveMatch[1]]
    if (!plan) throw new Error('活动策划不存在。')
    if (plan.status === 'applied') throw new Error('活动方案已经批准执行。')
    const actions = (body as { actions: string[] }).actions
    if (!actions.length) throw new Error('请选择至少一项建议操作。')
    let projectID: string | null = null
    let contentID: string | null = null
    const milestoneIDs: string[] = []
    if (actions.includes('create_project')) {
      projectID = `project_${Date.now()}`
      adminProjects.unshift({ id: projectID, title: plan.title, summary: plan.objective, status: 'research', tags: ['AI活动策划', '校园活动'], is_public: false, owner: mockUser?.display_name ?? 'Owner', member_count: 1, milestone_count: 0, updated_at: new Date().toISOString() })
      adminProjectMembers[projectID] = []
      adminProjectMilestones[projectID] = plan.proposed_actions.filter((action) => action.kind === 'milestone' && actions.includes(action.key)).map((action, index) => {
        const id = `milestone_${Date.now()}_${index}`; milestoneIDs.push(id)
        return { id, project_id: projectID!, title: action.title, status: 'planned', due_at: action.due_at ?? null, completed_at: null, updated_at: new Date().toISOString() }
      })
      adminProjects[0].milestone_count = milestoneIDs.length
    }
    if (actions.includes('create_announcement_draft')) {
      contentID = `content_${Date.now()}`
      adminContent.unshift({ id: contentID, title: `活动预告｜${plan.title}`, type: 'news', category: '校园活动', status: 'draft', author: mockUser?.display_name ?? 'Owner', excerpt: plan.objective, body: plan.run.output_markdown, updated_at: new Date().toISOString() })
    }
    Object.assign(plan, { status: 'applied', approved_actions: actions, project_id: projectID, announcement_content_id: contentID, approved_by: mockUser?.id, approved_at: new Date().toISOString(), updated_at: new Date().toISOString() })
    return { ...structuredClone(plan), created_project_id: projectID, created_milestone_ids: milestoneIDs, created_content_id: contentID } as T
  }
  if (path.endsWith('/admin/ai/runs')) {
    const payload = body as { agent_key: string; task: string; context_refs: Array<{ type: 'content'; id: string }> }
    const sourceItems = payload.context_refs.map((reference) => adminContent.find((item) => item.id === reference.id && item.type === 'knowledge'))
    if (sourceItems.some((item) => !item)) throw new Error('引用资料不存在或不在当前组织的可访问知识范围内。')
    const now = new Date().toISOString()
    const id = `ai_run_${Date.now()}`
    const title = payload.task.trim().slice(0, 80) || 'AI 内容提案'
    const citations = sourceItems.map((item, index) => ({
      id: `ai_citation_${Date.now()}_${index}`,
      source_type: 'content' as const,
      source_id: item!.id,
      title: item!.title,
      excerpt: item!.excerpt ?? '',
      source_updated_at: item!.updated_at,
    }))
    const run: AIAgentRun = {
      id,
      agent_key: payload.agent_key,
      agent_name: '内容协作智能体',
      status: 'succeeded',
      task: payload.task,
      output_title: title,
      output_excerpt: `开发 Mock 提案：${payload.task}`.slice(0, 180),
      output_markdown: `# ${title}\n\n> 此内容由开发 Mock 生成，仅用于验证智能体 API 与权限闭环，不代表真实模型输出。\n\n${sourceItems.map((item) => `- **${item!.title}**：${item!.excerpt ?? ''}`).join('\n')}\n\n## 引用\n${sourceItems.map((item) => `- [${item!.title}](qutc://knowledge/${item!.id})`).join('\n')}`,
      provider: 'mock',
      mode: 'mock',
      model: 'mock-content-v1',
      prompt_version: 'content-copilot/v1',
      input_tokens: Math.ceil(payload.task.length / 4),
      output_tokens: 64,
      failure_code: '',
      failure_message: '',
      request_id: `mock-request-${id}`,
      citations,
      started_at: now,
      completed_at: now,
      expires_at: new Date(Date.now() + 24 * 3600 * 1000).toISOString(),
      created_at: now,
      updated_at: now,
    }
    aiRuns[id] = run
    return structuredClone(run) as T
  }
  const aiRunCancelMatch = path.match(/\/admin\/ai\/runs\/([^/]+)\/cancel$/)
  if (aiRunCancelMatch) {
    const run = aiRuns[aiRunCancelMatch[1]]
    if (!run) throw new Error('智能体运行不存在。')
    if (!['queued', 'running'].includes(run.status)) throw new Error('智能体运行已经结束，不能取消。')
    run.status = 'canceled'
    run.completed_at = new Date().toISOString()
    run.updated_at = run.completed_at
    return structuredClone(run) as T
  }
  if (path.endsWith('/admin/portal/config/enable')) {
    if (!portalConfiguration.draft_manifest) throw new Error('请先保存有效的门户草稿。')
    portalConfiguration = {
      ...portalConfiguration,
      active_manifest: structuredClone(portalConfiguration.draft_manifest),
      active: true,
      activated_by: mockUser?.id,
      activated_at: new Date().toISOString(),
    }
    return structuredClone(portalConfiguration) as T
  }
  if (path.endsWith('/admin/portal/config/restore-default')) {
    const restoredAt = new Date().toISOString()
    portalConfiguration = {
      ...portalConfiguration,
      id: portalConfiguration.id ?? 'portal_config_mock',
      draft_manifest: structuredClone(defaultPortalManifest),
      active_manifest: structuredClone(defaultPortalManifest),
      active: true,
      updated_by: mockUser?.id,
      updated_at: restoredAt,
      activated_by: mockUser?.id,
      activated_at: restoredAt,
    }
    return structuredClone(portalConfiguration) as T
  }
	if (path.endsWith('/admin/content')) {
		const payload = body as Pick<AdminContent, 'title' | 'type' | 'category' | 'knowledge_directory_id' | 'excerpt' | 'body'>
		if (payload.type === 'knowledge' && !payload.knowledge_directory_id) throw new Error('知识库文章必须关联目录。')
		if (payload.knowledge_directory_id && !adminKnowledgeDirectories.some((directory) => directory.id === payload.knowledge_directory_id)) throw new Error('知识库目录不存在。')
		const content: AdminContent = { id: `content_${Date.now()}`, title: payload.title, type: payload.type, category: payload.category, knowledge_directory_id: payload.knowledge_directory_id ?? null, excerpt: payload.excerpt, body: payload.body, status: 'draft', author: mockUser?.display_name ?? 'BBKarasu', updated_at: new Date().toISOString() }
    adminContent = [content, ...adminContent]
		recordContentRevision(content, 'create')
    return content as T
  }
  if (path.endsWith('/admin/invitation-batches')) {
    const payload = body as { invitations: Array<{ email: string; role: AdminInvitation['role']; expires_in_hours?: number }> }
    if (!Array.isArray(payload.invitations) || payload.invitations.length < 1 || payload.invitations.length > 20) throw new Error('批量邀请必须包含 1 到 20 条记录。')
    const results: BatchInvitationResponse['results'] = []
    for (const [index, item] of payload.invitations.entries()) {
      const email = item.email.trim().toLowerCase()
      const existingInvitation = adminInvitations.some((invitation) => invitation.email.toLowerCase() === email && invitation.status === 'pending')
      const existingMember = adminUsers.some((user) => user.email.toLowerCase() === email && user.state === 'active')
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || !['member', 'editor', 'administrator'].includes(item.role)) {
        results.push({ index, email, succeeded: false, error: { code: 'invitation.validation_failed', message: '邮箱、角色或邀请有效期不符合要求。' } })
        continue
      }
      if (existingInvitation || existingMember) {
        results.push({ index, email, succeeded: false, error: { code: existingMember ? 'membership.already_active' : 'invitation.already_pending', message: existingMember ? '该邮箱已经是当前组织的有效成员。' : '该邮箱已有尚未处理的邀请。' } })
        continue
      }
      const token = `mock-invite-${Date.now()}-${index}`
      const invitation: AdminInvitation = {
        id: `invitation_${Date.now()}_${index}`,
        organization_id: 'org_qutcraft', organization_name: organization.name,
        email, role: item.role, status: 'pending',
        expires_at: new Date(Date.now() + (item.expires_in_hours ?? 168) * 3600 * 1000).toISOString(),
        created_at: new Date().toISOString(), invite_url: `/invite/${token}`,
        delivery: { status: 'disabled', adapter: 'disabled', attempts: 0 },
      }
      adminInvitations = [invitation, ...adminInvitations]
      results.push({ index, email, succeeded: true, invitation: structuredClone(invitation) })
    }
    const succeeded = results.filter((item) => item.succeeded).length
    return { total: results.length, succeeded, failed: results.length - succeeded, results } as T
  }
  if (path.endsWith('/admin/invitations')) {
    const payload = body as { email: string; role: AdminInvitation['role']; expires_in_hours?: number }
    const token = `mock-invite-${Date.now()}`
    const invitation: AdminInvitation = {
      id: `invitation_${Date.now()}`,
      organization_id: 'org_qutcraft',
      organization_name: organization.name,
      email: payload.email,
      role: payload.role,
      status: 'pending',
      expires_at: new Date(Date.now() + (payload.expires_in_hours ?? 168) * 3600 * 1000).toISOString(),
      created_at: new Date().toISOString(),
      invite_url: `/invite/${token}`,
      delivery: { status: 'disabled', adapter: 'disabled', attempts: 0 },
    }
    adminInvitations = [invitation, ...adminInvitations]
    return invitation as T
  }
  const invitationRetryMatch = path.match(/\/admin\/invitations\/([^/]+)\/email\/retry$/)
  if (invitationRetryMatch) {
    const invitation = adminInvitations.find((item) => item.id === invitationRetryMatch[1])
    if (!invitation || invitation.status !== 'pending') throw new Error('邀请不存在或已失效。')
    invitation.invite_url = `/invite/mock-invite-retry-${Date.now()}`
    invitation.delivery = {
      status: 'failed',
      adapter: 'smtp',
      attempts: invitation.delivery.attempts + 1,
      last_error: 'Mock 邮件适配器未连接真实 SMTP。',
      last_attempt_at: new Date().toISOString(),
    }
    return structuredClone(invitation) as T
  }
  const invitationAcceptMatch = path.match(/\/api\/v1\/invitations\/([^/]+)\/accept$/)
  if (invitationAcceptMatch) {
    requireMockAdmin()
    const invitation = adminInvitations.find((item) => item.invite_url.endsWith(invitationAcceptMatch[1]))
    if (!invitation || invitation.status !== 'pending') throw new Error('邀请链接不存在或已失效。')
    invitation.status = 'accepted'
    return { ...invitation, membership_id: `membership_${Date.now()}` } as T
  }
  if (path.endsWith('/admin/assets')) {
    const assetID = `asset_${Date.now()}`
    const formData = body as FormData
    const file = formData.get('file') as File | null
    const contentID = String(formData.get('content_id') ?? '') || null
    const now = new Date().toISOString()
    const asset: MediaAsset = {
      id: assetID,
      content_id: contentID,
      original_name: file?.name ?? 'mock-upload.bin',
      mime_type: file?.type || 'application/octet-stream',
      size_bytes: file?.size ?? 0,
      download_count: 0,
      last_downloaded_at: null,
      created_at: now,
      download_url: `/api/v1/admin/assets/${assetID}/download`,
    }
    adminAssets = [asset, ...adminAssets]
    assetDownloadStats[assetID] = { download_count: 0, last_downloaded_at: null }
    return asset as T
  }
  if (path.endsWith('/admin/knowledge/directories')) {
    const payload = body as Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>
    const directory: AdminKnowledgeDirectory = { id: `knowledge_directory_${Date.now()}`, ...payload, updated_at: new Date().toISOString() }
    adminKnowledgeDirectories = [...adminKnowledgeDirectories, directory]
    return directory as T
  }
  const projectMemberCreateMatch = path.match(/\/admin\/projects\/([^/]+)\/members$/)
  if (projectMemberCreateMatch) {
    const project = adminProjects.find((item) => item.id === projectMemberCreateMatch[1])
    const payload = body as { user_id: string; role: AdminProjectMember['role'] }
    const user = adminUsers.find((item) => item.id === payload.user_id && item.state === 'active')
    if (!project || !user) throw new Error('只能添加当前组织中的活跃成员。')
    const members = adminProjectMembers[project.id] ?? (adminProjectMembers[project.id] = [])
    const existing = members.find((item) => item.user_id === user.id)
    if (existing) {
      if (existing.role === 'owner') throw new Error('项目负责人不能通过成员角色接口修改。')
      existing.role = payload.role
      return existing as T
    }
    const member: AdminProjectMember = { user_id: user.id, name: user.name, email: user.email, state: user.state, role: payload.role, assigned_at: new Date().toISOString() }
    members.push(member)
    project.member_count = members.length
    return member as T
  }
  const projectMilestoneCreateMatch = path.match(/\/admin\/projects\/([^/]+)\/milestones$/)
  if (projectMilestoneCreateMatch) {
    const project = adminProjects.find((item) => item.id === projectMilestoneCreateMatch[1])
    if (!project) throw new Error('项目不存在。')
    const payload = body as { title: string; status: AdminProjectMilestone['status']; due_at?: string }
    const now = new Date().toISOString()
    const milestone: AdminProjectMilestone = { id: `milestone_${Date.now()}`, project_id: project.id, title: payload.title, status: payload.status, due_at: payload.due_at || null, completed_at: payload.status === 'completed' ? now : null, updated_at: now }
    const milestones = adminProjectMilestones[project.id] ?? (adminProjectMilestones[project.id] = [])
    milestones.push(milestone)
    project.milestone_count = milestones.length
    return milestone as T
  }
  const contentDecision = path.match(/\/admin\/content\/([^/]+)\/(publish|archive)$/)
  if (contentDecision) {
    const content = adminContent.find((item) => item.id === contentDecision[1])
    if (!content) throw new Error('内容不存在。')
    content.status = contentDecision[2] === 'publish' ? 'published' : 'archived'
    content.published_at = contentDecision[2] === 'publish' ? new Date().toISOString() : null
    content.updated_at = new Date().toISOString()
    recordContentRevision(content, contentDecision[2] === 'publish' ? 'published' : 'archived')
    return content as T
  }
  if (path.endsWith('/admin/projects')) {
    const payload = body as Pick<AdminProject, 'title' | 'summary' | 'status' | 'tags' | 'is_public'>
    const project: AdminProject = { id: `project_${Date.now()}`, ...payload, owner: mockUser?.display_name ?? 'BBKarasu', updated_at: new Date().toISOString() }
    adminProjects = [project, ...adminProjects]
    adminProjectMembers[project.id] = [{ user_id: 'user_bk', name: 'BBKarasu', email: 'gdd233@qq.com', state: 'active', role: 'owner', assigned_at: new Date().toISOString() }]
    adminProjectMilestones[project.id] = []
    project.member_count = 1
    project.milestone_count = 0
    return project as T
  }
  const decision = path.match(/\/admin\/applications\/([^/]+)\/(approve|reject)$/)
  if (decision) {
    const application = applications.find((item) => item.id === decision[1])
    if (!application) throw new Error('Application not found')
    const reason = String((body as { reason?: string } | undefined)?.reason ?? '').trim()
    if (decision[2] === 'reject' && !reason) throw new Error('拒绝申请时必须填写审核原因。')
    application.status = decision[2] === 'approve' ? 'approved' : 'rejected'
    application.decision_reason = reason
    application.decided_at = new Date().toISOString()
    return application as T
  }
  const retrySync = path.match(/\/admin\/applications\/([^/]+)\/server-sync\/retry$/)
  if (retrySync) {
    const application = applications.find((item) => item.id === retrySync[1])
    if (!application?.server_sync || application.server_sync.status !== 'failed') throw new Error('当前服务器同步状态不能重试。')
    application.server_sync.status = 'succeeded'
    application.server_sync.attempts += 1
    application.server_sync.last_error = ''
    application.server_sync.message = 'Mock 适配器已模拟白名单同步。'
    application.server_sync.completed_at = new Date().toISOString()
    return application.server_sync as T
  }
  if (path.endsWith('/admin/server/commands')) {
    const command = (body as { command: string }).command
    adminServer.last_command_at = new Date().toISOString()
    return { accepted: true, executed: false, mode: 'mock', message: `命令“${command}”已被模拟环境记录，未连接真实 RCON。`, executed_at: adminServer.last_command_at } as T
  }
  throw new Error(`Mock endpoint not implemented: ${path}`)
}

export async function mockPatch<T>(path: string, body: unknown): Promise<T> {
  await wait()
  requireMockAdmin()
	if (path.endsWith('/admin/organization')) {
		const currentOrganization = mockUser?.organization_id === campusOrganization.id ? campusOrganization : organization
		Object.assign(currentOrganization, body as Partial<Organization>, { updated_at: new Date().toISOString() })
		return structuredClone(currentOrganization) as T
	}
	if (path.endsWith('/admin/notifications/invitation-template')) {
		Object.assign(invitationTemplate, body as Pick<InvitationTemplate, 'subject_template' | 'body_template'>)
		return structuredClone(invitationTemplate) as T
	}
	if (path.endsWith('/admin/integrations')) {
		const payload = body as IntegrationSettingsUpdate
		const emailConfigured = payload.email.driver === 'smtp'
			&& Boolean(payload.email.host && payload.email.port && payload.email.from_address)
			&& (!payload.email.username || Boolean(payload.email.password || (integrationSettings.email.password_configured && !payload.email.clear_password)))
		const storageConfigured = payload.storage.driver === 'local' || Boolean(
			payload.storage.endpoint && payload.storage.bucket
			&& (payload.storage.access_key || (integrationSettings.storage.access_key_configured && !payload.storage.clear_access_key))
			&& (payload.storage.secret_key || (integrationSettings.storage.secret_key_configured && !payload.storage.clear_secret_key)),
		)
		integrationSettings = {
			...integrationSettings,
			public_web_base_url: payload.public_web_base_url,
			source: 'web',
			email: {
				...integrationSettings.email, ...payload.email, source: 'web',
				enabled: payload.email.driver === 'smtp', configured: payload.email.driver === 'disabled' || emailConfigured,
				password_configured: payload.email.clear_password ? false : Boolean(payload.email.password) || integrationSettings.email.password_configured,
				password_hint: payload.email.clear_password ? undefined : payload.email.password ? `••••••${payload.email.password.slice(-4)}` : integrationSettings.email.password_hint,
			},
			storage: {
				...integrationSettings.storage, ...payload.storage, source: 'web', configured: storageConfigured,
				access_key_configured: payload.storage.clear_access_key ? false : Boolean(payload.storage.access_key) || integrationSettings.storage.access_key_configured,
				access_key_hint: payload.storage.clear_access_key ? undefined : payload.storage.access_key ? `••••••${payload.storage.access_key.slice(-4)}` : integrationSettings.storage.access_key_hint,
				secret_key_configured: payload.storage.clear_secret_key ? false : Boolean(payload.storage.secret_key) || integrationSettings.storage.secret_key_configured,
				secret_key_hint: payload.storage.clear_secret_key ? undefined : payload.storage.secret_key ? `••••••${payload.storage.secret_key.slice(-4)}` : integrationSettings.storage.secret_key_hint,
			},
			updated_at: new Date().toISOString(),
		}
		return structuredClone(integrationSettings) as T
	}
  if (path.endsWith('/admin/ai/config')) {
    const payload = body as Partial<AIConfiguration> & { provider?: AIConfiguration['provider']['provider']; base_url?: string; api_key?: string; model?: string }
    aiConfiguration = {
      ...aiConfiguration,
      enabled: payload.enabled ?? aiConfiguration.enabled,
      run_limit_per_hour: payload.run_limit_per_hour ?? aiConfiguration.run_limit_per_hour,
      request_timeout_seconds: payload.request_timeout_seconds ?? aiConfiguration.request_timeout_seconds,
      max_sources: payload.max_sources ?? aiConfiguration.max_sources,
      max_context_characters: payload.max_context_characters ?? aiConfiguration.max_context_characters,
      provider: payload.provider ? { ...aiConfiguration.provider, provider: payload.provider, mode: payload.provider === 'openai_compatible' ? 'real' : payload.provider, enabled: payload.provider !== 'disabled', configured: true } : aiConfiguration.provider,
      provider_config: payload.provider ? { ...aiConfiguration.provider_config, driver: payload.provider, base_url: payload.base_url ?? aiConfiguration.provider_config.base_url, model: payload.model ?? aiConfiguration.provider_config.model, api_key_configured: Boolean(payload.api_key) || aiConfiguration.provider_config.api_key_configured, api_key_hint: payload.api_key ? '••••••' + payload.api_key.slice(-4) : aiConfiguration.provider_config.api_key_hint } : aiConfiguration.provider_config,
      id: aiConfiguration.id ?? 'ai_config_mock',
      updated_by: mockUser?.id,
      updated_at: new Date().toISOString(),
    }
    return structuredClone(aiConfiguration) as T
  }
  if (path.endsWith('/admin/portal/config')) {
    const payload = body as { manifest: PortalManifest }
    portalConfiguration = {
      ...portalConfiguration,
      id: portalConfiguration.id ?? 'portal_config_mock',
      draft_manifest: structuredClone(payload.manifest ?? defaultPortalManifest),
      updated_by: mockUser?.id,
      updated_at: new Date().toISOString(),
    }
    return structuredClone(portalConfiguration) as T
  }
  const userMatch = path.match(/\/admin\/users\/([^/]+)$/)
  if (userMatch) {
    const user = adminUsers.find((item) => item.id === userMatch[1])
    if (!user) throw new Error('成员不存在。')
    const payload = body as { role: AdminUser['role']; state: 'active' | 'disabled' }
    Object.assign(user, payload)
    return user as T
  }
  const projectMemberMatch = path.match(/\/admin\/projects\/([^/]+)\/members\/([^/]+)$/)
  if (projectMemberMatch) {
    const member = (adminProjectMembers[projectMemberMatch[1]] ?? []).find((item) => item.user_id === projectMemberMatch[2])
    if (!member) throw new Error('项目成员不存在。')
    if (member.role === 'owner') throw new Error('项目负责人不能通过成员角色接口修改。')
    Object.assign(member, body)
    return member as T
  }
  const projectMilestoneMatch = path.match(/\/admin\/projects\/([^/]+)\/milestones\/([^/]+)$/)
  if (projectMilestoneMatch) {
    const milestone = (adminProjectMilestones[projectMilestoneMatch[1]] ?? []).find((item) => item.id === projectMilestoneMatch[2])
    if (!milestone) throw new Error('里程碑不存在。')
    const payload = body as Partial<Pick<AdminProjectMilestone, 'title' | 'status' | 'due_at'>>
    Object.assign(milestone, payload, { due_at: payload.due_at || null, completed_at: payload.status === 'completed' ? new Date().toISOString() : null, updated_at: new Date().toISOString() })
    return milestone as T
  }
  const projectMatch = path.match(/\/admin\/projects\/([^/]+)$/)
  if (projectMatch) {
    const project = adminProjects.find((item) => item.id === projectMatch[1])
    if (!project) throw new Error('项目不存在。')
    Object.assign(project, body, { updated_at: new Date().toISOString() })
    return project as T
  }
  const directoryMatch = path.match(/\/admin\/knowledge\/directories\/([^/]+)$/)
  if (directoryMatch) {
    const directory = adminKnowledgeDirectories.find((item) => item.id === directoryMatch[1])
    if (!directory) throw new Error('知识库目录不存在。')
    Object.assign(directory, body, { updated_at: new Date().toISOString() })
    return directory as T
  }
  const match = path.match(/\/admin\/content\/([^/]+)$/)
  if (!match) throw new Error(`Mock endpoint not implemented: ${path}`)
  const item = adminContent.find((content) => content.id === match[1])
  if (!item) throw new Error('内容不存在。')
	const payload = body as Pick<AdminContent, 'title' | 'type' | 'category' | 'knowledge_directory_id' | 'excerpt' | 'body'>
	if (payload.type === 'knowledge' && !payload.knowledge_directory_id) throw new Error('知识库文章必须关联目录。')
  if (payload.knowledge_directory_id && !adminKnowledgeDirectories.some((directory) => directory.id === payload.knowledge_directory_id)) throw new Error('知识库目录不存在。')
  Object.assign(item, payload, { updated_at: new Date().toISOString() })
  recordContentRevision(item, 'update')
  return item as T
}

export async function mockPut<T>(path: string, body: unknown): Promise<T> {
  await wait()
  requireMockAdmin()
  const activityEvaluationMatch = path.match(/\/admin\/ai\/activity-plans\/([^/]+)\/evaluation$/)
  if (activityEvaluationMatch) {
    const plan = activityPlans[activityEvaluationMatch[1]]
    if (!plan) throw new Error('活动策划不存在。')
    if (plan.status !== 'ready' && plan.status !== 'applied') throw new Error('活动方案尚未生成完成。')
    const payload = body as Pick<ActivityPlanEvaluation, 'accuracy' | 'feasibility' | 'campus_fit' | 'clarity' | 'adoptability' | 'notes'>
    const scores = [payload.accuracy, payload.feasibility, payload.campus_fit, payload.clarity, payload.adoptability]
    if (scores.some((score) => !Number.isInteger(score) || score < 1 || score > 5) || payload.notes.length > 1000) throw new Error('请完成全部五维评分。')
    const existing = activityPlanEvaluations[plan.id]
    const now = new Date().toISOString()
    const evaluation: ActivityPlanEvaluation = {
      id: existing?.id ?? `activity_plan_evaluation_${Date.now()}`, plan_id: plan.id,
      reviewer_user_id: mockUser?.id ?? 'user_bk', ...payload,
      overall_score: scores.reduce((sum, score) => sum + score, 0) / scores.length,
      created_at: existing?.created_at ?? now, updated_at: now,
    }
    activityPlanEvaluations[plan.id] = evaluation
    return structuredClone(evaluation) as T
  }
  throw new Error(`Mock endpoint not implemented: ${path}`)
}

export async function mockDelete<T>(path: string): Promise<T> {
  await wait()
  requireMockAdmin()
  const assetMatch = path.match(/\/admin\/assets\/([^/]+)$/)
  if (assetMatch) {
    const assetIndex = adminAssets.findIndex((item) => item.id === assetMatch[1])
    if (assetIndex < 0) throw new Error('媒体资源不存在。')
    if (adminAssets[assetIndex].content_id) throw new Error('该文件已被内容引用，请先解除引用后再删除。')
    adminAssets.splice(assetIndex, 1)
    return { removed: true, id: assetMatch[1] } as T
  }
  const invitationMatch = path.match(/\/admin\/invitations\/([^/]+)$/)
  if (invitationMatch) {
    const invitation = adminInvitations.find((item) => item.id === invitationMatch[1])
    if (!invitation || invitation.status !== 'pending') throw new Error('只有待处理邀请可以撤销。')
    invitation.status = 'revoked'
    const { invite_url: _inviteUrl, delivery: _delivery, ...result } = invitation
    return structuredClone(result) as T
  }
  const memberMatch = path.match(/\/admin\/projects\/([^/]+)\/members\/([^/]+)$/)
  if (memberMatch) {
    const members = adminProjectMembers[memberMatch[1]] ?? []
    const memberIndex = members.findIndex((item) => item.user_id === memberMatch[2])
    if (memberIndex < 0) throw new Error('项目成员不存在。')
    if (members[memberIndex].role === 'owner') throw new Error('项目负责人不能移除。')
    members.splice(memberIndex, 1)
    const project = adminProjects.find((item) => item.id === memberMatch[1])
    if (project) project.member_count = members.length
    return { removed: true } as T
  }
  const milestoneMatch = path.match(/\/admin\/projects\/([^/]+)\/milestones\/([^/]+)$/)
  if (milestoneMatch) {
    const milestones = adminProjectMilestones[milestoneMatch[1]] ?? []
    const milestoneIndex = milestones.findIndex((item) => item.id === milestoneMatch[2])
    if (milestoneIndex < 0) throw new Error('里程碑不存在。')
    milestones.splice(milestoneIndex, 1)
    const project = adminProjects.find((item) => item.id === milestoneMatch[1])
    if (project) project.milestone_count = milestones.length
    return { removed: true } as T
  }
  throw new Error(`Mock endpoint not implemented: ${path}`)
}
