import { useCallback, useEffect, useState } from 'react'
import { Outlet, useNavigate, useLocation, NavLink } from 'react-router-dom'
import { Header } from './Header'
import { MobileNav } from './MobileNav'
import { useKeyboard } from '../../hooks/useKeyboard'
import { useBreakpoint } from '../../hooks/useBreakpoint'
import { Home, ArrowRightLeft, Target, Settings as SettingsIcon, Wallet, GraduationCap } from 'lucide-react'

export function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { isDesktop } = useBreakpoint()
  const [animKey, setAnimKey] = useState(0)

  useEffect(() => { setAnimKey(prev => prev + 1) }, [location.pathname])

  const navigateTo = useCallback((path: string) => () => navigate(path), [navigate])
  useKeyboard({
    'g': navigateTo('/goals'), 'o': navigateTo('/operations'), 'b': navigateTo('/budgets'),
    'd': navigateTo('/dashboard'), 'n': navigateTo('/operations/new'), 'h': navigateTo('/dashboard'),
    's': navigateTo('/settings'), '1': navigateTo('/dashboard'), '2': navigateTo('/operations'),
    '3': navigateTo('/goals'), '4': navigateTo('/budgets'),
  })

  const sidebarItems = [
    { to: '/dashboard', icon: Home, label: 'Главная' },
    { to: '/operations', icon: ArrowRightLeft, label: 'Операции' },
    { to: '/goals', icon: Target, label: 'Цели' },
    { to: '/budgets', icon: GraduationCap, label: 'Бюджеты' },
    { to: '/accounts', icon: Wallet, label: 'Счета' },
    { to: '/settings', icon: SettingsIcon, label: 'Настройки' },
  ]

  return (
    <div className="flex min-h-screen" style={{background: 'var(--bg-page)'}}>
      {isDesktop && (
        <aside className="w-60 border-r flex flex-col fixed inset-y-0 left-0 z-30"
          style={{background: 'var(--bg-surface)', borderColor: 'var(--border-default)'}}>
          <div className="flex items-center gap-2.5 px-5 h-14 border-b" style={{borderColor: 'var(--border-default)'}}>
            <div className="w-8 h-8 rounded-lg bg-gradient-primary flex items-center justify-center text-white font-bold text-sm shadow-md">₽</div>
            <span className="font-semibold text-base" style={{color: 'var(--text-primary)'}}>FinHelper</span>
          </div>
          <nav className="flex-1 p-3 space-y-1">
            {sidebarItems.map(({ to, icon: Icon, label }) => (
              <NavLink key={to} to={to} end={to === '/dashboard'}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 ${
                    isActive ? 'shadow-sm' : 'hover:bg-gray-100/50'
                  }`
                }
                style={({ isActive }) => ({
                  color: isActive ? 'var(--color-primary-700)' : 'var(--text-secondary)',
                  background: isActive ? 'var(--color-primary-50)' : 'transparent',
                })}
              >
                <Icon size={18} /> {label}
              </NavLink>
            ))}
          </nav>
          <div className="p-4 border-t text-[10px] text-center" style={{borderColor: 'var(--border-default)', color: 'var(--text-tertiary)'}}>
            FinHelper v1.0
          </div>
        </aside>
      )}

      <div className={`flex flex-col flex-1 ${isDesktop ? 'ml-60' : ''}`}>
        <Header />
        <main className="flex-1 px-4 pt-4 pb-24 max-w-2xl mx-auto w-full">
          <div key={animKey} className="animate-page-in">
            <Outlet />
          </div>
        </main>
        {!isDesktop && <MobileNav />}
      </div>
    </div>
  )
}
