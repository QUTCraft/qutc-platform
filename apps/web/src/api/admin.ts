import { del, get, getPage, patch, post, upload } from '@/api/client'
import type { AdminApplication, AdminContent, AdminDashboard, AdminInvitation, AdminKnowledgeDirectory, AdminProject, AdminProjectMember, AdminProjectMilestone, AdminServerStatus, AdminUser, InvitationRole, MediaAsset, ServerCommandResult } from '@/api/types'

const adminBase = '/api/v1/admin'

export const adminApi = {
  getDashboard: () => get<AdminDashboard>(`${adminBase}/dashboard`),
  getContent: () => getPage<AdminContent>(`${adminBase}/content`),
  createContent: (payload: Pick<AdminContent, 'title' | 'type'>) => post<AdminContent>(`${adminBase}/content`, payload),
  updateContent: (id: string, payload: Pick<AdminContent, 'title' | 'type' | 'category' | 'excerpt' | 'body'>) => patch<AdminContent>(`${adminBase}/content/${id}`, payload),
  publishContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/publish`),
  archiveContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/archive`),
  uploadAsset: (file: File, contentId?: string) => {
    const formData = new FormData()
    formData.append('file', file)
    if (contentId) formData.append('content_id', contentId)
    return upload<MediaAsset>(`${adminBase}/assets`, formData)
  },
  getKnowledgeDirectories: () => getPage<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories`),
  createKnowledgeDirectory: (payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => post<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories`, payload),
  updateKnowledgeDirectory: (id: string, payload: Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>) => patch<AdminKnowledgeDirectory>(`${adminBase}/knowledge/directories/${id}`, payload),
  getUsers: () => getPage<AdminUser>(`${adminBase}/users`),
  createInvitation: (payload: { email: string; role: InvitationRole; expires_in_hours?: number }) => post<AdminInvitation>(`${adminBase}/invitations`, payload),
  updateUser: (id: string, payload: Pick<AdminUser, 'state' | 'role'>) => patch<AdminUser>(`${adminBase}/users/${id}`, payload),
  getProjects: () => getPage<AdminProject>(`${adminBase}/projects`),
  createProject: (payload: Pick<AdminProject, 'title' | 'summary' | 'status' | 'tags' | 'is_public'>) => post<AdminProject>(`${adminBase}/projects`, payload),
  updateProject: (id: string, payload: Pick<AdminProject, 'title' | 'summary' | 'status' | 'tags' | 'is_public'>) => patch<AdminProject>(`${adminBase}/projects/${id}`, payload),
  getProjectMembers: (id: string) => getPage<AdminProjectMember>(`${adminBase}/projects/${id}/members`),
  addProjectMember: (id: string, payload: { user_id: string; role: AdminProjectMember['role'] }) => post<AdminProjectMember>(`${adminBase}/projects/${id}/members`, payload),
  updateProjectMember: (projectId: string, userId: string, payload: { role: AdminProjectMember['role'] }) => patch<AdminProjectMember>(`${adminBase}/projects/${projectId}/members/${userId}`, payload),
  removeProjectMember: (projectId: string, userId: string) => del<{ removed: boolean }>(`${adminBase}/projects/${projectId}/members/${userId}`),
  getProjectMilestones: (id: string) => getPage<AdminProjectMilestone>(`${adminBase}/projects/${id}/milestones`),
  createProjectMilestone: (id: string, payload: { title: string; status: AdminProjectMilestone['status']; due_at?: string }) => post<AdminProjectMilestone>(`${adminBase}/projects/${id}/milestones`, payload),
  updateProjectMilestone: (projectId: string, milestoneId: string, payload: { title: string; status: AdminProjectMilestone['status']; due_at?: string }) => patch<AdminProjectMilestone>(`${adminBase}/projects/${projectId}/milestones/${milestoneId}`, payload),
  removeProjectMilestone: (projectId: string, milestoneId: string) => del<{ removed: boolean }>(`${adminBase}/projects/${projectId}/milestones/${milestoneId}`),
  getApplications: () => getPage<AdminApplication>(`${adminBase}/applications`),
  approveApplication: (id: string) => post<AdminApplication>(`${adminBase}/applications/${id}/approve`),
  rejectApplication: (id: string) => post<AdminApplication>(`${adminBase}/applications/${id}/reject`),
  getServerStatus: () => get<AdminServerStatus>(`${adminBase}/server/status`),
  runServerCommand: (command: string) => post<ServerCommandResult>(`${adminBase}/server/commands`, { command }),
}
