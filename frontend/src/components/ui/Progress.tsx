interface ProgressProps { value: number; max?: number; size?: 'sm' | 'md' | 'lg'; showLabel?: boolean; className?: string; color?: string }

const sizes = { sm: 'h-1.5', md: 'h-2.5', lg: 'h-4' }

export function Progress({ value, max = 100, size = 'md', showLabel, className = '', color }: ProgressProps) {
  const pct = Math.min(Math.max(0, (value / max) * 100), 100)
  return (
    <div className={`w-full ${className}`}>
      <div className={`w-full rounded-full ${sizes[size]}`} style={{background: 'var(--border-default)'}}>
        <div
          className={`${sizes[size]} rounded-full progress-bar`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && <p className="text-xs mt-1 text-right" style={{color: 'var(--text-tertiary)'}}>{pct.toFixed(0)}%</p>}
    </div>
  )
}
