import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { Home, ArrowRightLeft, Target, Calculator, Grid3X3, Plus } from 'lucide-react'
import { MoreMenu } from './MoreMenu'
import { CalculatorsMenu } from './CalculatorsMenu'

export function MobileNav() {
  const [showMore, setShowMore] = useState(false)
  const [showCalculators, setShowCalculators] = useState(false)
  const navigate = useNavigate()

  const navItems = [
    { to: '/dashboard', icon: Home, label: 'Главная' },
    { to: '/operations', icon: ArrowRightLeft, label: 'Операции' },
    { to: '/goals', icon: Target, label: 'Цели' },
  ]

  return (
    <>
      <nav className="fixed bottom-0 left-0 right-0 z-40 pb-safe"
        style={{
          background: 'var(--glass-bg)',
          backdropFilter: 'blur(20px)',
          WebkitBackdropFilter: 'blur(20px)',
          borderTop: '1px solid var(--glass-border)',
        }}>
        <div className="max-w-2xl mx-auto px-2 h-16 flex items-center justify-around">

          {/* Left: 2 nav items */}
          {navItems.slice(0, 2).map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-all duration-200 ${isActive ? '' : ''}`
              }
              style={({ isActive }) => ({
                color: isActive ? 'var(--color-primary-600)' : 'var(--text-tertiary)',
                background: isActive ? 'var(--color-primary-50)' : 'transparent',
              })}
            >
              <div className="relative">
                <Icon size={22} />
                {/* Active indicator handled by NavLink style */}
              </div>
              <span className="text-[10px] font-medium">{label}</span>
            </NavLink>
          ))}

          {/* Center: FAB */}
          <button
            onClick={() => navigate('/operations/new')}
            className="relative -top-6 w-14 h-14 rounded-full bg-gradient-primary text-white shadow-2xl flex items-center justify-center hover:shadow-xl transition-all duration-200 btn-press"
            style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.2)' }}
          >
            <Plus size={26} />
          </button>

          {/* Right: 2 nav items */}
          {navItems.slice(2).map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-all duration-200 ${isActive ? '' : ''}`
              }
              style={({ isActive }) => ({
                color: isActive ? 'var(--color-primary-600)' : 'var(--text-tertiary)',
                background: isActive ? 'var(--color-primary-50)' : 'transparent',
              })}
            >
              <div className="relative">
                <Icon size={22} />
              </div>
              <span className="text-[10px] font-medium">{label}</span>
            </NavLink>
          ))}

          {/* Calculators button */}
          <button
            onClick={() => setShowCalculators(true)}
            className="flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-colors btn-press"
            style={{ color: 'var(--text-tertiary)' }}
          >
            <Calculator size={22} />
            <span className="text-[10px] font-medium">Калькуляторы</span>
          </button>

          {/* More button */}
          <button
            onClick={() => setShowMore(true)}
            className="flex flex-col items-center gap-0.5 px-3 py-1.5 rounded-xl transition-colors btn-press"
            style={{ color: 'var(--text-tertiary)' }}
          >
            <Grid3X3 size={22} />
            <span className="text-[10px] font-medium">Ещё</span>
          </button>
        </div>
      </nav>

      {showMore && <MoreMenu onClose={() => setShowMore(false)} />}
      {showCalculators && <CalculatorsMenu onClose={() => setShowCalculators(false)} />}
    </>
  )
}