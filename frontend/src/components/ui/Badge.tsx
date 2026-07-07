import type { ReactNode } from 'react'

export function Badge({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <span className={`badge ${className}`}
    style={{background: 'var(--color-primary-50)', color: 'var(--color-primary-700)'}}>
    {children}
  </span>
}

export function BadgeDanger({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <span className={`badge ${className}`}
    style={{background: 'var(--color-danger-50)', color: 'var(--color-danger-600)'}}>
    {children}
  </span>
}

export function BadgeWarning({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <span className={`badge ${className}`}
    style={{background: 'var(--color-warning-400)', color: '#78350f', opacity: 0.9}}>
    {children}
  </span>
}
