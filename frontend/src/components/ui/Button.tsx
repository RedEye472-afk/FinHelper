import type { ReactNode, ButtonHTMLAttributes } from 'react'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline'
  size?: 'sm' | 'md' | 'lg'
  children: ReactNode
  icon?: boolean
}

const sizes = { sm: 'px-3 py-1.5 text-xs', md: 'px-4 py-2 text-sm', lg: 'px-6 py-3 text-base' }
const iconSizes = { sm: 'p-1.5', md: 'p-2', lg: 'p-3' }

export function Button({ variant = 'primary', size = 'md', icon, className = '', children, ...props }: ButtonProps) {
  const sizeClass = icon ? iconSizes[size] : sizes[size]
  const variantClass = `btn-${variant}`
  return (
    <button className={`btn ${variantClass} ${sizeClass} ${className}`} {...props}>
      {children}
    </button>
  )
}
