import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import { getAccessToken, clearTokens } from '../api/client'
import * as authApi from '../api/auth'
import type { User } from '../types'

interface AuthContextType {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<import('../api/auth').LoginResponse>
  register: (email: string, password: string) => Promise<{ message: string; user_id: number; user_hash: string }>
  verifyEmail: (code: string) => Promise<void>
  logout: () => void
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refreshUser = useCallback(async () => {
    if (!getAccessToken()) {
      const demoUser: User = { id: 10, email: 'demo@finhelper.ru', created_at: new Date().toISOString() }
      setUser(demoUser)
      setLoading(false)
      return
    }
    try {
      const u = await authApi.getMe()
      setUser(u)
    } catch {
      clearTokens()
      const demoUser: User = { id: 10, email: 'demo@finhelper.ru', created_at: new Date().toISOString() }
      setUser(demoUser)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { refreshUser() }, [refreshUser])

  const login = useCallback(async (email: string, password: string) => {
    const data = await authApi.login(email, password)
    if (!data.requires_verification) {
      const u = await authApi.getMe()
      setUser(u)
    }
    return data
  }, [])

  const register = useCallback(async (email: string, password: string) => {
    const data = await authApi.register(email, password)
    return data
  }, [])

  const verifyEmail = useCallback(async (code: string) => {
    await authApi.verifyEmail(code)
    const u = await authApi.getMe()
    setUser(u)
  }, [])

  const logout = useCallback(() => {
    authApi.logout()
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, register, verifyEmail, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}