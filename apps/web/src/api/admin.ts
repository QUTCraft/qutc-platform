import { get, getPage, post } from '@/api/client'
import type { AdminApplication, AdminContent, AdminDashboard, AdminServerStatus, AdminUser, ServerCommandResult } from '@/api/types'

const adminBase = '/api/v1/admin'

export const adminApi = {
  getDashboard: () => get<AdminDashboard>(`${adminBase}/dashboard`),
  getContent: () => getPage<AdminContent>(`${adminBase}/content`),
  createContent: (payload: Pick<AdminContent, 'title' | 'type'>) => post<AdminContent>(`${adminBase}/content`, payload),
  getUsers: () => getPage<AdminUser>(`${adminBase}/users`),
  getApplications: () => getPage<AdminApplication>(`${adminBase}/applications`),
  approveApplication: (id: string) => post<AdminApplication>(`${adminBase}/applications/${id}/approve`),
  rejectApplication: (id: string) => post<AdminApplication>(`${adminBase}/applications/${id}/reject`),
  getServerStatus: () => get<AdminServerStatus>(`${adminBase}/server/status`),
  runServerCommand: (command: string) => post<ServerCommandResult>(`${adminBase}/server/commands`, { command }),
}
