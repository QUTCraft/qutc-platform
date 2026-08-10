import { reactive } from 'vue'
import { authApi } from '@/api/auth'
import { clearSessionExpiry, readSessionExpiry, writeSessionExpiry } from '@/auth/session-cookie'
import type { AuthUser, TokenPair } from '@/api/types'

const maxTimerDelay = 2_147_000_000
let expiryTimer: number | undefined

export const session = reactive<{ initialized: boolean; user: AuthUser | null; expiresAt: string | null }>({ initialized: false, user: null, expiresAt: null })

function scheduleExpiry(expiresAt?: string) {
  if (expiryTimer !== undefined) window.clearTimeout(expiryTimer)
  if (expiresAt) writeSessionExpiry(expiresAt)
  const expiry = readSessionExpiry()
  session.expiresAt = expiry ? new Date(expiry).toISOString() : null
  if (!expiry) return
  const remaining = expiry - Date.now()
  if (remaining <= 0) {
    void authApi.logout().catch(() => undefined)
    window.dispatchEvent(new Event('qutc:session-expired'))
    return
  }
  expiryTimer = window.setTimeout(() => {
    const currentExpiry = readSessionExpiry()
    if (!currentExpiry || currentExpiry <= Date.now()) {
      void authApi.logout().catch(() => undefined)
      window.dispatchEvent(new Event('qutc:session-expired'))
      return
    }
    scheduleExpiry()
  }, Math.min(remaining, maxTimerDelay))
}

function apply(pair: TokenPair) {
  session.user = pair.user
  scheduleExpiry(pair.session_expires_at)
}

function clear() {
  if (expiryTimer !== undefined) window.clearTimeout(expiryTimer)
  expiryTimer = undefined
  clearSessionExpiry()
  session.user = null
  session.expiresAt = null
}

export async function restoreSession() {
  if (session.initialized) return
  const storedExpiry = readSessionExpiry()
  if (storedExpiry && storedExpiry <= Date.now()) {
    try { await authApi.logout() } catch { /* Expired server cookies may already be gone. */ }
    clear()
    session.initialized = true
    return
  }
  try {
    session.user = await authApi.getMe()
    scheduleExpiry()
  } catch { clear() }
  session.initialized = true
}

export async function signIn(email: string, password: string) { apply(await authApi.login({ email, password })); session.initialized = true }
export async function signUp(payload: { email: string; display_name: string; password: string; invitation_token?: string }) { apply(await authApi.register(payload)); session.initialized = true }
export async function switchSessionOrganization(organizationId: string) { apply(await authApi.switchOrganization(organizationId)); session.initialized = true }
export async function signOut() { try { await authApi.logout() } finally { clear(); session.initialized = true } }
export function expireSession() { clear(); session.initialized = true }
