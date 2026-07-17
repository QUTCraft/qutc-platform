import type {
  KnowledgeArticle,
  Organization,
  Page,
  Project,
  PublicPost,
  Resource,
  ServerStatus,
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
  { id: 'resource_overview', title: 'QUTCraft CMS 产品说明', description: '项目目标、公开门户范围与 MVP 路线。', kind: 'document', size_bytes: 2_600_000, updated_at: '2026-07-17T01:00:00Z', download_url: '#' },
  { id: 'resource_event-kit', title: '社团活动策划模板', description: '用于活动立项、分工和复盘的基础模板。', kind: 'template', size_bytes: 540_000, updated_at: '2026-07-15T01:00:00Z', download_url: '#' },
  { id: 'resource_portal-api', title: 'Portal API 快速开始', description: '为自定义门户开发者准备的接口与 Manifest 示例。', kind: 'package', size_bytes: 1_400_000, updated_at: '2026-07-12T01:00:00Z', download_url: '#' },
]

const knowledge: KnowledgeArticle[] = [
  { id: 'knowledge_handoff', title: '如何让社团项目可交接', summary: '从目标、角色、决策记录到发布节奏，建立不依赖个人记忆的项目协作方式。', category: '项目协作', updated_at: '2026-07-16T02:00:00Z', reading_minutes: 8 },
  { id: 'knowledge_portal', title: 'Portal API 的公开能力边界', summary: '哪些内容能被主题门户消费，哪些数据必须只留在后台。', category: '开发规范', updated_at: '2026-07-14T02:00:00Z', reading_minutes: 6 },
  { id: 'knowledge_server', title: '服务器公开状态的设计原则', summary: '门户展示状态、申请入口；后台处理审核与命令执行。', category: '服务器', updated_at: '2026-07-11T02:00:00Z', reading_minutes: 5 },
]

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

const page = <T>(items: T[]): Page<T> => ({ items, page: 1, page_size: 20, total: items.length })
const wait = () => new Promise((resolve) => window.setTimeout(resolve, 160))

export async function mockGet<T>(path: string): Promise<T> {
  await wait()
  if (/\/organizations\/[^/]+$/.test(path)) return organization as T
  if (path.endsWith('/posts')) return page(posts) as T
  if (path.endsWith('/projects')) return page(projects) as T
  if (path.endsWith('/resources')) return page(resources) as T
  if (path.endsWith('/knowledge/articles')) return page(knowledge) as T
  if (path.endsWith('/server-status')) return serverStatus as T
  throw new Error(`Mock endpoint not implemented: ${path}`)
}
