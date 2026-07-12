import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  Wallet, GraduationCap, Settings as SettingsIcon, BookOpen,
  PiggyBank, CreditCard, Scale, Home, FileText,
} from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useSettings } from '../hooks/useSettings'

const sections = [
  {
    title: 'Основное',
    items: [
      { to: '/accounts', icon: Wallet, label: 'Счета', desc: 'Управление счетами' },
      { to: '/budgets', icon: GraduationCap, label: 'Бюджеты', desc: 'Лимиты по категориям' },
    ],
  },
  {
    title: 'Калькуляторы',
    items: [
      { to: '/deposit', icon: PiggyBank, label: 'Депозит', desc: 'Доход по вкладу' },
      { to: '/credit', icon: CreditCard, label: 'Кредит', desc: 'ПСК и переплата' },
      { to: '/affordability', icon: Scale, label: 'Доступность', desc: 'Кредитная нагрузка' },
      { to: '/mortgage-rent', icon: Home, label: 'Ипотека vs Аренда', desc: 'Сравнение расходов' },
    ],
  },
  {
    title: 'Прочее',
    items: [
      { to: '/settings', icon: SettingsIcon, label: 'Настройки', desc: 'Тема, валюта, видимость' },
      { to: '/onboarding', icon: BookOpen, label: 'Обучение', desc: 'Как пользоваться FinHelper' },
      { to: '/import', icon: FileText, label: 'Импорт выписок', desc: 'Сбербанк, CSV, PDF' },
    ],
  },
]

const cardVariant = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.25, ease: 'easeOut' as const } },
}

export function MorePage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { initials, name, email } = useSettings()

  return (
    <motion.div className="space-y-4" initial="hidden" animate="show"
      variants={{ show: { transition: { staggerChildren: 0.04 } } }}>

      {/* User card */}
      <motion.div variants={cardVariant}>
        <div className="rounded-[20px] border p-5 relative overflow-hidden"
          style={{
            background: 'linear-gradient(135deg, #141A2D, #1E293B)',
            borderColor: 'rgba(255,255,255,0.06)',
          }}>
          <div className="relative flex items-center gap-4">
            <div className="w-14 h-14 rounded-full flex items-center justify-center text-white text-lg font-bold shrink-0"
              style={{
                background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
                boxShadow: '0 0 24px rgba(110,86,207,0.3)',
              }}>
              {initials}
            </div>
            <div>
              <p className="text-base font-semibold">{name}</p>
              <p className="text-xs" style={{ color: '#94A3B8' }}>{user?.email || email}</p>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Sections */}
      {sections.map((section, si) => (
        <motion.div key={section.title} variants={cardVariant}>
          <div className="rounded-[20px] border overflow-hidden"
            style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
            <p className="px-4 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-wider"
              style={{ color: '#64748B' }}>
              {section.title}
            </p>
            {section.items.map((item, ii) => {
              const Icon = item.icon
              const isLast = ii === section.items.length - 1
              return (
                <button key={item.to} onClick={() => navigate(item.to)}
                  className="flex items-center gap-4 w-full px-4 py-3.5 text-left transition-all btn-press card-hover"
                  style={{
                    borderBottom: isLast ? 'none' : '1px solid rgba(255,255,255,0.04)',
                  }}>
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
                    style={{ background: 'rgba(110,86,207,0.1)', color: '#A78BFA' }}>
                    <Icon size={20} strokeWidth={1.5} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                      {item.label}
                    </p>
                    <p className="text-xs truncate" style={{ color: '#64748B' }}>{item.desc}</p>
                  </div>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
                    style={{ color: '#64748B', flexShrink: 0 }}>
                    <path d="M9 18l6-6-6-6" />
                  </svg>
                </button>
              )
            })}
          </div>
        </motion.div>
      ))}

      <motion.div variants={cardVariant}>
        <p className="text-xs text-center py-4" style={{ color: '#64748B' }}>
          FinHelper v1.0.0 • Приватный финансовый трекер
        </p>
      </motion.div>
    </motion.div>
  )
}
