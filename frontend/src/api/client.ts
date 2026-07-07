const ACCESS_KEY = 'fh_access'
const REFRESH_KEY = 'fh_refresh'

let accessToken: string | null = null
let refreshToken: string | null = null

/** Restore tokens from sessionStorage into module vars. Call on app init. */
export function initAuthFromStorage() {
  accessToken = sessionStorage.getItem(ACCESS_KEY)
  refreshToken = sessionStorage.getItem(REFRESH_KEY)
}

// Restore on module load
initAuthFromStorage()

export function setTokens(access: string, refresh: string) {
  accessToken = access
  refreshToken = refresh
  sessionStorage.setItem(ACCESS_KEY, access)
  sessionStorage.setItem(REFRESH_KEY, refresh)
}

export function clearTokens() {
  accessToken = null
  refreshToken = null
  sessionStorage.removeItem(ACCESS_KEY)
  sessionStorage.removeItem(REFRESH_KEY)
}

export function getAccessToken() { return accessToken }
export function getRefreshToken() { return refreshToken }
export function isAuthenticated() { return accessToken !== null || refreshToken !== null }

async function tryRefresh(): Promise<boolean> {
  if (!refreshToken) return false
  try {
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) { clearTokens(); return false }
    const data = await res.json() as { access_token: string; refresh_token: string }
    setTokens(data.access_token, data.refresh_token)
    return true
  } catch {
    clearTokens()
    return false
  }
}

export async function apiRequest<T>(method: string, path: string, body?: unknown, opts?: { skipAuth?: boolean }): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (!opts?.skipAuth && accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  // Rate limiting: don't make network call if not authenticated for authed requests
  if (!opts?.skipAuth && !isAuthenticated()) {
    throw new ApiRequestError(401, 'Not authenticated')
  }

  const res = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined })

  if (res.status === 401 && !opts?.skipAuth) {
    const refreshed = await tryRefresh()
    if (refreshed) {
      headers['Authorization'] = `Bearer ${accessToken}`
      const retry = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined })
      if (!retry.ok) { const err = await retry.json().catch(() => ({})); throw new ApiRequestError(retry.status, err.detail || err.title || 'Request failed') }
      return retry.json()
    }
    clearTokens()
    throw new ApiRequestError(401, 'Session expired')
  }

  if (!res.ok) { const err = await res.json().catch(() => ({})); throw new ApiRequestError(res.status, err.detail || err.title || err.message || 'Request failed') }
  if (res.status === 204) return undefined as T
  return res.json()
}

export class ApiRequestError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiRequestError'
  }
}