import type { ReactNode, CSSProperties } from 'react'

interface CardProps { children: ReactNode; className?: string; style?: CSSProperties; onClick?: () => void; glass?: boolean; glow?: boolean }
export function Card({ children, className = '', style, onClick, glass, glow }: CardProps) {
  const Comp = onClick ? 'button' : 'div'
  return (
    <Comp
      onClick={onClick}
      style={style}
      className={`${glass ? 'card-glass' : 'card'} p-4 ${onClick ? 'w-full text-left card-hover cursor-pointer' : ''} ${glow ? 'animate-pulse-glow' : ''} ${className}`}
    >
      {children}
    </Comp>
  )
}
