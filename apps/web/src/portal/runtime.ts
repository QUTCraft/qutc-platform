import { portalApi } from '@/api/portal'
import type { PortalManifest } from '@/api/types'

// 门户配置和入口探测都设置短超时：公开首页应优先可用，外部定制门户异常时回退默认界面。
const runtimeTimeoutMs = 1800
const fallbackStorageKey = 'qutc.portal.runtime_fallback'
const platformPaths = ['/admin', '/login', '/register', '/invite', '/apply']

export interface PortalFallbackRecord {
  portal_id: string
  version: string
  reason: 'configuration_timeout' | 'configuration_unavailable' | 'entry_timeout' | 'entry_unavailable' | 'entry_marker_mismatch'
  occurred_at: string
}

/**
 * 在公共平台页面启动时尝试接管到已启用的自定义门户。
 * 配置、入口或标记任一环节失败时保持默认门户，并记录本标签页可读的回退原因。
 */
export async function bootstrapPortalRuntime(): Promise<void> {
	// 管理、登录和申请等平台页面不能被自定义门户接管，避免破坏内部工作流。
  if (!isPublicPlatformRoute(window.location.pathname)) return
  if (forceDefaultPortal()) {
    clearPortalFallback()
    return
  }

  // 先取得当前启用的门户清单；请求使用 AbortController，确保慢网络不会阻塞应用启动。
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

  // 跳转前先验证入口可访问且确实属于期望门户，防止配置错误带来白屏或跳到错误页面。
  const probe = await probePortalEntry(manifest)
  if (!probe.ok) {
    recordFallback(manifest, probe.reason)
    return
  }

  clearPortalFallback()
  window.location.replace(manifest.entry)
}

/** 读取本标签页最近一次门户接管失败记录；解析异常或没有记录时返回 null。 */
export function readPortalFallback(): PortalFallbackRecord | null {
	// 失败记录仅存于当前标签页，供诊断 UI 使用，不会跨会话污染用户的门户选择。
  try {
    return JSON.parse(window.sessionStorage.getItem(fallbackStorageKey) ?? 'null') as PortalFallbackRecord | null
  } catch {
    return null
  }
}

/** 删除本标签页的门户回退诊断记录。 */
export function clearPortalFallback(): void {
  window.sessionStorage.removeItem(fallbackStorageKey)
}

/** 判断 pathname 是否属于可由门户接管的公开页面，而非后台或认证流程页面。 */
function isPublicPlatformRoute(pathname: string): boolean {
  if (pathname.startsWith('/portals/')) return false
  return !platformPaths.some((path) => pathname === path || pathname.startsWith(`${path}/`))
}

/** 判断 URL 是否显式要求保留内置 Material Design 3 门户，便于排障和演示。 */
function forceDefaultPortal(): boolean {
  return new URLSearchParams(window.location.search).get('portal') === 'md3'
}

/** 判断清单入口是否仍指向内置首页；内置入口无需重定向或探测。 */
function isBuiltInEntry(entry: string): boolean {
  return entry === '/index.html' || entry === '/'
}

/**
 * 在跳转前探测 manifest.entry 的可访问性与所有权标记。
 * 成功仅表示该入口适合接管当前页面；失败结果携带可展示或上报的明确原因。
 */
async function probePortalEntry(manifest: PortalManifest): Promise<{ ok: true } | { ok: false; reason: PortalFallbackRecord['reason'] }> {
	// 除 HTTP 状态外，还读取入口内的 qutc-portal-id 标记，避免 CDN/反向代理把错误 HTML 当成门户入口。
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

/**
 * 持久化并广播门户回退事件。
 * manifest 只需提供标识和版本，reason 用于区分配置超时、入口不可用和标记不匹配。
 */
function recordFallback(manifest: Pick<PortalManifest, 'id' | 'version'>, reason: PortalFallbackRecord['reason']) {
	// 同时写入 sessionStorage、控制台和自定义事件：分别服务于页面诊断、开发排查与监控集成。
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
