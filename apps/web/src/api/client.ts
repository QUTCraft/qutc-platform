import { mockDelete, mockGet, mockPatch, mockPost, mockPut } from '@/api/mock'
import { clearTokens, getAccessToken, saveTokens } from '@/auth/token-storage'
import type { ApiEnvelope, Page } from '@/api/types'

const apiMode = import.meta.env.VITE_API_MODE ?? 'mock'
// Production uses the web container's same-origin /api reverse proxy, so the
// browser never needs an administrator-maintained API hostname. A Vite value
// remains available for split-port local development only.
const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL?.trim() || window.location.origin).replace(/\/$/, '')

export function resolveApiUrl(path: string): string {
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

function headers(json = false) {
  const token = getAccessToken()
  return {
    Accept: 'application/json',
    ...(json ? { 'Content-Type': 'application/json' } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

let refreshInFlight: Promise<boolean> | null = null

function isAuthPath(path: string) {
  return path === '/api/v1/auth/login' || path === '/api/v1/auth/register' || path === '/api/v1/auth/refresh' || path === '/api/v1/auth/logout'
}

function notifySessionExpired() {
  clearTokens()
  window.dispatchEvent(new Event('qutc:session-expired'))
}

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
		const payload = await response.json().catch(() => null) as ApiEnvelope<{ access_token: string }> | null
        if (!response.ok || !payload || !('data' in payload)) return false
		saveTokens(payload.data.access_token)
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

async function fetchWithSessionRetry(path: string, request: () => RequestInit) {
  let response = await fetch(`${apiBaseUrl}${path}`, request())
  if (apiMode === 'mock' || response.status !== 401 || isAuthPath(path)) return response
  if (!await refreshAccessToken()) {
    notifySessionExpired()
    return response
  }
  response = await fetch(`${apiBaseUrl}${path}`, request())
  if (response.status === 401) notifySessionExpired()
  return response
}

export async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  if (apiMode === 'mock') return mockGet<T>(path)

  const response = await fetchWithSessionRetry(path, () => ({ headers: headers(), credentials: 'include', signal }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

export async function getPage<T>(path: string, signal?: AbortSignal): Promise<Page<T>> {
  if (apiMode === 'mock') return mockGet<Page<T>>(path)

  const response = await fetchWithSessionRetry(path, () => ({ headers: headers(), credentials: 'include', signal }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T[]> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload) || !Array.isArray(payload.data)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return {
    items: payload.data,
    page: payload.meta.page ?? 1,
    page_size: payload.meta.page_size ?? payload.data.length,
    total: payload.meta.total ?? payload.data.length,
  }
}

export async function post<T>(path: string, body?: unknown): Promise<T> {
  if (apiMode === 'mock') return mockPost<T>(path, body)

  const response = await fetchWithSessionRetry(path, () => ({ method: 'POST', headers: headers(true), credentials: 'include', body: body === undefined ? undefined : JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

export async function upload<T>(path: string, formData: FormData): Promise<T> {
  if (apiMode === 'mock') return mockPost<T>(path, formData)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'POST', headers: headers(), credentials: 'include', body: formData }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

export async function patch<T>(path: string, body: unknown): Promise<T> {
  if (apiMode === 'mock') return mockPatch<T>(path, body)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'PATCH', headers: headers(true), credentials: 'include', body: JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}

export async function put<T>(path: string, body: unknown): Promise<T> {
  if (apiMode === 'mock') return mockPut<T>(path, body)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'PUT', headers: headers(true), credentials: 'include', body: JSON.stringify(body) }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}

export async function del<T>(path: string): Promise<T> {
  if (apiMode === 'mock') return mockDelete<T>(path)
  const response = await fetchWithSessionRetry(path, () => ({ method: 'DELETE', headers: headers(), credentials: 'include' }))
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) { const error = payload && 'error' in payload ? payload.error : undefined; throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。') }
  return payload.data
}
