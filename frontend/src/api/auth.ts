import { apiRequest, setTokens, clearTokens } from './client'
import type { User } from '../types'

type AuthResponse = {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

/** Login may return requiresVerification=true for unverified accounts. */
export type LoginResponse = AuthResponse & {
  requires_verification?: boolean
  email?: string
  message?: string
}

/**
 * Register creates a user and returns a message (not tokens).
 * The user must verify their email via verifyEmail() before login works.
 */
export async function register(email: string, password: string): Promise<{ message: string; user_id: number; user_hash: string }> {
  const data = await apiRequest<{ message: string; user_id: number; user_hash: string }>(
    'POST', '/api/v1/auth/register', { email, password }, { skipAuth: true }
  )
  return data
}

/**
 * Login returns either { access_token, refresh_token, ... } for verified users,
 * or { requires_verification: true, email, message } for unverified users.
 * We export the raw response so callers can inspect requires_verification.
 */
export async function login(email: string, password: string): Promise<LoginResponse> {
  const data = await apiRequest<LoginResponse>(
    'POST', '/api/v1/auth/login', { email, password }, { skipAuth: true }
  )
  if (!('requires_verification' in data)) {
    setTokens(data.access_token, data.refresh_token)
  }
  return data
}

/**
 * VerifyEmail confirms the 6-digit code. On success, returns tokens
 * and the user is logged in.
 */
export async function verifyEmail(code: string): Promise<AuthResponse> {
  const data = await apiRequest<AuthResponse>('POST', '/api/v1/auth/verify-email', { code }, { skipAuth: true })
  setTokens(data.access_token, data.refresh_token)
  return data
}

/**
 * SendCode resends the verification code to the given email.
 */
export async function sendCode(email: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>('POST', '/api/v1/auth/send-code', { email }, { skipAuth: true })
}

/**
 * ForgotPassword sends a reset link to the given email.
 */
export async function forgotPassword(email: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>('POST', '/api/v1/auth/forgot-password', { email }, { skipAuth: true })
}

/**
 * ResetPassword consumes the reset token and updates the password.
 */
export async function resetPassword(email: string, token: string, password: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>(
    'POST', '/api/v1/auth/reset-password', { email, token, password }, { skipAuth: true }
  )
}

export async function getMe(): Promise<User> {
  return apiRequest<User>('GET', '/api/v1/me')
}

export function logout() {
  clearTokens()
}
