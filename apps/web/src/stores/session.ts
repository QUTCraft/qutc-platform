import { reactive } from 'vue'
import { authApi } from '@/api/auth'
import { clearTokens, getAccessToken, getRefreshToken, saveTokens } from '@/auth/token-storage'
import type { AuthUser, TokenPair } from '@/api/types'

const userKey = 'qutc.session_user'
const storedUser = () => { try { return JSON.parse(window.localStorage.getItem(userKey) ?? 'null') as AuthUser | null } catch { return null } }

export const session = reactive<{ initialized: boolean; user: AuthUser | null }>({ initialized: false, user: storedUser() })

function apply(pair: TokenPair) { saveTokens(pair.access_token, pair.refresh_token); window.localStorage.setItem(userKey, JSON.stringify(pair.user)); session.user = pair.user }
function clear() { clearTokens(); window.localStorage.removeItem(userKey); session.user = null }

export async function restoreSession() {
  if (session.initialized) return
  const accessToken = getAccessToken()
  const refreshToken = getRefreshToken()
  if (!accessToken) { clear(); session.initialized = true; return }
  try { session.user = await authApi.getMe() }
  catch {
    try { if (!refreshToken) throw new Error('no refresh token'); apply(await authApi.refresh(refreshToken)) }
    catch { clear() }
  }
  session.initialized = true
}

export async function signIn(email: string, password: string) { apply(await authApi.login({ email, password })); session.initialized = true }
export async function signOut() { try { await authApi.logout(getRefreshToken() ?? undefined) } finally { clear(); session.initialized = true } }
