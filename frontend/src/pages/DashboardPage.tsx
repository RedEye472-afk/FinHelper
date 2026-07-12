/**
 * DashboardPage — banking home screen (T-Bank style)
 * Balance → Accounts → Month stats → Recent ops → Quick actions
 * Premium Dark Neon, real API, no floats.
 */
import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  Plus, Eye, EyeOff, TrendingUp, TrendingDown,
  Wallet, PiggyBank, CreditCard, Landmark, Target,
  ShoppingBag, Car, Film, ShoppingCart,
} from 'lucide-react'
import { useFinance } from '../store'
import { useSettings } from '../hooks/useSettings'
import { useAccounts, useOperations, useGoals } from '../api/queries'
import { formatMoney, formatCompact, toDecimal, M } from '../lib/money'
import { Skeleton } from '../components/ui/Skeleton'

const cardVariant = {
  hidden: { opacity: 0, y: 16 },
  show: { opacity: 1, y: 0, transition: { duration: 0.3, ease: 'easeOut' as const } },
}

const accountIcons: Record<string, typeof Wallet> = {
  cash: Wallet, bank: Landmark, savings: PiggyBank, investment: TrendingUp, crypto: CreditCard, debt: CreditCard,
}

export function DashboardPage() {
  const navigate = useNavigate()
  const { kpi, expensesByCategory, isLoading } = useFinance()
  const { hideBalance, setSettings } = useSettings()
  const { data: accountsData } = useAccounts()
  const { data: opsData } = useOperations(10)
  const { data: goalsData } = useGoals()

  const accounts = accountsData ?? []
  const recentOps = opsData?.items ?? []
  const goals = goalsData ?? []
  const activeGoals = goals.filter(g => toDecimal(g.current_amount).lt(toDecimal(g.target_amount)))

  const totalSpent = useMemo(
    () => expensesByCategory.reduce((s, e) => s.plus(e.amount), M.zero()),
    [expensesByCategory],
  )
  const monthTxCount = recentOps.filter(op => {
    const d = new Date(op.operation_date)
    const now = new Date()
    return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear()
  }).length
  const hasData = kpi.totalBalance.gt(0) || accounts.length > 0

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-44 rounded-[20px]" />
        <div className="flex gap-3 overflow-hidden"><Skeleton className="h-28 w-40 rounded-[16px] shrink-0" /><Skeleton className="h-28 w-40 rounded-[16px] shrink-0" /></div>
        <Skeleton className="h-52 rounded-[20px]" />
      </div>
    )
  }

  return (
    <motion.div className="space-y-4" initial="hidden" animate="show" variants={{ show: { transition: { staggerChildren: 0.06 } } }}>

      {!hasData ? (
        <motion.div variants={cardVariant}>
          <div className="rounded-[20px] border p-8 text-center"
            style={{
              background: 'linear-gradient(135deg, #141A2D, #1E293B)',
              borderColor: 'rgba(255,255,255,0.06)',
            }}>
            <div className="w-16 h-16 rounded-2xl flex items-center justify-center mx-auto mb-4"
              style={{ background: 'rgba(110,86,207,0.15)' }}>
              <Wallet size={32} style={{ color: '#A78BFA' }} />
            </div>
            <h2 className="text-xl font-bold mb-1">Добро пожаловать</h2>
            <p className="text-sm mb-5" style={{ color: 'var(--text-secondary)' }}>
              Добавьте первую операцию и начнём
            </p>
            <button onClick={() => navigate('/operations/new')}
              className="w-full py-3 rounded-xl font-semibold text-sm flex items-center justify-center gap-2"
              style={{ background: 'linear-gradient(135deg, #6E56CF, #A78BFA)', color: '#fff' }}>
              <Plus size={18} /> Первая операция
            </button>
          </div>
        </motion.div>
      ) : (
        <>
          {/* ═══ 1. BALANCE ═══ */}
          <motion.div variants={cardVariant}>
            <div className="relative overflow-hidden rounded-[20px] border p-5"
              style={{
                background: 'linear-gradient(135deg, #141A2D, #1E293B)',
                borderColor: 'rgba(255,255,255,0.06)',
                boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
              }}
            >
              <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1)' }} />
              <div className="relative">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-xs font-medium" style={{ color: '#94A3B8' }}>Общий баланс</p>
                  <button onClick={() => setSettings({ hideBalance: !hideBalance })}
                    className="p-1.5 rounded-lg" style={{ color: '#64748B' }}>
                    {hideBalance ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
                <p className="text-3xl font-bold font-mono-money tracking-tight mb-3">
                  {hideBalance ? '··· ···' : formatMoney(kpi.totalBalance)}
                </p>
                <div className="flex gap-3">
                  <div className="flex-1 rounded-xl px-3.5 py-2.5"
                    style={{ background: 'rgba(255,255,255,0.06)', backdropFilter: 'blur(12px)' }}>
                    <div className="flex items-center gap-1 text-xs mb-0.5" style={{ color: '#94A3B8' }}>
                      <TrendingUp size={12} style={{ color: '#22C55E' }} /> Доходы
                    </div>
                    <p className="text-sm font-semibold font-mono-money" style={{ color: '#22C55E' }}>
                      +{hideBalance ? '···' : formatMoney(kpi.monthIncome)}
                    </p>
                  </div>
                  <div className="flex-1 rounded-xl px-3.5 py-2.5"
                    style={{ background: 'rgba(255,255,255,0.06)', backdropFilter: 'blur(12px)' }}>
                    <div className="flex items-center gap-1 text-xs mb-0.5" style={{ color: '#94A3B8' }}>
                      <TrendingDown size={12} style={{ color: '#F43F5E' }} /> Расходы
                    </div>
                    <p className="text-sm font-semibold font-mono-money" style={{ color: '#F43F5E' }}>
                      −{hideBalance ? '···' : formatMoney(kpi.monthExpense)}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </motion.div>

          {/* ═══ 2. ACCOUNTS ═══ */}
          <motion.div variants={cardVariant}>
            <div className="flex items-center justify-between mb-2 px-1">
              <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Мои счета</p>
              <button onClick={() => navigate('/accounts')}
                className="text-xs font-medium" style={{ color: '#A78BFA' }}>Все</button>
            </div>
            <div className="flex gap-3 overflow-x-auto scrollbar-hide -mx-4 px-4">
              {accounts.length === 0 ? (
                <div className="flex-1 rounded-[16px] border p-4 text-center"
                  style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                  <p className="text-xs" style={{ color: '#64748B' }}>Нет счетов</p>
                </div>
              ) : accounts.map(acc => {
                const Icon = accountIcons[acc.account_type] || Wallet
                return (
                  <div key={acc.id} className="shrink-0 w-44 rounded-[16px] border p-4 card-hover"
                    style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                    <div className="flex items-center gap-2 mb-3">
                      <div className="w-8 h-8 rounded-lg flex items-center justify-center"
                        style={{ background: 'rgba(110,86,207,0.12)', color: '#A78BFA' }}>
                        <Icon size={16} />
                      </div>
                      <span className="text-xs font-medium truncate" style={{ color: '#94A3B8' }}>{acc.name}</span>
                    </div>
                    <p className="text-base font-bold font-mono-money tracking-tight">
                      {hideBalance ? '···' : formatCompact(toDecimal(acc.balance))}
                    </p>
                  </div>
                )
              })}
            </div>
          </motion.div>

          {/* ═══ 3. MONTH STATS ═══ */}
          <motion.div variants={cardVariant}>
            <div className="grid grid-cols-3 gap-3">
              <div className="rounded-[16px] border p-3.5 text-center"
                style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                <p className="text-lg font-bold font-mono-money" style={{ color: '#22C55E' }}>
                  {hideBalance ? '···' : formatCompact(kpi.monthIncome)}
                </p>
                <p className="text-[10px] mt-0.5" style={{ color: '#64748B' }}>Доход</p>
              </div>
              <div className="rounded-[16px] border p-3.5 text-center"
                style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                <p className="text-lg font-bold font-mono-money" style={{ color: '#F43F5E' }}>
                  {hideBalance ? '···' : formatCompact(totalSpent)}
                </p>
                <p className="text-[10px] mt-0.5" style={{ color: '#64748B' }}>Расход</p>
              </div>
              <div className="rounded-[16px] border p-3.5 text-center"
                style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                <p className="text-lg font-bold font-mono-money" style={{ color: '#A78BFA' }}>
                  {monthTxCount}
                </p>
                <p className="text-[10px] mt-0.5" style={{ color: '#64748B' }}>Операций</p>
              </div>
            </div>
          </motion.div>

          {/* ═══ 4. RECENT OPERATIONS ═══ */}
          <motion.div variants={cardVariant}>
            <div className="rounded-[20px] border p-4"
              style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
              <div className="flex items-center justify-between mb-3">
                <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Последние операции</p>
                <button onClick={() => navigate('/operations')}
                  className="text-xs font-medium" style={{ color: '#A78BFA' }}>Все</button>
              </div>
              {recentOps.length === 0 ? (
                <div className="py-6 text-center">
                  <p className="text-sm" style={{ color: '#64748B' }}>Нет операций</p>
                </div>
              ) : (
                <div>
                  {recentOps.slice(0, 5).map((op, idx) => {
                    const amount = toDecimal(op.amount)
                    const isIncome = op.type === 'income'
                    const OpIcon = [ShoppingBag, Car, Film, ShoppingCart, Target][idx % 5]
                    return (
                      <div key={op.id}
                        className="flex items-center gap-3 py-2.5"
                        style={{ borderBottom: idx < 4 ? '1px solid rgba(255,255,255,0.04)' : 'none' }}>
                        <div className="w-9 h-9 rounded-[10px] flex items-center justify-center shrink-0"
                          style={{ background: isIncome ? 'rgba(34,197,94,0.1)' : 'rgba(244,63,94,0.1)' }}>
                          {isIncome ? <TrendingUp size={16} style={{ color: '#22C55E' }} /> : <OpIcon size={16} style={{ color: '#F43F5E' }} />}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">{op.description || 'Без описания'}</p>
                          <p className="text-xs" style={{ color: '#64748B' }}>
                            {new Date(op.operation_date).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })}
                          </p>
                        </div>
                        <span className="font-mono-money font-semibold text-sm shrink-0"
                          style={{ color: isIncome ? '#22C55E' : '#F43F5E' }}>
                          {isIncome ? '+' : '−'}{hideBalance ? '···' : formatCompact(amount)}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}
              <button onClick={() => navigate('/operations/new')}
                className="w-full mt-2 py-2.5 rounded-xl text-sm font-medium flex items-center justify-center gap-1.5"
                style={{ color: '#A78BFA' }}>
                <Plus size={16} /> Добавить операцию
              </button>
            </div>
          </motion.div>

          {/* ═══ 5. QUICK ACTIONS ═══ */}
          <motion.div variants={cardVariant}>
            <div className="grid grid-cols-4 gap-2">
              {[
                { icon: Plus, label: 'Пополнить', href: '/operations/new', color: '#6E56CF' },
                { icon: TrendingUp, label: 'Перевести', href: '/operations/new', color: '#22C55E' },
                { icon: Target, label: 'Цели', href: '/goals', color: '#3B82F6' },
                { icon: Wallet, label: 'Счета', href: '/accounts', color: '#F59E0B' },
              ].map(({ icon: Icon, label, href, color }) => (
                <button key={label} onClick={() => navigate(href)}
                  className="flex flex-col items-center gap-2 py-4 rounded-[16px] border card-hover"
                  style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center"
                    style={{ background: `${color}18`, color }}>
                    <Icon size={18} strokeWidth={1.75} />
                  </div>
                  <span className="text-[11px] font-medium leading-tight text-center" style={{ color: '#94A3B8' }}>
                    {label}
                  </span>
                </button>
              ))}
            </div>
          </motion.div>
        </>
      )}
    </motion.div>
  )
}
