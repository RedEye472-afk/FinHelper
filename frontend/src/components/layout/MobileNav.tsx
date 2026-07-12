import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { LayoutDashboard, ArrowRightLeft, Target, Grid3X3, Plus } from 'lucide-react'
import { MoreMenu } from './MoreMenu'

export function MobileNav() {
  const [showMore, setShowMore] = useState(false)
  const navigate = useNavigate()

  const tabs = [
    { to: '/dashboard', icon: LayoutDashboard, label: 'Главная' },
    { to: '/operations', icon: ArrowRightLeft, label: 'Операции' },
    { to: '/goals', icon: Target, label: 'Цели' },
  ]

  return (
    <>
      <nav
        className="fixed bottom-0 left-0 right-0 z-40 pb-safe border-t md:hidden"
        style={{
          background: 'rgba(11,15,26,0.92)',
          backdropFilter: 'blur(20px) saturate(150%)',
          WebkitBackdropFilter: 'blur(20px) saturate(150%)',
          borderColor: 'rgba(255,255,255,0.06)',
        }}
      >
        <div className="max-w-lg mx-auto px-2 h-[68px] flex items-center justify-around">
          {tabs.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/dashboard'}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-all duration-200 relative ${
                  isActive ? 'active-tab' : ''
                }`
              }
              style={({ isActive }) => ({
                color: isActive ? 'var(--color-primary-400)' : 'var(--text-tertiary)',
              })}
            >
              <Icon size={22} strokeWidth={1.5} />
              <span className="text-[10px] font-medium">{label}</span>
            </NavLink>
          ))}

          {/* FAB */}
          <button
            onClick={() => navigate('/operations/new')}
            className="relative -top-5 w-14 h-14 rounded-full flex items-center justify-center btn-press transition-transform hover:scale-105 active:scale-95"
            style={{
              background: 'linear-gradient(135deg, #6E56CF 0%, #A78BFA 100%)',
              boxShadow: '0 8px 24px rgba(110,86,207,0.45)',
            }}
          >
            <Plus size={26} className="text-white" />
          </button>

          <button
            onClick={() => setShowMore(true)}
            className="flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-colors"
            style={{ color: 'var(--text-tertiary)' }}
          >
            <Grid3X3 size={22} strokeWidth={1.5} />
            <span className="text-[10px] font-medium">Ещё</span>
          </button>
        </div>
      </nav>

      {/* Desktop bottom nav — wider layout */}
      <nav
        className="fixed bottom-0 left-0 right-0 z-40 pb-safe border-t hidden md:flex"
        style={{
          background: 'rgba(11,15,26,0.92)',
          backdropFilter: 'blur(20px) saturate(150%)',
          WebkitBackdropFilter: 'blur(20px) saturate(150%)',
          borderColor: 'rgba(255,255,255,0.06)',
        }}
      >
        <div className="max-w-3xl mx-auto px-4 h-[68px] flex items-center justify-around w-full">
          {tabs.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/dashboard'}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-6 py-1.5 rounded-xl transition-all duration-200 ${
                  isActive ? 'active-tab' : ''
                }`
              }
              style={({ isActive }) => ({
                color: isActive ? 'var(--color-primary-400)' : 'var(--text-tertiary)',
              })}
            >
              <Icon size={22} strokeWidth={1.5} />
              <span className="text-xs font-medium">{label}</span>
            </NavLink>
          ))}
          <button
            onClick={() => navigate('/operations/new')}
            className="flex items-center gap-2 px-5 py-2.5 rounded-full font-medium text-sm btn-press"
            style={{
              background: 'linear-gradient(135deg, #6E56CF 0%, #A78BFA 100%)',
              color: '#fff',
              boxShadow: '0 4px 16px rgba(110,86,207,0.35)',
            }}
          >
            <Plus size={18} /> <span>Операция</span>
          </button>
          <button
            onClick={() => setShowMore(true)}
            className="flex flex-col items-center gap-0.5 px-6 py-1.5 rounded-xl transition-colors"
            style={{ color: 'var(--text-tertiary)' }}
          >
            <Grid3X3 size={22} strokeWidth={1.5} />
            <span className="text-xs font-medium">Ещё</span>
          </button>
        </div>
      </nav>

      {showMore && <MoreMenu onClose={() => setShowMore(false)} />}
    </>
  )
}
