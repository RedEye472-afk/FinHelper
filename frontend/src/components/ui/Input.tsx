import type { InputHTMLAttributes } from 'react'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  hint?: string
  leftIcon?: React.ReactNode
}

export function Input({ label, error, hint, leftIcon, className = '', ...props }: InputProps) {
  return (
    <div className="space-y-1.5">
      {label && <label className="block text-sm font-medium" style={{color: 'var(--text-primary)'}}>{label}</label>}
      <div className="relative">
        {leftIcon && <div className="absolute left-3 top-1/2 -translate-y-1/2" style={{color: 'var(--text-tertiary)'}}>{leftIcon}</div>}
        <input
          className={`input ${leftIcon ? 'pl-9' : ''} ${error ? 'input-error' : ''} ${className}`}
          {...props}
        />
      </div>
      {error && <p className="text-xs" style={{color: 'var(--color-danger-500)'}}>{error}</p>}
      {hint && !error && <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{hint}</p>}
    </div>
  )
}
