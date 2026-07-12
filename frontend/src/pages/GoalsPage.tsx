import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Plus, BarChart3, PiggyBank, X, Loader2, Target, ArrowDown, ArrowUp, TrendingUp, Calendar, Clock } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { BottomSheet } from '../components/ui/BottomSheet'
import { Modal } from '../components/ui/Modal'
import { useGoals, useCreateGoal, useContributeGoal, useGoalProjection } from '../api/queries'
import { useFinance } from '../store'
import { formatMoney, toDecimal, M } from '../lib/money'
import type { Decimal } from '../lib/money'
import { useSettings } from '../hooks/useSettings'
import type { Goal } from '../types'

/* ───────────────────────────────────────────
   Premium Dark Neon CSS Variables (inline)
   ─────────────────────────────────────────── */
const vars = {
  bgPage: '#0B1020',
  surface: '#141A2D',
  elevated: '#1B2340',
  primary: '#6E56CF',
  primaryLight: '#8B5CF6',
  primaryDark: '#5B21B6',
  primaryGlow: 'rgba(110,86,207,0.35)',
  accent: '#A78BFA',
  textPrimary: '#FFFFFF',
  textSecondary: '#94A3B8',
  textTertiary: '#64748B',
  border: 'rgba(255,255,255,0.06)',
  success: '#10B981',
  warning: '#F59E0B',
  danger: '#F43F5E',
} as const

const goalIcons = ['🛡', '💻', '✈', '📈', '🏠', '🚗', '🎓', '💍', '🏥', '🎮', '📱', '🎸', '🐱', '🌴', '🏋', '🎯']

/** Deterministic icon from goal id/name so server goals (no icon field) render consistently. */
function pickIcon(goal: Goal): string {
  const seed = goal.name + goal.id
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return goalIcons[h % goalIcons.length]
}

/** Local MoneyDisplay replacement for string-based amounts (server returns strings). */
function MoneySpan({ amount, size = 'md', className = '' }: { amount: string; size?: 'sm' | 'md' | 'lg'; className?: string }) {
  const { hideBalance } = useSettings()
  if (hideBalance) return <span className={`font-mono-money ${className}`} style={{ color: vars.textTertiary }}>••••</span>
  const sizes = { sm: 'text-sm', md: 'text-base', lg: 'text-xl' }
  return <span className={`font-mono-money font-semibold ${sizes[size]} ${className}`} style={{ color: vars.textPrimary }}>{formatMoney(amount)}</span>
}

/** Circular progress ring SVG with neon glow. */
function ProgressRing({ pct, size = 44, stroke = 4 }: { pct: number; size?: number; stroke?: number }) {
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const clamped = Math.min(Math.max(pct, 0), 100)
  const offset = c - (clamped / 100) * c

  const ringColor = pct >= 100 ? vars.success : pct >= 50 ? vars.primaryLight : vars.warning
  const glowId = `glow-${Math.random().toString(36).slice(2, 8)}`

  return (
    <svg width={size} height={size} className="flex-shrink-0 -rotate-90">
      <defs>
        <filter id={glowId}>
          <feGaussianBlur stdDeviation="2" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" strokeWidth={stroke} style={{ stroke: 'rgba(255,255,255,0.06)' }} />
      <circle
        cx={size / 2} cy={size / 2} r={r} fill="none" strokeWidth={stroke}
        strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round"
        filter={`url(#${glowId})`}
        style={{
          stroke: ringColor,
          transition: 'stroke-dashoffset 0.6s cubic-bezier(0.4, 0, 0.2, 1), stroke 0.3s ease',
        }}
      />
    </svg>
  )
}

/* ── Page component ── */
export function GoalsPage() {
  const { kpi } = useFinance()
  const { data: goals, isLoading: goalsLoading } = useGoals()
  const createGoal = useCreateGoal()
  const contribute = useContributeGoal()

  const [showAdd, setShowAdd] = useState(false)
  const [name, setName] = useState('')
  const [target, setTarget] = useState('')
  const [deadline, setDeadline] = useState('')
  const [icon, setIcon] = useState('🎯')
  const [depositGoal, setDepositGoal] = useState<Goal | null>(null)
  const [depositAmount, setDepositAmount] = useState('')
  const [withdrawGoal, setWithdrawGoal] = useState<Goal | null>(null)
  const [withdrawAmount, setWithdrawAmount] = useState('')
  const [simulateGoal, setSimulateGoal] = useState<Goal | null>(null)

  const reset = () => { setName(''); setTarget(''); setDeadline(''); setIcon('🎯') }

  const handleSave = async () => {
    const tDec = M.fromInput(target)
    if (!name || !M.isPositive(tDec)) return
    if (createGoal.isPending) return
    try {
      await createGoal.mutateAsync({
        name,
        target_amount: tDec.toFixed(2),
        target_date: deadline || new Date(Date.now() + 90 * 86400000).toISOString().slice(0, 10),
      })
      setShowAdd(false); reset()
    } catch (e) {
      console.error('createGoal failed', e)
    }
  }

  const handleDeposit = async () => {
    if (!depositGoal) return
    const amt = M.fromInput(depositAmount)
    if (!M.isPositive(amt)) return
    if (contribute.isPending) return
    try {
      await contribute.mutateAsync({ goalId: depositGoal.id, data: { amount: amt.toFixed(2), contribution_id: crypto.randomUUID() } })
      setDepositGoal(null); setDepositAmount('')
    } catch (e) {
      console.error('contribute failed', e)
    }
  }

  const handleWithdraw = async () => {
    if (!withdrawGoal) return
    const amtDec = M.fromInput(withdrawAmount)
    if (!M.isPositive(amtDec)) return
    const current = toDecimal(withdrawGoal.current_amount)
    if (amtDec.gt(current)) return
    if (contribute.isPending) return
    try {
      await contribute.mutateAsync({ goalId: withdrawGoal.id, data: { amount: M.neg(amtDec).toFixed(2), contribution_id: crypto.randomUUID() } })
      setWithdrawGoal(null); setWithdrawAmount('')
    } catch (e) {
      console.error('withdraw failed', e)
    }
  }

  const totalProgress = useMemo(() => {
    const list = goals ?? []
    if (list.length === 0) return 0
    const totalCurrent = list.reduce((acc: Decimal, g) => acc.plus(toDecimal(g.current_amount)), M.zero())
    const totalTarget = list.reduce((acc: Decimal, g) => acc.plus(toDecimal(g.target_amount)), M.zero())
    return totalTarget.gt(0) ? Number(totalCurrent.div(totalTarget).mul(100).toFixed(2)) : 0
  }, [goals])

  const list = goals ?? []
  const isBusy = createGoal.isPending

  return (
    <div className="space-y-5 pb-24">

      {/* ── Premium Header with gradient + SVG ring ── */}
      <div
        className="relative rounded-2xl p-6 overflow-hidden"
        style={{
          background: 'linear-gradient(135deg, #6E56CF 0%, #7C3AED 40%, #A78BFA 100%)',
          boxShadow: '0 8px 32px rgba(110,86,207,0.35)',
        }}
      >
        {/* Decorative glow circles */}
        <div
          className="absolute -top-10 -right-10 w-40 h-40 rounded-full opacity-30"
          style={{ background: 'radial-gradient(circle, rgba(167,139,250,0.6) 0%, transparent 70%)' }}
        />
        <div
          className="absolute -bottom-8 -left-8 w-32 h-32 rounded-full opacity-20"
          style={{ background: 'radial-gradient(circle, rgba(110,86,207,0.5) 0%, transparent 70%)' }}
        />

        <div className="relative z-10">
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="text-xs font-medium tracking-wider uppercase opacity-70 mb-1">
                Общий прогресс
              </p>
              <p className="text-4xl font-bold font-mono-money tracking-tight">
                {Math.round(totalProgress)}%
              </p>
            </div>
            <ProgressRing pct={totalProgress} size={64} stroke={5} />
          </div>

          {/* Neon progress bar */}
          <div
            className="relative w-full rounded-full h-2 overflow-hidden"
            style={{ background: 'rgba(255,255,255,0.12)' }}
          >
            <div
              className="h-full rounded-full transition-all duration-700 ease-out"
              style={{
                width: `${Math.min(totalProgress, 100)}%`,
                background: 'linear-gradient(90deg, #A78BFA 0%, #FFFFFF 100%)',
                boxShadow: '0 0 12px rgba(167,139,250,0.6)',
              }}
            />
          </div>

          <div className="flex items-center justify-between mt-3 text-xs text-white/60">
            <span>
              {list.length} {list.length === 1 ? 'цель' : list.length >= 2 && list.length <= 4 ? 'цели' : 'целей'}
            </span>
            {totalProgress > 0 && <span>{list.filter(g => toDecimal(g.current_amount).gte(toDecimal(g.target_amount))).length} выполнено</span>}
          </div>
        </div>
      </div>

      {/* ── Section header ── */}
      <div className="flex items-center justify-between px-0.5">
        <div className="flex items-center gap-2">
          <Target size={16} style={{ color: vars.primaryLight }} />
          <h2 className="text-sm font-semibold" style={{ color: vars.textPrimary }}>
            Мои цели
            {goalsLoading && <Loader2 className="inline ml-1.5 animate-spin" size={12} style={{ color: vars.primaryLight }} />}
          </h2>
        </div>
        <button
          onClick={() => { reset(); setShowAdd(true) }}
          className="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase transition-opacity hover:opacity-80"
          style={{ color: vars.primaryLight }}
        >
          <Plus size={14} />
          Новая
        </button>
      </div>

      {/* ── Goal cards ── */}
      <div className="space-y-3">
        {list.length === 0 && !goalsLoading ? (
          <Card className="py-10 text-center" style={{ border: `1px solid ${vars.border}` }}>
            <div
              className="w-14 h-14 rounded-2xl flex items-center justify-center mx-auto mb-3"
              style={{ background: `rgba(110,86,207,0.1)` }}
            >
              <PiggyBank size={28} style={{ color: vars.primaryLight }} />
            </div>
            <p className="text-sm font-medium" style={{ color: vars.textSecondary }}>Нет целей</p>
            <p className="text-xs mt-1" style={{ color: vars.textTertiary }}>Создайте первую финансовую цель</p>
          </Card>
        ) : (
          <AnimatePresence mode="popLayout">
            {list.map((g, idx) => {
              const current = toDecimal(g.current_amount)
              const targetVal = toDecimal(g.target_amount)
              const pct = targetVal.gt(0) ? Number(current.div(targetVal).mul(100).toFixed(2)) : 0
              const barGradient = pct >= 100
                ? 'linear-gradient(90deg, #10B981, #34D399)'
                : pct >= 50
                  ? 'linear-gradient(90deg, #6E56CF, #8B5CF6)'
                  : 'linear-gradient(90deg, #F59E0B, #FBBF24)'
              const daysLeft = g.target_date ? Math.ceil((new Date(g.target_date).getTime() - Date.now()) / 86400000) : 0
              const isComplete = pct >= 100
              const statusLabel = isComplete ? 'Выполнена' : 'В процессе'
              const icon = pickIcon(g)

              return (
                <motion.div
                  key={g.id}
                  layout
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -12, scale: 0.96 }}
                  transition={{ duration: 0.25, delay: idx * 0.04 }}
                  className="rounded-xl p-5 w-full"
                  style={{
                    background: vars.surface,
                    border: `1px solid ${vars.border}`,
                    boxShadow: '0 4px 24px rgba(0,0,0,0.3)',
                  }}
                >
                  {/* Top row: icon + name + pct badge */}
                  <div className="flex items-start justify-between mb-3.5">
                    <div className="flex items-center gap-3">
                      <div
                        className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
                        style={{
                          background: isComplete
                            ? 'rgba(16,185,129,0.12)'
                            : 'rgba(110,86,207,0.12)',
                        }}
                      >
                        {icon}
                      </div>
                      <div>
                        <p className="text-sm font-semibold" style={{ color: vars.textPrimary }}>{g.name}</p>
                        <div className="flex items-center gap-1.5 mt-0.5">
                          <span
                            className="inline-block w-1.5 h-1.5 rounded-full"
                            style={{ background: isComplete ? vars.success : vars.primaryLight }}
                          />
                          <span className="text-[10px] font-medium" style={{ color: vars.textSecondary }}>
                            {statusLabel}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg font-mono-money font-bold text-sm"
                      style={{
                        background: isComplete
                          ? 'rgba(16,185,129,0.1)'
                          : 'rgba(110,86,207,0.1)',
                        color: isComplete ? vars.success : vars.primaryLight,
                      }}
                    >
                      {isComplete && (
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                          <polyline points="20 6 9 17 4 12" />
                        </svg>
                      )}
                      {Math.round(pct)}%
                    </div>
                  </div>

                  {/* Progress bar */}
                  <div
                    className="w-full rounded-full h-2.5 mb-3 overflow-hidden"
                    style={{ background: 'rgba(255,255,255,0.05)' }}
                  >
                    <div
                      className="h-full rounded-full transition-all duration-700 ease-out"
                      style={{
                        width: `${Math.min(pct, 100)}%`,
                        background: barGradient,
                        boxShadow: pct > 0 ? `0 0 8px ${isComplete ? 'rgba(16,185,129,0.4)' : 'rgba(110,86,207,0.3)'}` : 'none',
                      }}
                    />
                  </div>

                  {/* Amount row */}
                  <div className="flex items-center justify-between text-xs mb-3">
                    <div className="flex items-center gap-1.5">
                      <ArrowUp size={10} style={{ color: vars.success }} />
                      <MoneySpan amount={g.current_amount} size="sm" className="!text-xs" />
                    </div>
                    <span style={{ color: vars.textTertiary }}>
                      из <MoneySpan amount={g.target_amount} size="sm" className="!text-xs" />
                    </span>
                    {daysLeft > 0 && !isComplete && (
                      <div className="flex items-center gap-1" style={{ color: daysLeft <= 30 ? vars.warning : vars.textSecondary }}>
                        <Clock size={10} />
                        <span>{daysLeft} {daysLeft === 1 ? 'день' : daysLeft < 5 ? 'дня' : 'дн.'}</span>
                      </div>
                    )}
                  </div>

                  {/* Action buttons (only for incomplete goals) */}
                  {!isComplete && (
                    <div className="flex gap-2 mt-1">
                      <button
                        onClick={() => { setDepositGoal(g); setDepositAmount('') }}
                        className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-xl font-medium text-xs transition-all duration-200"
                        style={{
                          background: 'rgba(16,185,129,0.1)',
                          color: vars.success,
                          border: '1px solid rgba(16,185,129,0.15)',
                        }}
                        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(16,185,129,0.18)' }}
                        onMouseLeave={e => { e.currentTarget.style.background = 'rgba(16,185,129,0.1)' }}
                      >
                        <ArrowUp size={13} />
                        Внести
                      </button>
                      {current.gt(0) && (
                        <button
                          onClick={() => { setWithdrawGoal(g); setWithdrawAmount('') }}
                          className="flex items-center justify-center gap-1.5 py-2.5 px-4 rounded-xl font-medium text-xs transition-all duration-200"
                          style={{
                            background: 'rgba(245,158,11,0.08)',
                            color: vars.warning,
                            border: '1px solid rgba(245,158,11,0.12)',
                          }}
                          onMouseEnter={e => { e.currentTarget.style.background = 'rgba(245,158,11,0.15)' }}
                          onMouseLeave={e => { e.currentTarget.style.background = 'rgba(245,158,11,0.08)' }}
                        >
                          <ArrowDown size={13} />
                          Изъять
                        </button>
                      )}
                      <button
                        onClick={() => setSimulateGoal(g)}
                        className="p-2.5 rounded-xl transition-all duration-200 flex items-center justify-center"
                        title="Расчёт"
                        style={{
                          background: 'rgba(110,86,207,0.08)',
                          color: vars.primaryLight,
                          border: `1px solid rgba(110,86,207,0.12)`,
                        }}
                        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(110,86,207,0.15)' }}
                        onMouseLeave={e => { e.currentTarget.style.background = 'rgba(110,86,207,0.08)' }}
                      >
                        <BarChart3 size={14} />
                      </button>
                    </div>
                  )}

                  {/* Completed badge for done goals */}
                  {isComplete && (
                    <div
                      className="flex items-center justify-center gap-1.5 py-2 rounded-xl text-xs font-semibold mt-1"
                      style={{
                        background: 'rgba(16,185,129,0.08)',
                        color: vars.success,
                        border: '1px solid rgba(16,185,129,0.12)',
                      }}
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                      Цель достигнута
                    </div>
                  )}
                </motion.div>
              )
            })}
          </AnimatePresence>
        )}
      </div>

      {/* ── FAB (Floating Action Button) ── */}
      <motion.button
        onClick={() => { reset(); setShowAdd(true) }}
        className="fixed bottom-6 right-6 z-40 w-14 h-14 rounded-2xl flex items-center justify-center shadow-2xl"
        style={{
          background: 'linear-gradient(135deg, #6E56CF 0%, #7C3AED 100%)',
          boxShadow: '0 4px 20px rgba(110,86,207,0.45)',
        }}
        whileHover={{ scale: 1.08, boxShadow: '0 6px 28px rgba(110,86,207,0.55)' }}
        whileTap={{ scale: 0.94 }}
      >
        <Plus size={24} color="white" />
      </motion.button>

      {/* ─── Add Goal BottomSheet ─── */}
      <BottomSheet open={showAdd} onClose={() => { setShowAdd(false); reset() }} title="Новая цель" scrollBody>
        <div className="space-y-4">
          <Input
            label="Название"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Подушка безопасности"
            style={{ background: vars.surface, borderColor: vars.border, color: vars.textPrimary }}
          />
          <Input
            label="Целевая сумма (₽)"
            type="number"
            step="0.01"
            min="0.01"
            value={target}
            onChange={e => setTarget(e.target.value)}
            placeholder="500000"
            style={{ background: vars.surface, borderColor: vars.border, color: vars.textPrimary }}
          />
          <div>
            <label className="block text-sm font-medium mb-1.5" style={{ color: vars.textSecondary }}>
              Дедлайн
            </label>
            <input
              type="date"
              value={deadline}
              onChange={e => setDeadline(e.target.value)}
              className="w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none"
              style={{
                background: vars.surface,
                border: `1.5px solid ${vars.border}`,
                color: vars.textPrimary,
              }}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2" style={{ color: vars.textSecondary }}>Иконка</label>
            <div className="flex flex-wrap gap-2">
              {goalIcons.map(ico => (
                <button
                  key={ico}
                  onClick={() => setIcon(ico)}
                  className="rounded-xl flex items-center justify-center text-lg transition-all duration-200"
                  style={
                    icon === ico
                      ? { width: 36, height: 36, background: `rgba(110,86,207,0.15)`, boxShadow: `0 0 0 2px ${vars.primaryLight}` }
                      : { width: 36, height: 36, background: vars.surface, border: `1px solid ${vars.border}` }
                  }
                >
                  {ico}
                </button>
              ))}
            </div>
          </div>
          <div className="flex gap-3 pt-2">
            <button
              onClick={handleSave}
              disabled={isBusy || !name || !M.isPositive(M.fromInput(target))}
              className="flex-1 py-2.5 rounded-xl text-white font-medium text-sm transition-all duration-200 disabled:opacity-40"
              style={{
                background: `linear-gradient(135deg, ${vars.primary} 0%, ${vars.primaryLight} 100%)`,
                boxShadow: `0 4px 16px ${vars.primaryGlow}`,
              }}
            >
              {isBusy ? (
                <span className="flex items-center justify-center gap-2">
                  <Loader2 size={14} className="animate-spin" />
                  Создание…
                </span>
              ) : 'Создать'}
            </button>
            <button
              onClick={() => { setShowAdd(false); reset() }}
              className="py-2.5 px-4 rounded-xl font-medium text-sm transition-colors"
              style={{ background: vars.elevated, color: vars.textSecondary, border: `1px solid ${vars.border}` }}
            >
              <X size={16} />
            </button>
          </div>
        </div>
      </BottomSheet>

      {/* ─── Deposit BottomSheet ─── */}
      <BottomSheet open={!!depositGoal} onClose={() => setDepositGoal(null)} title="Внести на цель">
        <div className="space-y-4">
          {depositGoal && (
            <div
              className="flex items-center gap-3 p-3.5 rounded-xl"
              style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
            >
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
                style={{ background: 'rgba(16,185,129,0.12)' }}
              >
                {pickIcon(depositGoal)}
              </div>
              <div>
                <p className="text-sm font-semibold" style={{ color: vars.textPrimary }}>{depositGoal.name}</p>
                <p className="text-xs mt-0.5" style={{ color: vars.textSecondary }}>
                  <MoneySpan amount={depositGoal.current_amount} size="sm" className="!text-xs" />
                  {' '}из{' '}
                  <MoneySpan amount={depositGoal.target_amount} size="sm" className="!text-xs" />
                </p>
              </div>
            </div>
          )}
          <Input
            label="Сумма взноса (₽)"
            type="number"
            step="0.01"
            min="0.01"
            value={depositAmount}
            onChange={e => setDepositAmount(e.target.value)}
            placeholder="10000"
            style={{ background: vars.surface, borderColor: vars.border, color: vars.textPrimary }}
          />
          <button
            onClick={handleDeposit}
            disabled={contribute.isPending || !M.isPositive(M.fromInput(depositAmount))}
            className="w-full py-2.5 rounded-xl text-white font-medium text-sm transition-all duration-200 disabled:opacity-40"
            style={{
              background: 'linear-gradient(135deg, #059669, #10B981)',
              boxShadow: '0 4px 16px rgba(16,185,129,0.3)',
            }}
          >
            {contribute.isPending ? (
              <span className="flex items-center justify-center gap-2">
                <Loader2 size={14} className="animate-spin" />
                Вношу…
              </span>
            ) : 'Внести'}
          </button>
        </div>
      </BottomSheet>

      {/* ─── Withdraw BottomSheet ─── */}
      <BottomSheet open={!!withdrawGoal} onClose={() => setWithdrawGoal(null)} title="Изъять со счёта цели">
        <div className="space-y-4">
          {withdrawGoal && (
            <div
              className="flex items-center gap-3 p-3.5 rounded-xl"
              style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
            >
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
                style={{ background: 'rgba(245,158,11,0.12)' }}
              >
                {pickIcon(withdrawGoal)}
              </div>
              <div>
                <p className="text-sm font-semibold" style={{ color: vars.textPrimary }}>{withdrawGoal.name}</p>
                <p className="text-xs mt-0.5" style={{ color: vars.textSecondary }}>
                  Доступно: <MoneySpan amount={withdrawGoal.current_amount} size="sm" className="!text-xs" />
                </p>
              </div>
            </div>
          )}
          <Input
            label="Сумма (₽)"
            type="number"
            step="0.01"
            min="0.01"
            value={withdrawAmount}
            onChange={e => setWithdrawAmount(e.target.value)}
            placeholder="5000"
            style={{ background: vars.surface, borderColor: vars.border, color: vars.textPrimary }}
          />
          <button
            onClick={handleWithdraw}
            disabled={
              contribute.isPending
              || !M.isPositive(M.fromInput(withdrawAmount))
              || (withdrawGoal !== null && M.fromInput(withdrawAmount).gt(toDecimal(withdrawGoal.current_amount)))
            }
            className="w-full py-2.5 rounded-xl text-white font-medium text-sm transition-all duration-200 disabled:opacity-40"
            style={{
              background: `linear-gradient(135deg, #D97706, ${vars.warning})`,
              boxShadow: '0 4px 16px rgba(245,158,11,0.3)',
            }}
          >
            {contribute.isPending ? (
              <span className="flex items-center justify-center gap-2">
                <Loader2 size={14} className="animate-spin" />
                Изымаю…
              </span>
            ) : 'Изъять'}
          </button>
        </div>
      </BottomSheet>

      {/* ─── Simulation Modal ─── */}
      <SimulationModal
        goal={simulateGoal}
        onClose={() => setSimulateGoal(null)}
        monthIncome={kpi.monthIncome.toString()}
        preserveGoal={setSimulateGoal}
      />
    </div>
  )
}

/* ── SimulationModal ── */
function SimulationModal({
  goal, onClose, monthIncome, preserveGoal,
}: {
  goal: Goal | null; onClose: () => void; monthIncome: string;
  preserveGoal: (g: Goal | null) => void
}) {
  const { data: proj, isLoading } = useGoalProjection(goal?.id ?? 0, !!goal)

  if (!goal) return null

  const current = toDecimal(goal.current_amount)
  const targetVal = toDecimal(goal.target_amount)
  const remaining = M.sub(targetVal, current)
  const feasible = proj ? toDecimal(proj.required_monthly).lte(toDecimal(monthIncome).div(2)) : false

  return (
    <Modal open onClose={onClose} title="Расчёт цели">
      <div className="space-y-4">
        {/* Goal info */}
        <div
          className="flex items-center gap-3 p-3.5 rounded-xl"
          style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
        >
          <div
            className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
            style={{ background: 'rgba(110,86,207,0.12)' }}
          >
            {pickIcon(goal)}
          </div>
          <div>
            <p className="text-sm font-semibold" style={{ color: vars.textPrimary }}>{goal.name}</p>
            <p className="text-xs mt-0.5" style={{ color: vars.textSecondary }}>
              <MoneySpan amount={goal.current_amount} size="sm" className="!text-xs" />
              {' '}из{' '}
              <MoneySpan amount={goal.target_amount} size="sm" className="!text-xs" />
            </p>
          </div>
        </div>

        {isLoading ? (
          <div
            className="py-8 rounded-xl flex items-center justify-center gap-2"
            style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
          >
            <Loader2 className="animate-spin" size={18} style={{ color: vars.primaryLight }} />
            <span className="text-sm" style={{ color: vars.textSecondary }}>Расчёт…</span>
          </div>
        ) : proj ? (
          <>
            {/* Stats grid */}
            <div className="grid grid-cols-2 gap-3">
              <div
                className="rounded-xl p-4"
                style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
              >
                <div className="flex items-center gap-2 mb-2">
                  <TrendingUp size={14} style={{ color: vars.primaryLight }} />
                  <p className="text-xs font-medium" style={{ color: vars.textSecondary }}>
                    Осталось накопить
                  </p>
                </div>
                <p className="text-lg font-bold font-mono-money" style={{ color: vars.textPrimary }}>
                  {formatMoney(remaining.toFixed(2))}
                </p>
              </div>
              <div
                className="rounded-xl p-4"
                style={{ background: vars.surface, border: `1px solid ${vars.border}` }}
              >
                <div className="flex items-center gap-2 mb-2">
                  <Calendar size={14} style={{ color: vars.primaryLight }} />
                  <p className="text-xs font-medium" style={{ color: vars.textSecondary }}>
                    Осталось месяцев
                  </p>
                </div>
                <p className="text-lg font-bold font-mono-money" style={{ color: vars.textPrimary }}>
                  {proj.months_left}
                </p>
              </div>
            </div>

            {/* Feasibility card */}
            <div
              className="rounded-xl p-4 border-2"
              style={
                feasible
                  ? {
                      borderColor: 'rgba(16,185,129,0.3)',
                      background: 'rgba(16,185,129,0.06)',
                      boxShadow: '0 0 20px rgba(16,185,129,0.08)',
                    }
                  : {
                      borderColor: 'rgba(245,158,11,0.3)',
                      background: 'rgba(245,158,11,0.06)',
                      boxShadow: '0 0 20px rgba(245,158,11,0.08)',
                    }
              }
            >
              <p className="text-xs font-medium mb-2" style={{ color: vars.textSecondary }}>
                Необходимо откладывать в месяц
              </p>
              <p className="text-xl font-bold font-mono-money" style={{ color: vars.primaryLight }}>
                {formatMoney(proj.required_monthly || '0')}
              </p>
              <div className="flex items-start gap-2 mt-2">
                <span className="text-base flex-shrink-0 mt-0.5">
                  {feasible ? '✅' : '⚠'}
                </span>
                <p className="text-xs leading-relaxed" style={{ color: vars.textSecondary }}>
                  {feasible
                    ? 'Цель достижима — это менее 50% вашего месячного дохода'
                    : 'Цель требует более 50% дохода — пересмотрите срок или сумму'}
                </p>
              </div>
            </div>
          </>
        ) : (
          <div
            className="py-8 rounded-xl text-center text-sm"
            style={{ background: vars.surface, border: `1px solid ${vars.border}`, color: vars.textTertiary }}
          >
            Нет данных по сроку цели
          </div>
        )}

        <button
          onClick={() => { preserveGoal(null); onClose() }}
          className="w-full py-2.5 rounded-xl font-medium text-sm transition-all duration-200"
          style={{
            background: vars.elevated,
            color: vars.textSecondary,
            border: `1px solid ${vars.border}`,
          }}
        >
          Закрыть
        </button>
      </div>
    </Modal>
  )
}
