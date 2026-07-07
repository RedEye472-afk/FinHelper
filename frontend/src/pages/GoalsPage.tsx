import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Plus, BarChart3, PiggyBank, X, Loader2, Target } from 'lucide-react'
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
  if (hideBalance) return <span className={`font-mono-money ${className}`} style={{ color: 'var(--text-tertiary)' }}>••••</span>
  const sizes = { sm: 'text-sm', md: 'text-base', lg: 'text-xl' }
  return <span className={`font-mono-money font-semibold ${sizes[size]} ${className}`}>{formatMoney(amount)}</span>
}

/** Circular progress ring SVG. */
function ProgressRing({ pct, size = 44, stroke = 4 }: { pct: number; size?: number; stroke?: number }) {
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const clamped = Math.min(Math.max(pct, 0), 100)
  const offset = c - (clamped / 100) * c
  return (
    <svg width={size} height={size} className="flex-shrink-0 -rotate-90">
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" strokeWidth={stroke} style={{ stroke: 'var(--border-default)' }} />
      <circle
        cx={size / 2} cy={size / 2} r={r} fill="none" strokeWidth={stroke}
        strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round"
        style={{
          stroke: pct >= 100 ? 'var(--color-primary-600)' : pct >= 50 ? 'var(--color-primary-500)' : 'var(--color-warning-500)',
          transition: 'stroke-dashoffset 0.5s ease',
        }}
      />
    </svg>
  )
}

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
    if (amtDec.gt(current)) return // can't withdraw more than available
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
    <div className="space-y-4">
      <div className="bg-gradient-primary rounded-2xl p-5 text-white relative overflow-hidden">
        <Target className="absolute -right-2 -bottom-2 opacity-20" size={72} strokeWidth={1.5} />
        <div className="relative flex items-center justify-between">
          <div>
            <p className="text-sm text-white/80 mb-1">Общий прогресс</p>
            <p className="text-3xl font-bold font-mono-money">{Math.round(totalProgress)}%</p>
          </div>
          <ProgressRing pct={totalProgress} size={56} stroke={5} />
        </div>
        <div className="relative w-full bg-white/20 rounded-full h-2 mt-3 overflow-hidden">
          <div className="bg-white h-2 rounded-full transition-all duration-500" style={{ width: `${Math.min(totalProgress, 100)}%` }} />
        </div>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
          Мои цели ({list.length}){goalsLoading && <Loader2 className="inline ml-1 animate-spin" size={12} />}
        </h2>
        <button onClick={() => { reset(); setShowAdd(true) }} className="flex items-center gap-1 text-sm font-medium" style={{ color: 'var(--color-primary-600)' }}><Plus size={16} /> Новая</button>
      </div>

      <div className="space-y-3">
        {list.length === 0 && !goalsLoading ? (
          <Card className="py-8 text-center">
            <PiggyBank className="mx-auto mb-2" size={32} style={{ color: 'var(--text-tertiary)' }} />
            <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Нет целей. Создайте первую!</p>
          </Card>
        ) : (
          <AnimatePresence mode="popLayout">
            {list.map(g => {
              const current = toDecimal(g.current_amount)
              const target = toDecimal(g.target_amount)
              const pct = target.gt(0) ? Number(current.div(target).mul(100).toFixed(2)) : 0
              const barColor = pct >= 100 ? 'var(--color-primary-600)' : pct >= 50 ? 'var(--color-primary-500)' : 'var(--color-warning-500)'
              const daysLeft = g.target_date ? Math.ceil((new Date(g.target_date).getTime() - Date.now()) / 86400000) : 0
              const status = pct >= 100 ? 'Выполнена' : 'В процессе'
              const icon = pickIcon(g)
              return (
                <motion.div
                  key={g.id}
                  layout
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.2 }}
                  className="card card-hover p-4 w-full"
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-2.5">
                      <span className="text-2xl">{icon}</span>
                      <div>
                        <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>{g.name}</p>
                        <span
                          className="text-[10px] font-medium px-2 py-0.5 rounded-full"
                          style={status === 'Выполнена'
                            ? { background: 'rgba(16,185,129,0.1)', color: 'var(--color-primary-600)' }
                            : { background: 'rgba(59,130,246,0.1)', color: 'var(--color-primary-600)' }
                          }
                        >{status}</span>
                      </div>
                    </div>
                    <span className="text-lg font-bold font-mono-money" style={{ color: pct >= 100 ? 'var(--color-primary-600)' : 'var(--text-primary)' }}>
                      {Math.round(pct)}%
                    </span>
                  </div>
                  <div className="w-full rounded-full h-2.5 mb-2 overflow-hidden" style={{ background: 'var(--border-default)' }}>
                    <div className="h-2.5 rounded-full transition-all duration-500" style={{ width: `${Math.min(pct, 100)}%`, background: barColor }} />
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <MoneySpan amount={g.current_amount} size="sm" />
                    <span style={{ color: 'var(--text-tertiary)' }}>из <MoneySpan amount={g.target_amount} size="sm" /></span>
                    {daysLeft > 0 && <span style={{ color: 'var(--text-secondary)' }}>{daysLeft} дн.</span>}
                  </div>
                  {pct < 100 && (
                    <div className="flex gap-2 mt-3">
                      <button onClick={() => { setDepositGoal(g); setDepositAmount('') }}
                        className="flex-1 py-2 rounded-xl font-medium text-xs transition-colors"
                        style={{ background: 'rgba(16,185,129,0.1)', color: 'var(--color-primary-600)' }}>
                        + Внести
                      </button>
                      {current.gt(0) && (
                        <button onClick={() => { setWithdrawGoal(g); setWithdrawAmount('') }}
                          className="py-2 px-3 rounded-xl font-medium text-xs transition-colors"
                          style={{ background: 'rgba(245,158,11,0.08)', color: 'var(--color-warning-500)' }}>
                          Изъять
                        </button>
                      )}
                      <button onClick={() => setSimulateGoal(g)}
                        className="p-2 rounded-xl transition-colors" title="Расчёт"
                        style={{ background: 'var(--bg-surface)', color: 'var(--text-tertiary)', border: '1px solid var(--border-default)' }}>
                        <BarChart3 size={14} />
                      </button>
                    </div>
                  )}
                </motion.div>
              )
            })}
          </AnimatePresence>
        )}
      </div>

      {/* Add Goal BottomSheet */}
      <BottomSheet open={showAdd} onClose={() => { setShowAdd(false); reset() }} title="Новая цель" scrollBody>
        <div className="space-y-4">
          <Input label="Название" value={name} onChange={e => setName(e.target.value)} placeholder="Подушка безопасности" />
          <Input label="Целевая сумма (₽)" type="number" step="0.01" min="0.01" value={target} onChange={e => setTarget(e.target.value)} placeholder="500000" />
          <div>
            <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Дедлайн</label>
            <input type="date" value={deadline} onChange={e => setDeadline(e.target.value)}
              className="w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2"
              style={{
                background: 'var(--bg-surface)',
                borderColor: 'var(--border-default)',
                color: 'var(--text-primary)',
              } as React.CSSProperties} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>Иконка</label>
            <div className="flex flex-wrap gap-2">
              {goalIcons.map(ico => (
                <button key={ico} onClick={() => setIcon(ico)}
                  className="rounded-xl flex items-center justify-center text-lg transition-colors"
                  style={icon === ico
                    ? { width: 36, height: 36, background: 'rgba(16,185,129,0.12)', boxShadow: '0 0 0 2px var(--color-primary-500)' }
                    : { width: 36, height: 36, background: 'var(--bg-surface)' }
                  }>
                  {ico}
                </button>
              ))}
            </div>
          </div>
          <div className="flex gap-3 pt-2">
            <button onClick={handleSave} disabled={isBusy || !name || !M.isPositive(M.fromInput(target))}
              className="flex-1 py-2.5 rounded-xl bg-primary-500 text-white font-medium text-sm hover:bg-primary-600 transition-colors btn-press disabled:opacity-50">
              {isBusy ? 'Создание…' : 'Создать'}
            </button>
            <button onClick={() => { setShowAdd(false); reset() }} className="py-2.5 px-4 rounded-xl font-medium text-sm transition-colors"
              style={{ background: 'var(--bg-surface)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}>
              <X size={16} />
            </button>
          </div>
        </div>
      </BottomSheet>

      {/* Deposit BottomSheet */}
      <BottomSheet open={!!depositGoal} onClose={() => setDepositGoal(null)} title="Внести на цель">
        <div className="space-y-4">
          {depositGoal && (
            <div className="flex items-center gap-3 p-3 rounded-xl" style={{ background: 'var(--bg-surface)' }}>
              <span className="text-2xl">{pickIcon(depositGoal)}</span>
              <div>
                <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{depositGoal.name}</p>
                <p className="text-xs" style={{ color: 'var(--text-secondary)' }}><MoneySpan amount={depositGoal.current_amount} size="sm" /> из <MoneySpan amount={depositGoal.target_amount} size="sm" /></p>
              </div>
            </div>
          )}
          <Input label="Сумма взноса (₽)" type="number" step="0.01" min="0.01" value={depositAmount} onChange={e => setDepositAmount(e.target.value)} placeholder="10000" />
          <button onClick={handleDeposit} disabled={contribute.isPending || !M.isPositive(M.fromInput(depositAmount))}
            className="w-full py-2.5 rounded-xl bg-primary-500 text-white font-medium text-sm hover:bg-primary-600 transition-colors btn-press disabled:opacity-50">
            {contribute.isPending ? 'Вношу…' : 'Внести'}
          </button>
        </div>
      </BottomSheet>

      {/* Withdraw BottomSheet */}
      <BottomSheet open={!!withdrawGoal} onClose={() => setWithdrawGoal(null)} title="Изъять со счёта цели">
        <div className="space-y-4">
          {withdrawGoal && (
            <div className="flex items-center gap-3 p-3 rounded-xl" style={{ background: 'var(--bg-surface)' }}>
              <span className="text-2xl">{pickIcon(withdrawGoal)}</span>
              <div>
                <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{withdrawGoal.name}</p>
                <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Доступно: <MoneySpan amount={withdrawGoal.current_amount} size="sm" /></p>
              </div>
            </div>
          )}
          <Input label="Сумма (₽)" type="number" step="0.01" min="0.01" value={withdrawAmount} onChange={e => setWithdrawAmount(e.target.value)} placeholder="5000" />
          <button onClick={handleWithdraw}
            disabled={contribute.isPending || !M.isPositive(M.fromInput(withdrawAmount)) || (withdrawGoal !== null && M.fromInput(withdrawAmount).gt(toDecimal(withdrawGoal.current_amount)))}
            className="w-full py-2.5 rounded-xl text-white font-medium text-sm transition-colors btn-press disabled:opacity-50"
            style={{ background: 'var(--color-warning-500)' }}>
            {contribute.isPending ? 'Изымаю…' : 'Изъять'}
          </button>
        </div>
      </BottomSheet>

      {/* Simulation Modal */}
      <SimulationModal goal={simulateGoal} onClose={() => setSimulateGoal(null)} monthIncome={kpi.monthIncome.toString()} preserveGoal={setSimulateGoal} />
    </div>
  )
}

function SimulationModal({ goal, onClose, monthIncome, preserveGoal }: {
  goal: Goal | null; onClose: () => void; monthIncome: string;
  preserveGoal: (g: Goal | null) => void
}) {
  // 👇 conditional query only when goal is set — patch made this possible
  const { data: proj, isLoading } = useGoalProjection(goal?.id ?? 0, !!goal)

  if (!goal) return null

  const current = toDecimal(goal.current_amount)
  const target = toDecimal(goal.target_amount)
  const remaining = M.sub(target, current)
  const feasible = proj ? toDecimal(proj.required_monthly).lte(toDecimal(monthIncome).div(2)) : false

  return (
    <Modal open onClose={onClose} title="Расчёт цели">
      <div className="space-y-4">
        <div className="flex items-center gap-3 p-3 rounded-xl" style={{ background: 'var(--bg-surface)' }}>
          <span className="text-2xl">{pickIcon(goal)}</span>
          <div>
            <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{goal.name}</p>
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}><MoneySpan amount={goal.current_amount} size="sm" /> из <MoneySpan amount={goal.target_amount} size="sm" /></p>
          </div>
        </div>

        {isLoading ? (
          <Card className="py-6 text-center"><Loader2 className="inline animate-spin" size={20} /> <span className="ml-2 text-sm" style={{ color: 'var(--text-secondary)' }}>Расчёт…</span></Card>
        ) : proj ? (
          <>
            <div className="grid grid-cols-2 gap-3">
              <Card>
                <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Осталось накопить</p>
                <MoneySpan amount={remaining.toFixed(2)} size="md" />
              </Card>
              <Card>
                <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Осталось месяцев</p>
                <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{proj.months_left}</p>
              </Card>
            </div>

            <Card
              className="border-2"
              style={feasible
                ? { borderColor: 'var(--color-primary-500)', background: 'rgba(16,185,129,0.08)' }
                : { borderColor: 'var(--color-warning-500)', background: 'rgba(245,158,11,0.08)' }
              }
            >
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Необходимо откладывать в месяц</p>
              <p className="text-lg font-bold font-mono-money" style={{ color: 'var(--color-primary-600)' }}>{formatMoney(proj.required_monthly || '0')}</p>
              <p className="text-xs mt-1" style={{ color: 'var(--text-secondary)' }}>
                {feasible
                  ? '✅ Цель достижима — это менее 50% вашего месячного дохода'
                  : '⚠ Цель требует более 50% дохода — пересмотрите срок или сумму'}
              </p>
            </Card>
          </>
        ) : (
          <Card className="py-6 text-center text-sm" style={{ color: 'var(--text-tertiary)' }}>Нет данных по сроку цели</Card>
        )}

        <button onClick={() => { preserveGoal(null); onClose() }}
          className="w-full py-2.5 rounded-xl font-medium text-sm transition-colors"
          style={{ background: 'var(--bg-surface)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' }}>
          Закрыть
        </button>
      </div>
    </Modal>
  )
}