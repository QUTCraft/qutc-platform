import { get, getPage, post } from '@/api/client'
import type { ApplicationPayload, KnowledgeArticle, KnowledgeDirectory, Organization, PortalRuntimeConfiguration, Project, PublicContentDetail, PublicPost, Resource, ServerStatus } from '@/api/types'

const defaultOrganizationSlug = import.meta.env.VITE_ORGANIZATION_SLUG ?? 'qutcraft'
const organizationStorageKey = 'qutc.portal_organization_slug'
const organizationSlugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

function normalizeOrganizationSlug(value: string | null | undefined) {
  const slug = value?.trim().toLowerCase() ?? ''
  return organizationSlugPattern.test(slug) ? slug : ''
}

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
