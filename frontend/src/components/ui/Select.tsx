import type { SelectHTMLAttributes } from 'react'
import { ChevronDown } from 'lucide-react'

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> { label?: string; error?: string }

export function Select({ label, error, className = '', children, ...props }: SelectProps) {
  return (
    <div className="space-y-1.5">
      {label && <label className="block text-sm font-medium" style={{color: 'var(--text-primary)'}}>{label}</label>}
      <div className="relative">
        <select
          className={`input appearance-none ${error ? 'input-error' : ''} pr-9 ${className}`}
          style={{background: 'var(--bg-surface)'}}
          {...props}
        >
          {children}
        </select>
        <ChevronDown size={16} className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none" style={{color: 'var(--text-tertiary)'}} />
      </div>
      {error && <p className="text-xs" style={{color: 'var(--color-danger-500)'}}>{error}</p>}
    </div>
  )
}
