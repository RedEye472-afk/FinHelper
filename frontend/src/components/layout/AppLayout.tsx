import { useCallback, useEffect, useState } from 'react'
import { Outlet, useNavigate, useLocation, NavLink } from 'react-router-dom'
import { Header } from './Header'
import { MobileNav } from './MobileNav'
import { useKeyboard } from '../../hooks/useKeyboard'
import { useBreakpoint } from '../../hooks/useBreakpoint'
import {
  LayoutDashboard, ArrowRightLeft, Target, Settings as SettingsIcon,
  Wallet, Calculator, CreditCard, PiggyBank, Eye, EyeOff,
  FileText, Landmark, Grid3X3,
} from 'lucide-react'
import { useSettings } from '../../hooks/useSettings'

const sidebarItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Обзор' },
  { to: '/operations', icon: ArrowRightLeft, label: 'Операции' },
  { to: '/goals', icon: Target, label: 'Цели' },
  { to: '/budgets', icon: Wallet, label: 'Бюджеты' },
  { to: '/accounts', icon: Landmark, label: 'Счета' },
]

const calcItems = [
  { to: '/deposit', icon: PiggyBank, label: 'Депозит', desc: 'Доходность вклада' },
  { to: '/credit', icon: CreditCard, label: 'Кредит', desc: 'График платежей, ПСК' },
  { to: '/affordability', icon: Calculator, label: 'Доступность', desc: 'Посильность кредита' },
  { to: '/mortgage-rent', icon: Calculator, label: 'Ипотека vs Аренда', desc: 'Что выгоднее' },
]

function DesktopSidebar() {
  const { hideBalance, setSettings } = useSettings()

  return (
    <aside
      className="w-60 border-r flex flex-col fixed inset-y-0 left-0 z-30"
      style={{ background: '#0B0F1A', borderColor: 'rgba(255,255,255,0.06)' }}
    >
      {/* Logo */}
      <div className="flex items-center gap-2.5 px-5 h-14 border-b shrink-0"
        style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
        <div
          className="w-8 h-8 rounded-[10px] flex items-center justify-center text-white font-bold text-sm"
          style={{
            background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
            boxShadow: '0 0 20px rgba(110,86,207,0.3)',
          }}
        >
          F
        </div>
        <span className="font-semibold text-base" style={{ color: '#FFFFFF' }}>
          Fin<span style={{ color: '#A78BFA' }}>Helper</span>
        </span>
      </div>

      {/* Nav */}
      <nav className="flex-1 p-3 space-y-0.5 overflow-y-auto">
        <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider"
          style={{ color: '#64748B' }}>
          Навигация
        </p>
        {sidebarItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/dashboard'}
            className="flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition-all duration-200 rounded-[10px]"
            style={({ isActive }) => ({
              color: isActive ? '#FFFFFF' : '#94A3B8',
              background: isActive ? 'rgba(110,86,207,0.12)' : 'transparent',
            })}
          >
            <Icon size={20} strokeWidth={1.5} />
            {label}
          </NavLink>
        ))}

        {/* Calculators */}
        <div className="pt-4 mt-2 border-t" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#64748B' }}>
            Калькуляторы
          </p>
          <div className="space-y-0.5">
            {calcItems.map(({ to, icon: Icon, label, desc }) => (
              <NavLink
                key={to}
                to={to}
                className="flex items-center gap-3 px-3 py-2 rounded-[10px] text-sm font-medium transition-all duration-200 group"
                style={{ color: '#94A3B8' }}
              >
                <div
                  className="w-7 h-7 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform"
                  style={{ background: 'rgba(110,86,207,0.12)', color: '#A78BFA' }}
                >
                  <Icon size={15} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{label}</p>
                  <p className="text-[10px] truncate" style={{ color: '#64748B' }}>{desc}</p>
                </div>
              </NavLink>
            ))}
          </div>
        </div>

        {/* Ещё */}
        <div className="pt-4 mt-2 border-t" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider" style={{ color: '#64748B' }}>
            Ещё
          </p>
          <NavLink to="/more"
            className="flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition-all duration-200 rounded-[10px]"
            style={{ color: '#94A3B8' }}>
            <Grid3X3 size={18} strokeWidth={1.5} />
            Все разделы
          </NavLink>
        </div>
      </nav>

      {/* Bottom */}
      <div className="p-3 border-t space-y-2" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
        <button
          onClick={() => setSettings({ hideBalance: !hideBalance })}
          className="flex items-center gap-3 w-full px-3 py-2.5 rounded-[10px] text-sm font-medium transition-all duration-200"
          style={{ color: '#64748B' }}
        >
          {hideBalance ? <Eye size={18} strokeWidth={1.5} /> : <EyeOff size={18} strokeWidth={1.5} />}
          <span>{hideBalance ? 'Показать суммы' : 'Скрыть суммы'}</span>
        </button>
        <NavLink
          to="/settings"
          className="flex items-center gap-3 px-3 py-2.5 rounded-[10px] text-sm font-medium transition-all duration-200"
          style={{ color: '#64748B' }}
        >
          <SettingsIcon size={18} strokeWidth={1.5} />
          Настройки
        </NavLink>
      </div>
    </aside>
  )
}

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

  return (
    <div className="flex min-h-screen" style={{ background: 'var(--bg-page)' }}>
      {isDesktop && <DesktopSidebar />}

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
