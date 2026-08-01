import { get, patch, post } from '@/api/client'
import type { AuthUser, TokenPair } from '@/api/types'

export const authApi = {
  register: (payload: { email: string; display_name: string; password: string; invitation_token?: string }) => post<TokenPair>('/api/v1/auth/register', payload),
  login: (payload: { email: string; password: string }) => post<TokenPair>('/api/v1/auth/login', payload),
	refresh: () => post<TokenPair>('/api/v1/auth/refresh', {}),
	logout: () => post<{ revoked: boolean }>('/api/v1/auth/logout', {}),
  getMe: () => get<AuthUser>('/api/v1/auth/me'),
  updateMe: (payload: { display_name: string; bio?: string; avatar_url?: string }) => patch<AuthUser>('/api/v1/auth/me', payload),
}
