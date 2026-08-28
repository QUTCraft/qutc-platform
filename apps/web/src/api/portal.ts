import { get, getPage, post } from '@/api/client'
import type { ApplicationPayload, KnowledgeArticle, KnowledgeDirectory, Organization, PortalRuntimeConfiguration, Project, PublicContentDetail, PublicPost, Resource } from '@/api/types'

// 组织标识的优先级为 URL 参数、当前标签页缓存、构建时默认值；仅允许规范 slug，
// 避免用户输入进入 API 路径后造成跨组织误请求。
const defaultOrganizationSlug = import.meta.env.VITE_ORGANIZATION_SLUG ?? 'qutcraft'
const organizationStorageKey = 'qutc.portal_organization_slug'
const organizationSlugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

// normalizeOrganizationSlug 归一化并校验候选 slug；非法值返回空串，交给调用方回退。
function normalizeOrganizationSlug(value: string | null | undefined) {
  const slug = value?.trim().toLowerCase() ?? ''
  return organizationSlugPattern.test(slug) ? slug : ''
}

// resolveOrganizationSlug 决定本页请求所属组织。sessionStorage 使 URL 选择在同一标签页后续导航中保持。
function resolveOrganizationSlug() {
  const fallback = normalizeOrganizationSlug(defaultOrganizationSlug) || 'qutcraft'
  if (typeof window === 'undefined') return fallback

  try {
    let requestedValue = ''
    new URLSearchParams(window.location.search).forEach((value, key) => {
      if (key === 'organization') requestedValue = value
    })
    const requested = normalizeOrganizationSlug(requestedValue)
    if (requested) {
      window.sessionStorage.setItem(organizationStorageKey, requested)
      return requested
    }
    return normalizeOrganizationSlug(window.sessionStorage.getItem(organizationStorageKey)) || fallback
  } catch {
    return fallback
  }
}

// organizationSlug 与 portalBase 在模块加载时固定，确保一次页面会话内所有门户请求属于同一组织。
export const organizationSlug = resolveOrganizationSlug()
export const portalBase = `/api/v1/portal/organizations/${organizationSlug}`

export interface PortalPageQuery {
  page?: number
  page_size?: number
  category?: string
  status?: Project['status']
  kind?: Resource['kind']
  q?: string
}

// withQuery 将可选门户筛选条件编码为 URL 查询串，并省略空值。
function withQuery(path: string, params: PortalPageQuery = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString()
  return `${path}${suffix ? `?${suffix}` : ''}`
}

// portalApi 描述匿名可读门户及公开申请接口；portalBase 已包含当前组织 slug。
export const portalApi = {
  getOrganization: () => get<Organization>(portalBase),
  // signal 允许门户启动逻辑超时取消配置读取，并立刻回退内置门户。
  getRuntimeConfiguration: (signal?: AbortSignal) => get<PortalRuntimeConfiguration>(`${portalBase}/configuration`, signal),
  getContentDetail: (id: string) => get<PublicContentDetail>(`${portalBase}/content/${id}`),
  getPosts: (params: PortalPageQuery = {}) => getPage<PublicPost>(withQuery(`${portalBase}/posts`, params)),
  getProjects: (params: PortalPageQuery = {}) => getPage<Project>(withQuery(`${portalBase}/projects`, params)),
  getResources: (params: PortalPageQuery = {}) => getPage<Resource>(withQuery(`${portalBase}/resources`, params)),
  getKnowledgeArticles: (params: PortalPageQuery = {}) => getPage<KnowledgeArticle>(withQuery(`${portalBase}/knowledge/articles`, params)),
  getKnowledgeDirectories: (params: PortalPageQuery = {}) => getPage<KnowledgeDirectory>(withQuery(`${portalBase}/knowledge/directories`, params)),
  submitApplication: (payload: ApplicationPayload) => post<{ id: string; status: string }>(`${portalBase}/apply`, payload),
}
