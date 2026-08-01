import { reactive } from 'vue'
import { authApi } from '@/api/auth'
import { clearTokens, getAccessToken, saveTokens } from '@/auth/token-storage'
import type { AuthUser, TokenPair } from '@/api/types'

const userKey = 'qutc.session_user'
const storedUser = () => { try { return JSON.parse(window.localStorage.getItem(userKey) ?? 'null') as AuthUser | null } catch { return null } }

export const session = reactive<{ initialized: boolean; user: AuthUser | null }>({ initialized: false, user: storedUser() })

function apply(pair: TokenPair) { saveTokens(pair.access_token); window.localStorage.setItem(userKey, JSON.stringify(pair.user)); session.user = pair.user }
function clear() { clearTokens(); window.localStorage.removeItem(userKey); session.user = null }

export async function restoreSession() {
  if (session.initialized) return
  const accessToken = getAccessToken()
	if (!accessToken) {
		try { apply(await authApi.refresh()) }
		catch { clear() }
		session.initialized = true
		return
	}
	try { session.user = await authApi.getMe() }
  catch {
	try { apply(await authApi.refresh()) }
    catch { clear() }
  }
  session.initialized = true
}

export async function signIn(email: string, password: string) { apply(await authApi.login({ email, password })); session.initialized = true }
export async function signUp(payload: { email: string; display_name: string; password: string; invitation_token?: string }) { apply(await authApi.register(payload)); session.initialized = true }
export async function signOut() { try { await authApi.logout() } finally { clear(); session.initialized = true } }
export function expireSession() { clear(); session.initialized = true }
