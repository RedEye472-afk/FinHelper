import { useSettings } from '../../hooks/useSettings'
import { toDecimal, formatMoney, formatCompact } from '../../lib/money'
import type { Decimal } from 'decimal.js'

interface MoneyDisplayProps {
  amount: number | string | Decimal
  type?: 'income' | 'expense' | 'neutral'
  size?: 'sm' | 'md' | 'lg'
  showSign?: boolean
  compact?: boolean
  className?: string
}

const sizes = { sm: 'text-sm', md: 'text-base', lg: 'text-xl' }
const colors = { income: 'text-emerald-600', expense: 'text-red-500', neutral: '' }

export function MoneyDisplay({ amount, type = 'neutral', size = 'md', showSign = false, compact = false, className = '' }: MoneyDisplayProps) {
  const { symbol, hideBalance } = useSettings()
  const d = toDecimal(String(amount))
  const isNeg = d.isNegative()
  const sign = showSign ? (isNeg ? '−' : '+') : ''
  const absD = showSign && isNeg ? d.abs() : d

  if (hideBalance) return <span className={`font-mono-money ${sizes[size]} ${className}`} style={{ color: 'var(--text-tertiary)' }}>••••</span>

  const formatted = compact ? formatCompact(absD) : formatMoney(absD)

  return (
    <span className={`font-mono-money font-semibold ${sizes[size]} ${colors[type]} ${className}`}
      style={type === 'neutral' ? { color: 'var(--text-primary)' } : {}}>
      {sign}{formatted}
    </span>
  )
}