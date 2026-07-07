import { useNavigate } from 'react-router-dom'
import { Wallet, GraduationCap, Settings as SettingsIcon, BookOpen, PiggyBank, CreditCard, Scale, Home } from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { useSettings } from '../../hooks/useSettings'

export function MoreMenu({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { name, initials } = useSettings()

  const mainItems = [
    { to: '/accounts', icon: Wallet, label: 'Счета', desc: 'Управление счетами и переводы' },
    { to: '/budgets', icon: GraduationCap, label: 'Бюджеты', desc: 'Лимиты по категориям' },
  ]

  const calcItems = [
    { to: '/deposit', icon: PiggyBank, label: 'Депозит', desc: 'Доход по вкладу' },
    { to: '/credit', icon: CreditCard, label: 'Кредит', desc: 'ПСК и переплата' },
    { to: '/affordability', icon: Scale, label: 'Доступность', desc: 'Кредитная нагрузка' },
    { to: '/mortgage-rent', icon: Home, label: 'Ипотека vs Аренда', desc: 'Сравнение расходов' },
  ]

  const bottomItems = [
    { to: '/settings', icon: SettingsIcon, label: 'Настройки', desc: 'Тема, валюта, видимость' },
    { to: '/onboarding', icon: BookOpen, label: 'Обучение', desc: 'Как пользоваться FinHelper' },
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in" />
      <div
        className="relative rounded-t-2xl w-full max-w-lg animate-slide-up shadow-2xl max-h-[85vh] flex flex-col"
        style={{background: 'var(--bg-elevated)', border: '1px solid var(--border-default)'}}
        onClick={e => e.stopPropagation()}
      >
        <div className="absolute top-2 left-1/2 -translate-x-1/2 w-10 h-1 rounded-full" style={{background: 'var(--text-tertiary)'}} />
        <div className="flex items-center justify-between px-5 pt-5 pb-3">
          <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>Меню</h2>
          <button onClick={onClose} className="p-1.5 rounded-lg transition-colors btn-press" style={{color: 'var(--text-tertiary)'}}>
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
          </button>
        </div>

        <div className="mx-5 mb-3 p-3.5 rounded-xl flex items-center gap-3"
          style={{background: 'var(--color-primary-50)'}}>
          <div className="w-10 h-10 rounded-full bg-gradient-primary flex items-center justify-center text-white text-sm font-semibold shadow-md">{initials}</div>
          <div>
            <p className="text-sm font-medium" style={{color: 'var(--color-primary-900)'}}>{user?.email?.split('@')[0] || name || 'Пользователь'}</p>
            <p className="text-xs" style={{color: 'var(--color-primary-600)'}}>{user?.email || ''}</p>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-2 space-y-1 pb-4">
          {/* Main items */}
          {mainItems.map(({ to, icon: Icon, label, desc }) => (
            <button key={to} onClick={() => { navigate(to); onClose() }}
              className="flex items-center gap-4 w-full p-3.5 rounded-xl card-hover text-left btn-press"
              style={{background: 'var(--bg-surface)'}}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--color-primary-50)'}
              onMouseLeave={e => e.currentTarget.style.background = 'var(--bg-surface)'}>
              <div className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
                style={{background: 'var(--color-primary-50)', color: 'var(--color-primary-600)'}}>
                <Icon size={20} />
              </div>
              <div>
                <p className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>{label}</p>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{desc}</p>
              </div>
            </button>
          ))}

          {/* Divider */}
          <div className="py-1">
            <p className="text-[10px] uppercase tracking-wider font-medium px-2" style={{color: 'var(--text-tertiary)'}}>
              Калькуляторы
            </p>
          </div>

          {/* Calculator items */}
          {calcItems.map(({ to, icon: Icon, label, desc }) => (
            <button key={to} onClick={() => { navigate(to); onClose() }}
              className="flex items-center gap-4 w-full p-3 rounded-xl card-hover text-left btn-press"
              style={{background: 'var(--bg-surface)'}}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--color-primary-50)'}
              onMouseLeave={e => e.currentTarget.style.background = 'var(--bg-surface)'}>
              <div className="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0"
                style={{background: 'var(--color-primary-50)', color: 'var(--color-primary-600)'}}>
                <Icon size={18} />
              </div>
              <div>
                <p className="text-sm font-medium" style={{color: 'var(--text-primary)'}}>{label}</p>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{desc}</p>
              </div>
            </button>
          ))}

          {/* Divider */}
          <div className="pt-2" />

          {/* Bottom items */}
          {bottomItems.map(({ to, icon: Icon, label, desc }) => (
            <button key={to} onClick={() => { navigate(to); onClose() }}
              className="flex items-center gap-4 w-full p-3.5 rounded-xl card-hover text-left btn-press"
              style={{background: 'var(--bg-surface)'}}
              onMouseEnter={e => e.currentTarget.style.background = 'var(--color-primary-50)'}
              onMouseLeave={e => e.currentTarget.style.background = 'var(--bg-surface)'}>
              <div className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
                style={{background: 'var(--color-primary-50)', color: 'var(--color-primary-600)'}}>
                <Icon size={20} />
              </div>
              <div>
                <p className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>{label}</p>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{desc}</p>
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}