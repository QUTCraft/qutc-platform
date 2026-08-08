import { portalApi } from '@/api/portal'
import type { PortalManifest } from '@/api/types'

const runtimeTimeoutMs = 1800
const fallbackStorageKey = 'qutc.portal.runtime_fallback'
const platformPaths = ['/admin', '/login', '/register', '/invite', '/apply']

export interface PortalFallbackRecord {
  portal_id: string
  version: string
  reason: 'configuration_timeout' | 'configuration_unavailable' | 'entry_timeout' | 'entry_unavailable' | 'entry_marker_mismatch'
  occurred_at: string
}

export async function bootstrapPortalRuntime(): Promise<void> {
  if (!isPublicPlatformRoute(window.location.pathname)) return
  if (forceDefaultPortal()) {
    clearPortalFallback()
    return
  }

  const configurationController = new AbortController()
  const configurationTimer = window.setTimeout(() => configurationController.abort(), runtimeTimeoutMs)
  let manifest: PortalManifest
  try {
    const configuration = await portalApi.getRuntimeConfiguration(configurationController.signal)
    if (configuration.source !== 'active' || isBuiltInEntry(configuration.manifest.entry)) {
      clearPortalFallback()
      return
    }
    manifest = configuration.manifest
  } catch (error) {
    recordFallback(
      { id: 'unknown', version: 'unknown' },
      error instanceof DOMException && error.name === 'AbortError' ? 'configuration_timeout' : 'configuration_unavailable',
    )
    return
  } finally {
    window.clearTimeout(configurationTimer)
  }

  const probe = await probePortalEntry(manifest)
  if (!probe.ok) {
    recordFallback(manifest, probe.reason)
    return
  }

  clearPortalFallback()
  window.location.replace(manifest.entry)
}

export function readPortalFallback(): PortalFallbackRecord | null {
  try {
    return JSON.parse(window.sessionStorage.getItem(fallbackStorageKey) ?? 'null') as PortalFallbackRecord | null
  } catch {
    return null
  }
}

export function clearPortalFallback(): void {
  window.sessionStorage.removeItem(fallbackStorageKey)
}

function isPublicPlatformRoute(pathname: string): boolean {
  if (pathname.startsWith('/portals/')) return false
  return !platformPaths.some((path) => pathname === path || pathname.startsWith(`${path}/`))
}

function forceDefaultPortal(): boolean {
  return new URLSearchParams(window.location.search).get('portal') === 'md3'
}

function isBuiltInEntry(entry: string): boolean {
  return entry === '/index.html' || entry === '/'
}

async function probePortalEntry(manifest: PortalManifest): Promise<{ ok: true } | { ok: false; reason: PortalFallbackRecord['reason'] }> {
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), runtimeTimeoutMs)
  try {
    const response = await fetch(manifest.entry, {
      method: 'GET',
      cache: 'no-store',
      credentials: 'same-origin',
      signal: controller.signal,
      headers: { Accept: 'text/html' },
    })
    if (!response.ok || !(response.headers.get('content-type') ?? '').toLowerCase().includes('text/html')) {
      return { ok: false, reason: 'entry_unavailable' }
    }
    const html = await response.text()
    const document = new DOMParser().parseFromString(html, 'text/html')
    const portalID = document.querySelector<HTMLMetaElement>('meta[name="qutc-portal-id"]')?.content
    if (portalID !== manifest.id) return { ok: false, reason: 'entry_marker_mismatch' }
    return { ok: true }
  } catch (error) {
    return { ok: false, reason: error instanceof DOMException && error.name === 'AbortError' ? 'entry_timeout' : 'entry_unavailable' }
  } finally {
    window.clearTimeout(timer)
  }
}

function recordFallback(manifest: Pick<PortalManifest, 'id' | 'version'>, reason: PortalFallbackRecord['reason']) {
  const record: PortalFallbackRecord = {
    portal_id: manifest.id,
    version: manifest.version,
    reason,
    occurred_at: new Date().toISOString(),
  }
  window.sessionStorage.setItem(fallbackStorageKey, JSON.stringify(record))
  console.warn('[qutc.portal.runtime_fallback]', record)
  window.dispatchEvent(new CustomEvent('qutc:portal-fallback', { detail: record }))
}
