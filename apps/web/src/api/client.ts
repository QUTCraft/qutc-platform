import { mockGet, mockPost } from '@/api/mock'
import { getAccessToken } from '@/auth/token-storage'
import type { ApiEnvelope, Page } from '@/api/types'

const apiMode = import.meta.env.VITE_API_MODE ?? 'mock'
const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080').replace(/\/$/, '')

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

export async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  if (apiMode === 'mock') return mockGet<T>(path)

  const response = await fetch(`${apiBaseUrl}${path}`, {
    headers: headers(),
    credentials: 'include',
    signal,
  })
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}

export async function getPage<T>(path: string, signal?: AbortSignal): Promise<Page<T>> {
  if (apiMode === 'mock') return mockGet<Page<T>>(path)

  const response = await fetch(`${apiBaseUrl}${path}`, {
    headers: headers(),
    credentials: 'include',
    signal,
  })
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

  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: 'POST',
    headers: headers(true),
    credentials: 'include',
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | { error?: { code?: string; message?: string } } | null
  if (!response.ok || !payload || !('data' in payload)) {
    const error = payload && 'error' in payload ? payload.error : undefined
    throw new ApiClientError(response.status, error?.code ?? 'network.request_failed', error?.message ?? '请求失败，请稍后重试。')
  }
  return payload.data
}
