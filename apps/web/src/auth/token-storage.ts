const accessTokenKey = 'qutc.access_token'
const refreshTokenKey = 'qutc.refresh_token'

export function getAccessToken() { return window.localStorage.getItem(accessTokenKey) }
export function getRefreshToken() { return window.localStorage.getItem(refreshTokenKey) }
export function saveTokens(accessToken: string, refreshToken: string) { window.localStorage.setItem(accessTokenKey, accessToken); window.localStorage.setItem(refreshTokenKey, refreshToken) }
export function clearTokens() { window.localStorage.removeItem(accessTokenKey); window.localStorage.removeItem(refreshTokenKey) }
