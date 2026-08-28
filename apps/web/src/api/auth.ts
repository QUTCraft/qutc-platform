import { get, patch, post } from '@/api/client'
import type { AuthUser, OrganizationMembership, TokenPair } from '@/api/types'

// authApi 封装会话与身份资料接口。令牌本体由后端通过 HttpOnly Cookie 维护，
// TokenPair 中的用户和到期信息仅用于更新前端会话状态。
export const authApi = {
  // invitation_token 可选；存在时注册后会直接加入其对应组织。
  register: (payload: { email: string; display_name: string; password: string; invitation_token?: string }) => post<TokenPair>('/api/v1/auth/register', payload),
  login: (payload: { email: string; password: string }) => post<TokenPair>('/api/v1/auth/login', payload),
	refresh: () => post<TokenPair>('/api/v1/auth/refresh', {}),
	logout: () => post<{ revoked: boolean }>('/api/v1/auth/logout', {}),
  getMe: () => get<AuthUser>('/api/v1/auth/me'),
  getOrganizations: () => get<OrganizationMembership[]>('/api/v1/auth/organizations'),
  // organizationId 是数据库主键；切换后后端会签发绑定新组织的会话。
  switchOrganization: (organizationId: string) => post<TokenPair>('/api/v1/auth/switch-organization', { organization_id: organizationId }),
  updateMe: (payload: { display_name: string; bio?: string; avatar_url?: string }) => patch<AuthUser>('/api/v1/auth/me', payload),
}
