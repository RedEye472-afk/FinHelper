import { useLocation, useNavigate } from 'react-router-dom'
import { ArrowLeft, Bell, WifiOff } from 'lucide-react'
import { useSettings } from '../../hooks/useSettings'
import { useDemoMode } from '../../api/queries'

const pageTitles: Record<string, string> = {
  '/dashboard': 'Обзор', '/operations': 'Операции', '/budgets': 'Бюджеты',
  '/goals': 'Цели', '/settings': 'Настройки', '/deposit': 'Депозитный калькулятор',
  '/credit': 'Кредитный калькулятор', '/affordability': 'Оценка доступности',
  '/mortgage-rent': 'Ипотека vs Аренда', '/accounts': 'Счета',
}

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { initials } = useSettings()
  const demo = useDemoMode()
  const path = location.pathname
  const title = pageTitles[path] || 'FinHelper'
  const isMain = path === '/dashboard'

  return (
    <header
      className={`sticky top-0 z-20 ${isMain ? '' : 'border-b'}`}
      style={isMain
        ? { background: 'transparent' }
        : { background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}
    >
      <div className="flex items-center justify-between h-14 px-4 max-w-6xl mx-auto w-full">
        <div className="flex items-center gap-3">
          {!isMain && (
            <button
              onClick={() => navigate(-1)}
              className="p-1 -ml-1 transition-colors rounded-lg btn-press"
              style={{ color: 'var(--text-secondary)' }}
            >
              <ArrowLeft size={22} />
            </button>
          )}
          <h1
            className="font-semibold text-lg tracking-tight"
            style={isMain ? { color: 'var(--text-primary)' } : { color: 'var(--text-primary)' }}
          >
            {title}
          </h1>
          {demo && (
            <span
              className="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium"
              style={{ background: 'rgba(245,158,11,0.12)', color: '#F59E0B' }}
            >
              <WifiOff size={10} />
              Демо
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <button
            className="p-2 rounded-xl transition-colors btn-press relative"
            style={{ color: 'var(--text-tertiary)' }}
          >
            <Bell size={20} strokeWidth={1.5} />
            <span
              className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full"
              style={{ background: 'var(--color-danger-500)' }}
            />
          </button>
          <button
            onClick={() => navigate('/settings')}
            className="w-9 h-9 rounded-[10px] flex items-center justify-center text-white text-xs font-semibold btn-press transition-transform hover:scale-105"
            style={{
              background: 'linear-gradient(135deg, var(--color-primary-600), #A78BFA)',
              boxShadow: '0 0 16px rgba(110,86,207,0.25)',
            }}
          >
            {initials}
          </button>
        </div>
      </div>
    </header>
  )
}
