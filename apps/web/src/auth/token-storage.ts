let accessToken = ''

export function getAccessToken() { return accessToken || null }
export function saveTokens(token: string) { accessToken = token }
export function clearTokens() { accessToken = '' }
