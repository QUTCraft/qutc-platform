const sessionExpiryCookieName = 'qutc_session_expires'

function cookieValue(name: string): string | null {
  const prefix = `${encodeURIComponent(name)}=`
  const entry = document.cookie.split(';').map((item) => item.trim()).find((item) => item.startsWith(prefix))
  return entry ? decodeURIComponent(entry.slice(prefix.length)) : null
}

export function readSessionExpiry(): number | null {
  const raw = cookieValue(sessionExpiryCookieName)
  if (!raw) return null
  const seconds = Number(raw)
  return Number.isFinite(seconds) && seconds > 0 ? seconds * 1000 : null
}

export function writeSessionExpiry(expiresAt: string): void {
  const expires = new Date(expiresAt)
  if (!Number.isFinite(expires.getTime())) return
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `${sessionExpiryCookieName}=${Math.floor(expires.getTime() / 1000)}; Path=/; Expires=${expires.toUTCString()}; SameSite=Strict${secure}`
}

export function clearSessionExpiry(): void {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = `${sessionExpiryCookieName}=; Path=/; Max-Age=0; SameSite=Strict${secure}`
}
