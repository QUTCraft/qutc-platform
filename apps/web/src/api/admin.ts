import { del, get, getPage, patch, post, upload } from '@/api/client'
import type { AdminApplication, AdminApplicationFilters, AdminContent, AdminDashboard, AdminInvitation, AdminKnowledgeDirectory, AdminMembershipWriteState, AdminProject, AdminProjectMember, AdminProjectMilestone, AdminServerStatus, AdminUser, AIAgentCatalog, AIAgentRun, AIConfiguration, AIKnowledgeResult, AISourceReference, AuditEvent, AuditEventFilters, EmailAdapterStatus, InvitationRole, MediaAsset, PortalConfiguration, PortalManifest, ServerCommandResult } from '@/api/types'

const adminBase = '/api/v1/admin'

export interface PageQuery {
  page?: number
  page_size?: number
}

function withQuery(path: string, params: object = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString()
  return `${path}${suffix ? `?${suffix}` : ''}`
}

export const adminApi = {
  getDashboard: () => get<AdminDashboard>(`${adminBase}/dashboard`),
  getContent: (params: PageQuery = {}) => getPage<AdminContent>(withQuery(`${adminBase}/content`, params)),
  getContentById: (id: string) => get<AdminContent>(`${adminBase}/content/${id}`),
  createContent: (payload: Pick<AdminContent, 'title' | 'type' | 'category' | 'knowledge_directory_id' | 'excerpt' | 'body'>) => post<AdminContent>(`${adminBase}/content`, payload),
  updateContent: (id: string, payload: Pick<AdminContent, 'title' | 'type' | 'category' | 'knowledge_directory_id' | 'excerpt' | 'body'>) => patch<AdminContent>(`${adminBase}/content/${id}`, payload),
  publishContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/publish`),
  archiveContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/archive`),
  uploadAsset: (file: File, contentId?: string) => {
    const formData = new FormData()
    formData.append('file', file)
    if (contentId) formData.append('content_id', contentId)
    return upload<MediaAsset>(`${adminBase}/assets`, formData)
  },
  getKnowledgeDirectories: (params: PageQuery = {}) => getPage<AdminKnowledgeDirectory>(withQuery(`${adminBase}/knowledge/directories`, params)),
  createKnowledgeDirectory: (payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => post<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories`, payload),
  updateKnowledgeDirectory: (id: string, payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => patch<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories/${id}`, payload),
  getUsers: (params: PageQuery = {}) => getPage<AdminUser>(withQuery(`${adminBase}/users`, params)),
  createInvitation: (payload: { email: string; role: InvitationRole; expires_in_hours?: number }) => post<AdminInvitation>(`${adminBase}/invitations`, payload),
  retryInvitationEmail: (id: string) => post<AdminInvitation>(`${adminBase}/invitations/${id}/email/retry`),
  getEmailAdapterStatus: () => get<EmailAdapterStatus>(`${adminBase}/notifications/email/status`),
  updateUser: (id: string, payload: { state: AdminMembershipWriteState; role: AdminUser['role'] }) => patch<AdminUser>(`${adminBase}/users/${id}`, payload),
  getAuditEvents: (params: AuditEventFilters = {}) => getPage<AuditEvent>(withQuery(`${adminBase}/audit`, params)),
  getAIConfiguration: () => get<AIConfiguration>(`${adminBase}/ai/config`),
  updateAIConfiguration: (payload: Pick<AIConfiguration, 'enabled' | 'run_limit_per_hour' | 'request_timeout_seconds' | 'max_sources' | 'max_context_characters'>) => patch<AIConfiguration>(`${adminBase}/ai/config`, payload),
  getAIAgents: () => get<AIAgentCatalog>(`${adminBase}/ai/agents`),
  searchAIKnowledge: (payload: { query: string; limit?: number }) => post<AIKnowledgeResult[]>(`${adminBase}/ai/knowledge/search`, payload),
  createAIRun: (payload: { agent_key: 'content-copilot'; task: string; context_refs: AISourceReference[]; output_mode: 'proposal' }) => post<AIAgentRun>(`${adminBase}/ai/runs`, payload),
  getAIRun: (id: string) => get<AIAgentRun>(`${adminBase}/ai/runs/${id}`),
  cancelAIRun: (id: string) => post<AIAgentRun>(`${adminBase}/ai/runs/${id}/cancel`),
  getProjects: (params: PageQuery = {}) => getPage<AdminProject>(withQuery(`${adminBase}/projects`, params)),
  createProject: (payload: Pick<AdminProject, 'title' | 'summary' | 'status' | 'tags' | 'is_public'>) => post<AdminProject>(`${adminBase}/projects`, payload),
  updateProject: (id: string, payload: Pick<AdminProject, 'title' | 'summary' | 'status' | 'tags' | 'is_public'>) => patch<AdminProject>(`${adminBase}/projects/${id}`, payload),
  getProjectMembers: (id: string, params: PageQuery = {}) => getPage<AdminProjectMember>(withQuery(`${adminBase}/projects/${id}/members`, params)),
  addProjectMember: (id: string, payload: { user_id: string; role: AdminProjectMember['role'] }) => post<AdminProjectMember>(`${adminBase}/projects/${id}/members`, payload),
  updateProjectMember: (projectId: string, userId: string, payload: { role: AdminProjectMember['role'] }) => patch<AdminProjectMember>(`${adminBase}/projects/${projectId}/members/${userId}`, payload),
  removeProjectMember: (projectId: string, userId: string) => del<{ removed: boolean }>(`${adminBase}/projects/${projectId}/members/${userId}`),
  getProjectMilestones: (id: string, params: PageQuery = {}) => getPage<AdminProjectMilestone>(withQuery(`${adminBase}/projects/${id}/milestones`, params)),
  createProjectMilestone: (id: string, payload: { title: string; status: AdminProjectMilestone['status']; due_at?: string }) => post<AdminProjectMilestone>(`${adminBase}/projects/${id}/milestones`, payload),
  updateProjectMilestone: (projectId: string, milestoneId: string, payload: { title: string; status: AdminProjectMilestone['status']; due_at?: string }) => patch<AdminProjectMilestone>(`${adminBase}/projects/${projectId}/milestones/${milestoneId}`, payload),
  removeProjectMilestone: (projectId: string, milestoneId: string) => del<{ removed: boolean }>(`${adminBase}/projects/${projectId}/milestones/${milestoneId}`),
  getApplications: (filters: AdminApplicationFilters = {}) => {
    const query = new URLSearchParams()
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') query.set(key, String(value))
    })
    const suffix = query.toString()
    return getPage<AdminApplication>(`${adminBase}/applications${suffix ? `?${suffix}` : ''}`)
  },
  approveApplication: (id: string, reason = '') => post<AdminApplication>(`${adminBase}/applications/${id}/approve`, { reason }),
  rejectApplication: (id: string, reason: string) => post<AdminApplication>(`${adminBase}/applications/${id}/reject`, { reason }),
  retryApplicationServerSync: (id: string) => post<NonNullable<AdminApplication['server_sync']>>(`${adminBase}/applications/${id}/server-sync/retry`),
  getServerStatus: () => get<AdminServerStatus>(`${adminBase}/server/status`),
  runServerCommand: (command: string) => post<ServerCommandResult>(`${adminBase}/server/commands`, { command }),
  getPortalConfiguration: () => get<PortalConfiguration>(`${adminBase}/portal/config`),
  savePortalDraft: (manifest: PortalManifest) => patch<PortalConfiguration>(`${adminBase}/portal/config`, { manifest }),
  enablePortalConfiguration: () => post<PortalConfiguration>(`${adminBase}/portal/config/enable`),
  restoreDefaultPortal: () => post<PortalConfiguration>(`${adminBase}/portal/config/restore-default`),
}
