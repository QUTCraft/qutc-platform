import { del, get, getPage, patch, post } from '@/api/client'
import type { AdminApplication, AdminContent, AdminDashboard, AdminProject, AdminProjectMember, AdminProjectMilestone, AdminServerStatus, AdminUser, ServerCommandResult } from '@/api/types'

const adminBase = '/api/v1/admin'

export const adminApi = {
  getDashboard: () => get<AdminDashboard>(`${adminBase}/dashboard`),
  getContent: () => getPage<AdminContent>(`${adminBase}/content`),
  createContent: (payload: Pick<AdminContent, 'title' | 'type'>) => post<AdminContent>(`${adminBase}/content`, payload),
  updateContent: (id: string, payload: Pick<AdminContent, 'title' | 'type' | 'category' | 'excerpt' | 'body'>) => patch<AdminContent>(`${adminBase}/content/${id}`, payload),
  publishContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/publish`),
  archiveContent: (id: string) => post<AdminContent>(`${adminBase}/content/${id}/archive`),
  getUsers: () => getPage<AdminUser>(`${adminBase}/users`),
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
