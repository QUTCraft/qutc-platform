import { get, post } from '@/api/client'
import type { AuthUser, TokenPair } from '@/api/types'

export const authApi = {
  register: (payload: { email: string; display_name: string; password: string }) => post<TokenPair>('/api/v1/auth/register', payload),
  login: (payload: { email: string; password: string }) => post<TokenPair>('/api/v1/auth/login', payload),
  refresh: (refresh_token: string) => post<TokenPair>('/api/v1/auth/refresh', { refresh_token }),
  logout: (refresh_token?: string) => post<{ revoked: boolean }>('/api/v1/auth/logout', { refresh_token }),
  getMe: () => get<AuthUser>('/api/v1/auth/me'),
}
