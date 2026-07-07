import { useLocation, useNavigate } from 'react-router-dom'
import { ArrowLeft, Bell } from 'lucide-react'
import { useSettings } from '../../hooks/useSettings'

const pageTitles: Record<string, string> = {
  '/dashboard': 'Главная', '/operations': 'Операции', '/budgets': 'Бюджеты',
  '/goals': 'Цели', '/settings': 'Настройки', '/deposit': 'Депозитный калькулятор',
  '/credit': 'Кредитный калькулятор', '/affordability': 'Оценка доступности',
  '/mortgage-rent': 'Ипотека vs Аренда', '/accounts': 'Счета', '/onboarding': 'Обучение',
}

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { initials } = useSettings()
  const path = location.pathname
  const title = pageTitles[path] || 'FinHelper'
  const isMain = path === '/dashboard'

  return (
    <header className={`sticky top-0 z-30 ${isMain ? '' : 'border-b'}`}
      style={isMain ? undefined : {background: 'var(--bg-surface)', borderColor: 'var(--border-default)'}}>
      <div className="max-w-2xl mx-auto px-4 h-14 flex items-center justify-between">
        <div className="flex items-center gap-3">
          {!isMain && (
            <button onClick={() => navigate(-1)} className="p-1 -ml-1 transition-colors rounded-lg btn-press"
              style={{color: 'var(--text-secondary)'}}>
              <ArrowLeft size={22} />
            </button>
          )}
          <h1 className={`font-semibold ${isMain ? 'text-white text-lg' : ''}`}
            style={isMain ? undefined : {color: 'var(--text-primary)'}}>{title}</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="p-2 rounded-lg transition-colors btn-press relative"
            style={{color: 'var(--text-tertiary)'}}>
            <Bell size={20} />
            <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full animate-pulse-glow"
              style={{background: 'var(--color-danger-500)'}} />
          </button>
          <button onClick={() => navigate('/settings')}
            className="w-9 h-9 rounded-full bg-gradient-primary flex items-center justify-center text-white text-xs font-semibold shadow-lg btn-press transition-transform hover:scale-105">
            {initials}
          </button>
        </div>
      </div>
    </header>
  )
}
