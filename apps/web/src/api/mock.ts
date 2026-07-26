import type {
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
  AuthUser,
  KnowledgeArticle,
  KnowledgeDirectory,
  Organization,
  Page,
  Project,
  PublicPost,
  PublicContentDetail,
  Resource,
  ServerStatus,
  TokenPair,
  Invitation,
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
    { label: '加入我们', href: '#join' },
  ],
}

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

const adminUsers: AdminUser[] = [
  { id: 'user_bk', name: 'BBKarasu', email: 'gdd233@qq.com', role: 'owner', state: 'active', joined_at: '2026-07-14T01:00:00Z' },
  { id: 'user_lin', name: 'Lin', email: 'lin@qutcraft.example', role: 'editor', state: 'active', joined_at: '2026-07-15T01:00:00Z' },
  { id: 'user_mori', name: 'Mori', email: 'mori@qutcraft.example', role: 'administrator', state: 'active', joined_at: '2026-07-15T01:00:00Z' },
  { id: 'user_nova', name: 'Nova', email: 'nova@qutcraft.example', role: 'member', state: 'invited', joined_at: '2026-07-16T01:00:00Z' },
]

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
  { id: 'application_001', applicant: 'Yukino', type: 'whitelist', submitted_at: '2026-07-17T02:30:00Z', note: '希望参与周末建筑测试。', status: 'pending' },
  { id: 'application_002', applicant: 'Dawn', type: 'membership', submitted_at: '2026-07-16T10:00:00Z', note: '想加入资源整理与 Wiki 维护。', status: 'pending' },
  { id: 'application_003', applicant: 'Kite', type: 'whitelist', submitted_at: '2026-07-15T08:00:00Z', note: '已参加过新生联机活动。', status: 'approved' },
]

const adminServer: AdminServerStatus = {
  enabled: true,
  label: 'QUTCraft Java 生存服',
  state: 'online',
  online_players: 18,
  max_players: 60,
}

const mockUserKey = 'qutc.mock_user'
const savedMockUser = () => { try { return JSON.parse(window.localStorage.getItem(mockUserKey) ?? 'null') as AuthUser | null } catch { return null } }
let mockUser: AuthUser | null = savedMockUser()
const saveMockUser = (user: AuthUser | null) => { mockUser = user; if (user) window.localStorage.setItem(mockUserKey, JSON.stringify(user)); else window.localStorage.removeItem(mockUserKey) }
const authPair = (user: AuthUser): TokenPair => ({ access_token: 'mock-access-token', refresh_token: 'mock-refresh-token', token_type: 'Bearer', expires_in: 900, user })
const requireMockAdmin = () => { if (!mockUser) throw new Error('请先登录后再访问管理工作台。') }

const page = <T>(items: T[]): Page<T> => ({ items, page: 1, page_size: 20, total: items.length })
const wait = () => new Promise((resolve) => window.setTimeout(resolve, 160))

export async function mockGet<T>(path: string): Promise<T> {
  await wait()
  if (path.endsWith('/auth/me')) { if (!mockUser) throw new Error('当前会话已失效。'); return mockUser as T }
  if (path.includes('/admin/')) requireMockAdmin()
  if (path.endsWith('/admin/dashboard')) {
    const dashboard: AdminDashboard = {
      organization_name: organization.name,
      updated_at: '2026-07-17T04:10:00Z',
      metrics: [
        { label: '活跃成员', value: 24, change: '较上周 +3', tone: 'primary' },
        { label: '已发布内容', value: 38, change: '本周 +5', tone: 'secondary' },
        { label: '待处理申请', value: applications.filter((item) => item.status === 'pending').length, change: '需要你的处理', tone: 'warning' },
        { label: '在线玩家', value: adminServer.online_players, change: '服务器状态正常', tone: 'neutral' },
      ],
      pending_applications: applications.filter((item) => item.status === 'pending'),
      recent_content: adminContent,
      server: adminServer,
    }
    return dashboard as T
  }
  if (path.endsWith('/admin/content')) return page(adminContent) as T
  if (path.endsWith('/admin/knowledge/directories')) return page(adminKnowledgeDirectories) as T
  if (path.endsWith('/admin/users')) return page(adminUsers) as T
  const invitationMatch = path.match(/\/api\/v1\/invitations\/([^/]+)$/)
  if (invitationMatch) {
    const invitation = adminInvitations.find((item) => item.invite_url.endsWith(invitationMatch[1]))
    if (!invitation) throw new Error('邀请链接不存在或已失效。')
    const { invite_url: _inviteUrl, ...preview } = invitation
    return preview as Invitation as T
  }
  const projectMembersMatch = path.match(/\/admin\/projects\/([^/]+)\/members$/)
  if (projectMembersMatch) return page(adminProjectMembers[projectMembersMatch[1]] ?? []) as T
  const projectMilestonesMatch = path.match(/\/admin\/projects\/([^/]+)\/milestones$/)
  if (projectMilestonesMatch) return page(adminProjectMilestones[projectMilestonesMatch[1]] ?? []) as T
  if (path.endsWith('/admin/projects')) return page(adminProjects) as T
  if (path.endsWith('/admin/applications')) return page(applications) as T
  if (path.endsWith('/admin/server/status')) return adminServer as T
  const contentMatch = path.match(/\/organizations\/[^/]+\/content\/([^/]+)$/)
  if (contentMatch) {
    const detail = contentDetails[contentMatch[1]]
    if (!detail) throw new Error('公开内容不存在或尚未发布。')
    return detail as T
  }
  if (/\/organizations\/[^/]+$/.test(path)) return organization as T
  if (path.endsWith('/posts')) return page(posts) as T
  if (path.endsWith('/projects')) return page(projects) as T
  if (path.endsWith('/resources')) return page(resources) as T
  if (path.endsWith('/knowledge/articles')) return page(knowledge) as T
  if (path.endsWith('/knowledge/directories')) return page(knowledgeDirectories) as T
  if (path.endsWith('/server-status')) return serverStatus as T
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
  if (path.endsWith('/apply')) return { id: `application_${Date.now()}`, status: 'pending', submitted_at: new Date().toISOString() } as T
  if (path.includes('/admin/')) requireMockAdmin()
  if (path.endsWith('/admin/content')) {
    const payload = body as Pick<AdminContent, 'title' | 'type'>
    const content: AdminContent = { id: `content_${Date.now()}`, title: payload.title, type: payload.type, status: 'draft', author: mockUser?.display_name ?? 'BBKarasu', updated_at: new Date().toISOString() }
    adminContent = [content, ...adminContent]
    return content as T
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
    }
    adminInvitations = [invitation, ...adminInvitations]
    return invitation as T
  }
  const invitationAcceptMatch = path.match(/\/api\/v1\/invitations\/([^/]+)\/accept$/)
  if (invitationAcceptMatch) {
    requireMockAdmin()
    const invitation = adminInvitations.find((item) => item.invite_url.endsWith(invitationAcceptMatch[1]))
    if (!invitation) throw new Error('邀请链接不存在或已失效。')
    invitation.status = 'accepted'
    return { ...invitation, membership_id: `membership_${Date.now()}` } as T
  }
  if (path.endsWith('/admin/assets')) {
    return { id: `asset_${Date.now()}`, original_name: 'mock-upload.bin', mime_type: 'application/octet-stream', size_bytes: 0, download_url: '#' } as T
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
    content.updated_at = new Date().toISOString()
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
    application.status = decision[2] === 'approve' ? 'approved' : 'rejected'
    return application as T
  }
  if (path.endsWith('/admin/server/commands')) {
    const command = (body as { command: string }).command
    adminServer.last_command_at = new Date().toISOString()
    return { accepted: true, message: `命令“${command}”已被模拟环境记录。`, executed_at: adminServer.last_command_at } as T
  }
  throw new Error(`Mock endpoint not implemented: ${path}`)
}

export async function mockPatch<T>(path: string, body: unknown): Promise<T> {
  await wait()
  requireMockAdmin()
  const userMatch = path.match(/\/admin\/users\/([^/]+)$/)
  if (userMatch) {
    const user = adminUsers.find((item) => item.id === userMatch[1])
    if (!user) throw new Error('成员不存在。')
    Object.assign(user, body)
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
  const payload = body as Pick<AdminContent, 'title' | 'type' | 'category' | 'excerpt' | 'body'>
  Object.assign(item, payload, { updated_at: new Date().toISOString() })
  return item as T
}

export async function mockDelete<T>(path: string): Promise<T> {
  await wait()
  requireMockAdmin()
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
