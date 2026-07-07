import { useNavigate } from 'react-router-dom'
import { PiggyBank, CreditCard, Scale, Home } from 'lucide-react'

const items = [
  { to: '/deposit', icon: PiggyBank, label: 'Депозитный калькулятор', desc: 'Расчёт дохода по вкладу с капитализацией и налогами', gradient: 'from-emerald-400 to-emerald-600' },
  { to: '/credit', icon: CreditCard, label: 'Кредитный калькулятор', desc: 'График платежей, ПСК, переплата', gradient: 'from-blue-400 to-blue-600' },
  { to: '/affordability', icon: Scale, label: 'Оценка доступности', desc: 'Анализ кредитной нагрузки и стресс-тест', gradient: 'from-amber-400 to-orange-500' },
  { to: '/mortgage-rent', icon: Home, label: 'Ипотека vs Аренда', desc: 'Сравнение с учётом скрытых расходов', gradient: 'from-violet-400 to-violet-600' },
]

export function CalculatorsMenu({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in" />
      <div
        className="relative rounded-t-2xl w-full max-w-lg animate-slide-up shadow-2xl"
        style={{background: 'var(--bg-elevated)', border: '1px solid var(--border-default)'}}
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 pt-5 pb-3">
          <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>Калькуляторы</h2>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors" style={{color: 'var(--text-tertiary)'}}>
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
          </button>
        </div>
        <div className="p-2 space-y-1">
          {items.map(({ to, icon: Icon, label, desc, gradient }) => (
            <button key={to} onClick={() => { navigate(to); onClose() }}
              className="flex items-center gap-4 w-full p-3.5 rounded-xl card-hover text-left"
              style={{background: 'var(--bg-surface)'}}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--color-primary-50)'}
              onMouseLeave={e => e.currentTarget.style.background = 'var(--bg-surface)'}>
              <div className={`w-11 h-11 rounded-xl bg-gradient-to-br ${gradient} flex items-center justify-center shadow-lg flex-shrink-0`}>
                <Icon size={20} className="text-white" />
              </div>
              <div>
                <p className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>{label}</p>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{desc}</p>
              </div>
            </button>
          ))}
        </div>
        <div className="h-4" />
      </div>
    </div>
  )
}
