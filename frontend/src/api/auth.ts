import { apiRequest, setTokens, clearTokens } from './client'
import type { User } from '../types'

type AuthResponse = {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export async function register(email: string, password: string): Promise<User> {
  const data = await apiRequest<AuthResponse>('POST', '/api/v1/auth/register', { email, password }, { skipAuth: true })
  setTokens(data.access_token, data.refresh_token)
  return getMe()
}

export async function login(email: string, password: string): Promise<User> {
  const data = await apiRequest<AuthResponse>('POST', '/api/v1/auth/login', { email, password }, { skipAuth: true })
  setTokens(data.access_token, data.refresh_token)
  return getMe()
}

export async function getMe(): Promise<User> {
  return apiRequest<User>('GET', '/api/v1/me')
}

export function logout() {
  clearTokens()
}
