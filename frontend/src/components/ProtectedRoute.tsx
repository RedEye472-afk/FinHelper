import { type ReactNode } from 'react'

export function ProtectedRoute({ children }: { children: ReactNode }) {
  // MVP: no auth wall, always render children
  return <>{children}</>
}