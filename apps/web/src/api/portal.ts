import { get, getPage, post } from '@/api/client'
import type { ApplicationPayload, KnowledgeArticle, KnowledgeDirectory, Organization, PortalRuntimeConfiguration, Project, PublicContentDetail, PublicPost, Resource, ServerStatus } from '@/api/types'

export const organizationSlug = import.meta.env.VITE_ORGANIZATION_SLUG ?? 'qutcraft'
export const portalBase = `/api/v1/portal/organizations/${organizationSlug}`

export interface PortalPageQuery {
  page?: number
  page_size?: number
  category?: string
  status?: Project['status']
  kind?: Resource['kind']
  q?: string
}

function withQuery(path: string, params: PortalPageQuery = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString()
  return `${path}${suffix ? `?${suffix}` : ''}`
}

export const portalApi = {
  getOrganization: () => get<Organization>(portalBase),
  getRuntimeConfiguration: (signal?: AbortSignal) => get<PortalRuntimeConfiguration>(`${portalBase}/configuration`, signal),
  getContentDetail: (id: string) => get<PublicContentDetail>(`${portalBase}/content/${id}`),
  getPosts: (params: PortalPageQuery = {}) => getPage<PublicPost>(withQuery(`${portalBase}/posts`, params)),
  getProjects: (params: PortalPageQuery = {}) => getPage<Project>(withQuery(`${portalBase}/projects`, params)),
  getResources: (params: PortalPageQuery = {}) => getPage<Resource>(withQuery(`${portalBase}/resources`, params)),
  getKnowledgeArticles: (params: PortalPageQuery = {}) => getPage<KnowledgeArticle>(withQuery(`${portalBase}/knowledge/articles`, params)),
  getKnowledgeDirectories: (params: PortalPageQuery = {}) => getPage<KnowledgeDirectory>(withQuery(`${portalBase}/knowledge/directories`, params)),
  getServerStatus: () => get<ServerStatus>(`${portalBase}/server-status`),
  submitApplication: (payload: ApplicationPayload) => post<{ id: string; status: string }>(`${portalBase}/apply`, payload),
}
