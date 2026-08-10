import { del, get, getPage, patch, post, put, upload } from '@/api/client'
import type { ActivityPlan, ActivityPlanApprovalResult, ActivityPlanEvaluation, ActivityPlanEvaluationSummary, ActivityPlanSummary, AdminApplication, AdminApplicationFilters, AdminContent, AdminDashboard, AdminInvitation, AdminInvitationSummary, AdminKnowledgeDirectory, AdminMembershipWriteState, AdminProject, AdminProjectMember, AdminProjectMilestone, AdminServerStatus, AdminUser, AIAgentCatalog, AIAgentRun, AIConfiguration, AIConfigurationUpdate, AIKnowledgeResult, AISourceReference, AuditEvent, AuditEventFilters, BatchInvitationResponse, ContentRevision, EmailAdapterStatus, IntegrationSettings, IntegrationSettingsUpdate, IntegrationTestResult, Invitation, InvitationRole, InvitationStatus, InvitationTemplate, MediaAsset, NotificationOutbox, Organization, PortalConfiguration, PortalManifest, PublishAssetResourceInput, ServerCommandResult } from '@/api/types'

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
	getOrganization: () => get<Organization>(`${adminBase}/organization`),
	updateOrganization: (payload: Pick<Organization, 'name' | 'short_name' | 'tagline' | 'introduction' | 'contact_email' | 'social_links' | 'is_public'>) => patch<Organization>(`${adminBase}/organization`, payload),
  getContent: (params: PageQuery = {}) => getPage<AdminContent>(withQuery(`${adminBase}/content`, params)),
  getContentById: (id: string) => get<AdminContent>(`${adminBase}/content/${id}`),
  getContentRevisions: (id: string, params: PageQuery = {}) => getPage<ContentRevision>(withQuery(`${adminBase}/content/${id}/revisions`, params)),
  getContentRevision: (contentId: string, revisionId: string) => get<ContentRevision>(`${adminBase}/content/${contentId}/revisions/${revisionId}`),
  restoreContentRevision: (contentId: string, revisionId: string) => post<AdminContent>(`${adminBase}/content/${contentId}/revisions/${revisionId}/restore`),
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
  getAssets: (params: PageQuery & { query?: string } = {}) => getPage<MediaAsset>(withQuery(`${adminBase}/assets`, params)),
  publishAssetAsResource: (id: string, payload: PublishAssetResourceInput) => post<AdminContent>(`${adminBase}/assets/${id}/publish`, payload),
  deleteAsset: (id: string) => del<{ removed: boolean; id: string; detached_content_id: string | null }>(`${adminBase}/assets/${id}`),
  getKnowledgeDirectories: (params: PageQuery = {}) => getPage<AdminKnowledgeDirectory>(withQuery(`${adminBase}/knowledge/directories`, params)),
  createKnowledgeDirectory: (payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => post<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories`, payload),
  updateKnowledgeDirectory: (id: string, payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => patch<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories/${id}`, payload),
  getUsers: (params: PageQuery = {}) => getPage<AdminUser>(withQuery(`${adminBase}/users`, params)),
  getInvitations: (params: PageQuery & { status?: InvitationStatus | '' } = {}) => getPage<AdminInvitationSummary>(withQuery(`${adminBase}/invitations`, params)),
  createInvitation: (payload: { email: string; role: InvitationRole; expires_in_hours?: number }) => post<AdminInvitation>(`${adminBase}/invitations`, payload),
  createBatchInvitations: (payload: { invitations: Array<{ email: string; role: InvitationRole; expires_in_hours?: number }> }) => post<BatchInvitationResponse>(`${adminBase}/invitation-batches`, payload),
  revokeInvitation: (id: string) => del<Invitation>(`${adminBase}/invitations/${id}`),
  retryInvitationEmail: (id: string) => post<AdminInvitation>(`${adminBase}/invitations/${id}/email/retry`),
  getEmailAdapterStatus: () => get<EmailAdapterStatus>(`${adminBase}/notifications/email/status`),
  getIntegrationSettings: () => get<IntegrationSettings>(`${adminBase}/integrations`),
  updateIntegrationSettings: (payload: IntegrationSettingsUpdate) => patch<IntegrationSettings>(`${adminBase}/integrations`, payload),
  testIntegration: (section: IntegrationTestResult['section']) => post<IntegrationTestResult>(`${adminBase}/integrations/test`, { section }),
  getInvitationTemplate: () => get<InvitationTemplate>(`${adminBase}/notifications/invitation-template`),
  updateInvitationTemplate: (payload: Pick<InvitationTemplate, 'subject_template' | 'body_template'>) => patch<InvitationTemplate>(`${adminBase}/notifications/invitation-template`, payload),
  getNotificationOutbox: (params: PageQuery & { status?: NotificationOutbox['status'] | '' } = {}) => getPage<NotificationOutbox>(withQuery(`${adminBase}/notifications/outbox`, params)),
  retryNotification: (id: string) => post<NotificationOutbox>(`${adminBase}/notifications/outbox/${id}/retry`),
  getAssetDownloadStats: (id: string) => get<{ id: string; content_id?: string; download_count: number; last_downloaded_at?: string | null }>(`${adminBase}/assets/${id}/stats`),
  updateUser: (id: string, payload: { state: AdminMembershipWriteState; role: AdminUser['role'] }) => patch<AdminUser>(`${adminBase}/users/${id}`, payload),
  getAuditEvents: (params: AuditEventFilters = {}) => getPage<AuditEvent>(withQuery(`${adminBase}/audit`, params)),
  getAIConfiguration: () => get<AIConfiguration>(`${adminBase}/ai/config`),
  updateAIConfiguration: (payload: AIConfigurationUpdate) => patch<AIConfiguration>(`${adminBase}/ai/config`, payload),
  getAIAgents: () => get<AIAgentCatalog>(`${adminBase}/ai/agents`),
  searchAIKnowledge: (payload: { query: string; limit?: number }) => post<AIKnowledgeResult[]>(`${adminBase}/ai/knowledge/search`, payload),
  createAIRun: (payload: { agent_key: 'content-copilot' | 'activity-planner'; task: string; context_refs: AISourceReference[]; output_mode: 'proposal' }) => post<AIAgentRun>(`${adminBase}/ai/runs`, payload),
  getAIRun: (id: string) => get<AIAgentRun>(`${adminBase}/ai/runs/${id}`),
  cancelAIRun: (id: string) => post<AIAgentRun>(`${adminBase}/ai/runs/${id}/cancel`),
  createActivityPlan: (payload: {
    title: string
    objective: string
    audience: string
    venue: string
    starts_at?: string
    ends_at?: string
    expected_participants: number
    budget: string
    constraints: string
    context_refs: AISourceReference[]
  }) => post<ActivityPlan>(`${adminBase}/ai/activity-plans`, payload),
  getActivityPlans: (params: PageQuery = {}) => getPage<ActivityPlanSummary>(withQuery(`${adminBase}/ai/activity-plans`, params)),
  getActivityPlanEvaluationSummary: () => get<ActivityPlanEvaluationSummary>(`${adminBase}/ai/activity-plans/evaluation-summary`),
  getActivityPlan: (id: string) => get<ActivityPlan>(`${adminBase}/ai/activity-plans/${id}`),
  getActivityPlanEvaluation: (id: string) => get<ActivityPlanEvaluation | null>(`${adminBase}/ai/activity-plans/${id}/evaluation`),
  saveActivityPlanEvaluation: (id: string, payload: Pick<ActivityPlanEvaluation, 'accuracy' | 'feasibility' | 'campus_fit' | 'clarity' | 'adoptability' | 'notes'>) => put<ActivityPlanEvaluation>(`${adminBase}/ai/activity-plans/${id}/evaluation`, payload),
  approveActivityPlan: (id: string, actions: string[]) => post<ActivityPlanApprovalResult>(`${adminBase}/ai/activity-plans/${id}/approve`, { actions }),
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
