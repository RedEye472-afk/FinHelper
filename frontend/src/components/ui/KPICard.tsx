import { TrendingUp, TrendingDown } from 'lucide-react'

interface KPICardProps { label: string; value: string; trend?: number; prefix?: string }

export function KPICard({ label, value, trend, prefix }: KPICardProps) {
  return (
    <div className="glass p-4 rounded-2xl">
      <p className="text-xs font-medium mb-1" style={{color: 'var(--text-secondary)'}}>{label}</p>
      <p className="text-xl font-bold font-mono-money gradient-text">{prefix}{value}</p>
      {trend !== undefined && (
        <div className={`flex items-center gap-1 mt-1.5 text-xs ${trend >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
          {trend >= 0 ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          <span>{Math.abs(trend)}% за месяц</span>
        </div>
      )}
    </div>
  )
}
