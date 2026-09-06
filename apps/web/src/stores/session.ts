import { reactive } from 'vue'
import { authApi } from '@/api/auth'
import { clearSessionExpiry, readSessionExpiry, writeSessionExpiry } from '@/auth/session-cookie'
import type { AuthUser, TokenPair } from '@/api/types'

// 浏览器 setTimeout 的最大安全延迟约为 24.8 天；更长会话需分段检查到期时间。
const maxTimerDelay = 2_147_000_000
let expiryTimer: number | undefined

// 全局响应式会话状态。令牌由 HttpOnly Cookie 持有，前端状态只保存用户资料和到期时间。
export const session = reactive<{ initialized: boolean; user: AuthUser | null; expiresAt: string | null }>({ initialized: false, user: null, expiresAt: null })

/**
 * 按 expiresAt（若提供）或 sessionStorage 中的到期时间安排本地退出。
 * 到期时先尽力通知后端注销，再派发全局失效事件；网络失败不阻止本地清理。
 */
function scheduleExpiry(expiresAt?: string) {
	// 每次登录、刷新或切换组织都重置计时器，确保旧计时器不能提前清理新会话。
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
  // 超过浏览器计时器上限时先重新调度；真正到期后再注销，避免长会话计时溢出。
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

/** 将登录、注册或切换组织接口返回的会话快照同步到全局响应式状态。 */
function apply(pair: TokenPair) {
	// 后端可能在登录、注册和切换组织后返回不同主体，统一由此处原子更新。
  session.user = pair.user
  scheduleExpiry(pair.session_expires_at)
}

/** 清除浏览器可见会话状态与本地到期记录，不直接修改服务端 Cookie。 */
function clear() {
	// 清理本地可见状态并取消定时器；服务端 Cookie 的撤销由 logout 接口负责。
  if (expiryTimer !== undefined) window.clearTimeout(expiryTimer)
  expiryTimer = undefined
  clearSessionExpiry()
  session.user = null
  session.expiresAt = null
}

/**
 * 在首次路由跳转前恢复会话。它通过 /me 重新从服务端确认 Cookie 有效性，
 * 而不信任仅保存在浏览器中的到期时间。
 */
export async function restoreSession() {
	// 路由守卫多次触发时只恢复一次，避免并发 /me 请求造成闪烁或竞态。
  if (session.initialized) return
  const storedExpiry = readSessionExpiry()
  if (storedExpiry && storedExpiry <= Date.now()) {
    try { await authApi.logout() } catch { /* Expired server cookies may already be gone. */ }
    clear()
    session.initialized = true
    return
  }
  try {
    // Cookie 自动随同请求发送；成功即证明服务端会话与当前组织成员关系仍有效。
    session.user = await authApi.getMe()
    scheduleExpiry()
  } catch { clear() }
  session.initialized = true
}

/** 使用邮箱和密码建立会话，并将服务端返回的用户与到期时间写入状态。 */
export async function signIn(email: string, password: string) { apply(await authApi.login({ email, password })); session.initialized = true }
/** 注册账户；invitation_token 存在时会同时完成受邀组织的加入。 */
export async function signUp(payload: { email: string; display_name: string; password: string; invitation_token?: string }) { apply(await authApi.register(payload)); session.initialized = true }
/** 切换当前会话所属组织；organizationId 是后端组织主键而非展示用 slug。 */
export async function switchSessionOrganization(organizationId: string) { apply(await authApi.switchOrganization(organizationId)); session.initialized = true }
/** 请求服务端撤销 Cookie；无论请求结果如何都清除本地状态。 */
export async function signOut() { try { await authApi.logout() } finally { clear(); session.initialized = true } }
/** 在 API 客户端检测到不可恢复的 401 时执行纯本地会话失效处理。 */
export function expireSession() { clear(); session.initialized = true }
