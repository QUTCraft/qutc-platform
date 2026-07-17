import { get, getPage } from '@/api/client'
import type { KnowledgeArticle, Organization, Project, PublicPost, Resource, ServerStatus } from '@/api/types'

const organizationSlug = import.meta.env.VITE_ORGANIZATION_SLUG ?? 'qutcraft'
const portalBase = `/api/v1/portal/organizations/${organizationSlug}`

export const portalApi = {
  getOrganization: () => get<Organization>(portalBase),
  getPosts: () => getPage<PublicPost>(`${portalBase}/posts`),
  getProjects: () => getPage<Project>(`${portalBase}/projects`),
  getResources: () => getPage<Resource>(`${portalBase}/resources`),
  getKnowledgeArticles: () => getPage<KnowledgeArticle>(`${portalBase}/knowledge/articles`),
  getServerStatus: () => get<ServerStatus>(`${portalBase}/server-status`),
}
