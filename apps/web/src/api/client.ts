import { writeSessionExpiry } from '@/auth/session-cookie'
import type { ApiEnvelope, Page } from '@/api/types'

// 生产包绝不能静默回退到模拟数据；模拟模式仅在 Vite 开发服务器中显式开启，
// 否则部署故障会被伪造的数据掩盖。
const requestedApiMode = import.meta.env.VITE_API_MODE?.trim().toLowerCase()
export const isMockApiMode = import.meta.env.DEV && requestedApiMode === 'mock'
const apiMode = isMockApiMode ? 'mock' : 'remote'
type MockApi = typeof import('@/api/mock')
let mockApiPromise: Promise<MockApi> | undefined
const loadMockApi = () => (mockApiPromise ??= import('@/api/mock'))
// 生产环境始终经由 Web 容器的同源 /api 反向代理访问后端。VITE_API_BASE_URL
// 仅供本地 Vite 开发服务器使用，避免把 localhost 写进构建产物后让访客请求自己的电脑。
const configuredDevelopmentApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim()
const apiBaseUrl = (
  import.meta.env.DEV && configuredDevelopmentApiBaseUrl
    ? configuredDevelopmentApiBaseUrl
    : window.location.origin
).replace(/\/$/, '')

/** 将接口相对路径解析为可请求 URL；绝对媒体 URL 和模拟模式路径原样返回。 */
export function resolveApiUrl(path: string): string {
	// 已是绝对地址的媒体链接不应再附加 API 域名；模拟模式同样保留其原始路径约定。
  if (/^https?:\/\//i.test(path)) return path
  if (apiMode === 'mock') return path
  return `${apiBaseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

export class ApiClientError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string) {
    super(message)
    this.name = 'ApiClientError'
  }
}

/** 根据请求体类型生成公共请求头；json 为 true 时才声明 JSON 内容类型。 */
function headers(json = false) {
	// 上传 FormData 时不能手动设置 Content-Type，否则浏览器不会补充 multipart boundary。
  return {
    Accept: 'application/json',
    ...(json ? { 'Content-Type': 'application/json' } : {}),
  }
}

// 多个并发请求同时遇到 401 时共享同一次刷新，避免刷新令牌轮换后互相使对方失效。
let refreshInFlight: Promise<boolean> | null = null

/** 判断 path 是否是认证端点，认证端点本身不能进入刷新令牌重试流程。 */
function isAuthPath(path: string) {
  return path === '/api/v1/auth/login' || path === '/api/v1/auth/register' || path === '/api/v1/auth/refresh' || path === '/api/v1/auth/logout'
}

/** 广播会话失效事件，由全局界面层统一清理状态并引导用户重新登录。 */
function notifySessionExpired() {
  window.dispatchEvent(new Event('qutc:session-expired'))
}

/**
 * 使用刷新 Cookie 换取新的访问 Cookie。
 * 返回值表示刷新是否成功；refreshInFlight 用于让并发的 401 请求复用本次刷新。
 */
async function refreshAccessToken() {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const response = await fetch(`${apiBaseUrl}/api/v1/auth/refresh`, {
          method: 'POST',
          headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
          credentials: 'include',
		  body: JSON.stringify({}),
        })
		const payload = await response.json().catch(() => null) as ApiEnvelope<{ access_token: string; session_expires_at?: string }> | null
        if (!response.ok || !payload || !('data' in payload)) return false
        // access token 存在 HttpOnly Cookie 中，前端只保存可安全读取的会话到期时间以安排退出。
        if (payload.data.session_expires_at) writeSessionExpiry(payload.data.session_expires_at)
        return true
      } catch {
        return false
      } finally {
        refreshInFlight = null
      }
    })()
  }
  return refreshInFlight
}

/**
 * 执行一次带 Cookie 的 API 请求；若普通业务请求首次收到 401，则刷新会话并仅重放一次。
 * path 是 API 相对路径，request 是可重复创建 RequestInit 的工厂，避免重放时复用已消费的请求体。
 */
async function fetchWithSessionRetry(path: string, request: () => RequestInit) {
	// 登录、注册和刷新接口本身不能触发刷新重试，否则令牌失效时会形成递归请求。
  let response = await fetch(`${apiBaseUrl}${path}`, request())
  if (apiMode === 'mock' || response.status !== 401 || isAuthPath(path)) return response
  if (!await refreshAccessToken()) {
    notifySessionExpired()
    return response
  }
  // 刷新成功后只重放一次原请求：第二次仍为 401 时交由全局会话失效事件统一处理。
  response = await fetch(`${apiBaseUrl}${path}`, request())
  if (response.status === 401) notifySessionExpired()
  return response
}

/** 发送 GET 请求并解包 API 信封；signal 可用于在组件卸载时取消请求。 */
export async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
	// 所有方法均将后端统一信封转换为 data 或 ApiClientError，视图层不需要重复解析错误格式。
  if (apiMode === 'mock') return (await loadMockApi()).mockGet<T>(path)

  const response = await fetchWithSessionRetry(path, () => ({ headers: headers(), credentials: 'include', signal }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

/** 获取列表并标准化为 Page，即使后端缺少部分分页元数据也提供可用的兜底值。 */
export async function getPage<T>(path: string, signal?: AbortSignal): Promise<Page<T>> {
  if (apiMode === 'mock') return (await loadMockApi()).mockGet<Page<T>>(path)

  const response = await fetchWithSessionRetry(path, () => ({ headers: headers(), credentials: 'include', signal }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T[]> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload) || !Array.isArray(payload.data)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  // 后端旧接口可能未返回完整分页元数据，使用数据长度兜底以保持调用方拿到稳定的 Page 结构。
  return {
    items: payload.data,
    page: payload.meta.page ?? 1,
    page_size: payload.meta.page_size ?? payload.data.length,
    total: payload.meta.total ?? payload.data.length,
  }
}

/** 发送 JSON POST 请求；body 为 undefined 时不附带请求体。 */
export async function post<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  if (apiMode === 'mock') return (await loadMockApi()).mockPost<T>(path, body)

  const response = await fetchWithSessionRetry(path, () => ({ method: 'POST', headers: headers(true), credentials: 'include', signal, body: body === undefined ? undefined : JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

/** 上传 multipart 表单；formData 通常包含文件与其业务元数据。 */
export async function upload<T>(path: string, formData: FormData): Promise<T> {
  if (apiMode === 'mock') return (await loadMockApi()).mockPost<T>(path, formData)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'POST', headers: headers(), credentials: 'include', body: formData }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

/** 发送局部更新 PATCH 请求，并返回后端保存后的数据。 */
export async function patch<T>(path: string, body: unknown): Promise<T> {
  if (apiMode === 'mock') return (await loadMockApi()).mockPatch<T>(path, body)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'PATCH', headers: headers(true), credentials: 'include', body: JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}

/** 发送幂等 PUT 请求，适用于完整替换或确定性写入。 */
export async function put<T>(path: string, body: unknown): Promise<T> {
  if (apiMode === 'mock') return (await loadMockApi()).mockPut<T>(path, body)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'PUT', headers: headers(true), credentials: 'include', body: JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}

/** 发送 DELETE 请求，并解包后端返回的操作结果。 */
export async function del<T>(path: string): Promise<T> {
  if (apiMode === 'mock') return (await loadMockApi()).mockDelete<T>(path)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'DELETE', headers: headers(), credentials: 'include' }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}
