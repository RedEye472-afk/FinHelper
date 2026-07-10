/**
 * DashboardPage — реальные данные через React Query + decimal.js.
 * Нет фейковых чисел. При отсутствии данных → empty state с CTA.
 */
import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Target, Calculator, PiggyBank, Eye, EyeOff, ArrowUpRight, ArrowDownRight } from 'lucide-react'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts'
import { motion, AnimatePresence } from 'framer-motion'
import { useFinance } from '../store'
import { useSettings } from '../hooks/useSettings'
import { useGoals, useBudgets, useOperations } from '../api/queries'
import { toDecimal, formatMoney, formatCompact, M } from '../lib/money'
import { Decimal } from '../lib/money'
import { Card } from '../components/ui/Card'
import { Skeleton } from '../components/ui/Skeleton'

const categoryColors: Record<string, string> = {
  'Зарплата': '#10b981', 'Фриланс': '#34d399', 'Продукты': '#f59e0b',
  'Транспорт': '#3b82f6', 'Рестораны': '#f97316', 'Жильё': '#ef4444',
  'Развлечения': '#8b5cf6', 'Здоровье': '#ec4899', 'Связь': '#06b6d4',
  'Одежда': '#6366f1', 'Образование': '#14b8a6', 'Подарки': '#f43f5e',
  'Спорт': '#a855f7', 'Подписки': '#0ea5e9', 'Переводы': '#64748b', 'Прочее': '#6b7280',
}

export function DashboardPage() {
  const navigate = useNavigate()
  const { kpi, expensesByCategory, isLoading } = useFinance()
  const { hideBalance, setSettings } = useSettings()

  const { data: goalsData } = useGoals()
  const { data: budgetsData } = useBudgets()
  const { data: opsData } = useOperations(10)

  const goals = goalsData ?? []
  const budgets = budgetsData ?? []
  const recentOps = opsData?.items ?? []

  const quickActions = [
    { icon: Plus, label: 'Операция', onClick: () => navigate('/operations/new') },
    { icon: Target, label: 'Цель', onClick: () => navigate('/goals') },
    { icon: Calculator, label: 'Калькулятор', onClick: () => navigate('/deposit') },
    { icon: PiggyBank, label: 'Бюджет', onClick: () => navigate('/budgets') },
  ]

  const activeGoals = useMemo(() => {
    return goals
      .filter(g => {
        const target = toDecimal(g.target_amount)
        const current = toDecimal(g.current_amount)
        const pct = target.gt(0) ? current.div(target).mul(100) : new Decimal(0)
        return pct.lt(100)
      })
      .map(g => {
        const target = toDecimal(g.target_amount)
        const current = toDecimal(g.current_amount)
        const pct = target.gt(0) ? current.div(target).mul(100) : new Decimal(0)
        return { ...g, target, current, pct }
      })
      .slice(0, 3)
  }, [goals])

  const budgetChart = useMemo(() => {
    if (budgets.length === 0) return []
    return budgets.slice(0, 8).map(b => {
      const name = (b.category_name || 'Прочее')
      return {
        name: name.length > 8 ? name.slice(0, 8) + '\u2026' : name,
        limit: toDecimal(b.limit_amount).toNumber(),
      }
    })
  }, [budgets])

  const pieData = useMemo(() => {
    if (expensesByCategory.length === 0) return []
    return expensesByCategory.slice(0, 6).map(e => ({
      category: e.category,
      color: e.color,
      value: e.amount.toNumber(),
    }))
  }, [expensesByCategory])

  const savedThisMonth = useMemo(() => M.sub(kpi.monthIncome, kpi.monthExpense), [kpi])

  const hasData = kpi.totalBalance.gt(0) || recentOps.length > 0 || activeGoals.length > 0 || budgets.length > 0

  return (
    <div className="space-y-5">
      <AnimatePresence mode="wait">
        {isLoading ? (
          <motion.div key="loading" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} transition={{ duration: 0.2 }} className="space-y-4">
            <Skeleton className="h-40 rounded-2xl" />
            <Skeleton className="h-24 rounded-2xl" />
            <Skeleton className="h-52 rounded-2xl" />
          </motion.div>
        ) : (
          <motion.div key="content" layout initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.25 }} className="space-y-5">
            {/* Empty state when no data */}
            {!hasData && (
              <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4, delay: 0.1 }} className="space-y-6">
                <Card className="bg-gradient-primary text-white shadow-glow">
                  <div className="p-6 text-center">
                    <div className="w-16 h-16 rounded-2xl bg-white/20 flex items-center justify-center mx-auto mb-4">
                      <Target size={32} className="text-white" />
                    </div>
                    <h2 className="text-xl font-bold mb-2">\u0414\u043e\u0431\u0440\u043e \u043f\u043e\u0436\u0430\u043b\u043e\u0432\u0430\u0442\u044c \u0432 FinHelper!</h2>
                    <p className="text-white/80 max-w-xs mx-auto">
                      \u0414\u043e\u0431\u0430\u0432\u044c\u0442\u0435 \u043f\u0435\u0440\u0432\u0443\u044e \u043e\u043f\u0435\u0440\u0430\u0446\u0438\u044e, \u0441\u043e\u0437\u0434\u0430\u0439\u0442\u0435 \u0446\u0435\u043b\u044c \u0438\u043b\u0438 \u0431\u044e\u0434\u0436\u0435\u0442 \u2014 \u0438 \u0437\u0434\u0435\u0441\u044c \u043f\u043e\u044f\u0432\u044f\u0442\u0441\u044f \u0432\u0430\u0448\u0438 \u0438\u043d\u0444\u043e\u0433\u0440\u0430\u0444\u0438\u043a\u0438.
                    </p>
                  </div>
                </Card>

                <div className="grid grid-cols-2 gap-3">
                  <button onClick={() => navigate('/operations/new')} className="card-hover p-4 rounded-2xl text-center border-2 border-dashed border-primary-300">
                    <Plus size={28} className="text-primary-500 mx-auto mb-2" />
                    <p className="font-medium">\u0414\u043e\u0431\u0430\u0432\u0438\u0442\u044c \u043e\u043f\u0435\u0440\u0430\u0446\u0438\u044e</p>
                    <p className="text-xs text-tertiary mt-1">\u0414\u043e\u0445\u043e\u0434 \u0438\u043b\u0438 \u0440\u0430\u0441\u0445\u043e\u0434</p>
                  </button>
                  <button onClick={() => navigate('/goals')} className="card-hover p-4 rounded-2xl text-center border-2 border-dashed border-primary-300">
                    <Target size={28} className="text-primary-500 mx-auto mb-2" />
                    <p className="font-medium">\u0421\u043e\u0437\u0434\u0430\u0442\u044c \u0446\u0435\u043b\u044c</p>
                    <p className="text-xs text-tertiary mt-1">\u041d\u0430\u043a\u043e\u043f\u043b\u0435\u043d\u0438\u044f \u043f\u043e\u0434 \u043c\u0435\u0447\u0442\u0443</p>
                  </button>
                  <button onClick={() => navigate('/budgets')} className="card-hover p-4 rounded-2xl text-center border-2 border-dashed border-primary-300">
                    <PiggyBank size={28} className="text-primary-500 mx-auto mb-2" />
                    <p className="font-medium">\u0417\u0430\u0432\u0435\u0441\u0442\u0438 \u0431\u044e\u0434\u0436\u0435\u0442</p>
                    <p className="text-xs text-tertiary mt-1">\u041a\u043e\u043d\u0442\u0440\u043e\u043b\u044c \u043b\u0438\u043c\u0438\u0442\u043e\u0432</p>
                  </button>
                  <button onClick={() => navigate('/deposit')} className="card-hover p-4 rounded-2xl text-center border-2 border-dashed border-primary-300">
                    <Calculator size={28} className="text-primary-500 mx-auto mb-2" />
                    <p className="font-medium">\u041a\u0430\u043b\u044c\u043a\u0443\u043b\u044f\u0442\u043e\u0440 \u0432\u043a\u043b\u0430\u0434\u0430</p>
                    <p className="text-xs text-tertiary mt-1">\u0420\u0430\u0441\u0447\u0451\u0442 \u0434\u043e\u0445\u043e\u0434\u043d\u043e\u0441\u0442\u0438</p>
                  </button>
                </div>

                <Card>
                  <h3 className="text-sm font-semibold mb-3" style={{color: 'var(--text-primary)'}}>\u0411\u044b\u0441\u0442\u0440\u044b\u0435 \u0434\u0435\u0439\u0441\u0442\u0432\u0438\u044f</h3>
                  <div className="grid grid-cols-4 gap-3">
                    {[
                      { icon: Plus, label: '\u041e\u043f\u0435\u0440\u0430\u0446\u0438\u044f', href: '/operations/new' },
                      { icon: Target, label: '\u0426\u0435\u043b\u044c', href: '/goals' },
                      { icon: Calculator, label: '\u041a\u0430\u043b\u044c\u043a\u0443\u043b\u044f\u0442\u043e\u0440', href: '/deposit' },
                      { icon: PiggyBank, label: '\u0411\u044e\u0434\u0436\u0435\u0442', href: '/budgets' },
                    ].map(({ icon: Icon, label, href }) => (
                      <button key={label} onClick={() => navigate(href)} className="flex flex-col items-center gap-2 p-3 rounded-2xl btn-press">
                        <div className="w-10 h-10 rounded-xl bg-gradient-primary flex items-center justify-center shadow-lg">
                          <Icon size={18} className="text-white" />
                        </div>
                        <span className="text-[11px] font-medium" style={{color: 'var(--text-secondary)'}}>{label}</span>
                      </button>
                    ))}
                  </div>
                </Card>
              </motion.div>
            )}

            {hasData && (
              <>
                {/* Hero */}
                <div className="relative -mx-4 -mt-4 px-4 pt-6 pb-8 bg-gradient-primary overflow-hidden">
                  <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1)' }} />
                  <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2)' }} />
                  <div className="relative">
                    <div className="flex items-center justify-between mb-4">
                      <div>
                        <p className="text-sm text-white/70">\u0414\u043e\u0431\u0440\u043e \u043f\u043e\u0436\u0430\u043b\u043e\u0432\u0430\u0442\u044c,</p>
                        <p className="text-lg font-semibold text-white">\u0410\u043b\u0435\u043a\u0441\u0435\u0439</p>
                      </div>
                      <button onClick={() => setSettings({ hideBalance: !hideBalance })} className="p-2 rounded-xl bg-white/15 text-white/80 hover:bg-white/25 transition-all backdrop-blur-sm">
                        {hideBalance ? <EyeOff size={20} /> : <Eye size={20} />}
                      </button>
                    </div>
                    <div className="mb-5">
                      <p className="text-xs text-white/60 font-medium mb-1">\u041e\u0431\u0449\u0438\u0439 \u0431\u0430\u043b\u0430\u043d\u0441</p>
                      <p className="text-3xl font-bold font-mono-money text-white tracking-tight">
                        {hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(kpi.totalBalance)}
                      </p>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <motion.div layout className="glass p-4 rounded-2xl">
                        <div className="flex items-center gap-1.5 text-xs text-white/70 mb-1">
                          <ArrowUpRight size={12} className="text-emerald-300" /> \u0414\u043e\u0445\u043e\u0434\u044b
                        </div>
                        <p className="text-base font-semibold font-mono-money text-emerald-300">
                          {hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(kpi.monthIncome)}
                        </p>
                      </motion.div>
                      <motion.div layout className="glass p-4 rounded-2xl">
                        <div className="flex items-center gap-1.5 text-xs text-white/70 mb-1">
                          <ArrowDownRight size={12} className="text-red-300" /> \u0420\u0430\u0441\u0445\u043e\u0434\u044b
                        </div>
                        <p className="text-base font-semibold font-mono-money text-red-300">
                          {hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(kpi.monthExpense)}
                        </p>
                      </motion.div>
                    </div>
                  </div>
                </div>

                {/* Quick Actions */}
                <div className="grid grid-cols-4 gap-3">
                  {quickActions.map((action) => (
                    <button key={action.label} onClick={action.onClick} className="flex flex-col items-center gap-2 p-3 rounded-2xl card card-hover btn-press">
                      <div className="w-10 h-10 rounded-xl bg-gradient-primary flex items-center justify-center shadow-lg">
                        <action.icon size={18} className="text-white" />
                      </div>
                      <span className="text-[11px] font-medium" style={{color: 'var(--text-secondary)'}}>{action.label}</span>
                    </button>
                  ))}
                </div>

                {/* Savings Rate */}
                <Card className="relative overflow-hidden">
                  <div className="absolute inset-0 bg-gradient-primary opacity-10" />
                  <div className="relative">
                    <div className="flex items-center justify-between mb-3">
                      <p className="text-sm font-medium" style={{color: 'var(--text-secondary)'}}>\u041d\u043e\u0440\u043c\u0430 \u0441\u0431\u0435\u0440\u0435\u0436\u0435\u043d\u0438\u0439</p>
                      <span className="text-2xl font-bold font-mono-money gradient-text">{kpi.savingsRate}%</span>
                    </div>
                    <div className="w-full rounded-full h-2.5" style={{background: 'var(--border-default)'}}>
                      <div className="h-2.5 rounded-full progress-bar" style={{ width: `${Math.min(kpi.savingsRate, 100)}%` }} />
                    </div>
                    <p className="text-xs mt-2" style={{color: 'var(--text-tertiary)'}}>
                      \u041d\u0430\u043a\u043e\u043f\u043b\u0435\u043d\u043e: {hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(savedThisMonth)} \u0432 \u044d\u0442\u043e\u043c \u043c\u0435\u0441\u044f\u0446\u0435
                    </p>
                  </div>
                </Card>

                {/* Budget Chart */}
                {budgetChart.length > 0 && (
                  <Card>
                    <h3 className="text-sm font-semibold mb-3" style={{color: 'var(--text-primary)'}}>\u0411\u044e\u0434\u0436\u0435\u0442\u044b</h3>
                    <div className="h-52">
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={budgetChart}>
                          <CartesianGrid strokeDasharray="3 3" stroke="var(--border-default)" />
                          <XAxis dataKey="name" tick={{ fontSize: 10, fill: 'var(--text-tertiary)' }} stroke="var(--border-default)" />
                          <YAxis tick={{ fontSize: 11, fill: 'var(--text-tertiary)' }} stroke="var(--border-default)" tickFormatter={v => formatCompact(new Decimal(v))} />
                          <Tooltip
                            contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', borderRadius: '12px' }}
                            formatter={(value: any) => [formatMoney(new Decimal(value))]}
                          />
                          <Bar dataKey="limit" fill="var(--color-primary-300)" name="\u041b\u0438\u043c\u0438\u0442" radius={[4, 4, 0, 0]} />
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  </Card>
                )}

                {/* Expenses Pie */}
                {pieData.length > 0 && (
                  <Card>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>\u0420\u0430\u0441\u0445\u043e\u0434\u044b \u043f\u043e \u043a\u0430\u0442\u0435\u0433\u043e\u0440\u0438\u044f\u043c</h3>
                      <button onClick={() => navigate('/operations')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>\u0412\u0441\u0435</button>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="w-36 h-36 flex-shrink-0">
                        <ResponsiveContainer width="100%" height="100%">
                          <PieChart>
                            <Pie data={pieData} cx="50%" cy="50%" innerRadius={35} outerRadius={65} dataKey="value" paddingAngle={3}>
                              {pieData.map((entry) => (
                                <Cell key={entry.category} fill={entry.color || categoryColors[entry.category] || '#6b7280'} />
                              ))}
                            </Pie>
                            <Tooltip
                              contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', borderRadius: '12px' }}
                              formatter={(value: any) => [formatMoney(new Decimal(value))]}
                            />
                          </PieChart>
                        </ResponsiveContainer>
                      </div>
                      <div className="flex-1 space-y-2">
                        {expensesByCategory.slice(0, 5).map((item) => (
                          <div key={item.category} className="flex items-center gap-2 text-xs">
                            <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: item.color || categoryColors[item.category] || '#6b7280' }} />
                            <span style={{color: 'var(--text-secondary)'}} className="flex-1 truncate">{item.category}</span>
                            <span className="font-medium" style={{color: 'var(--text-primary)'}}>{hideBalance ? '\u2022\u2022\u2022\u2022' : formatCompact(item.amount)}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </Card>
                )}

                {/* Active Goals */}
                {activeGoals.length > 0 && (
                  <Card>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>\u0424\u0438\u043d\u0430\u043d\u0441\u043e\u0432\u044b\u0435 \u0446\u0435\u043b\u0438</h3>
                      <button onClick={() => navigate('/goals')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>\u0412\u0441\u0435</button>
                    </div>
                    <div className="space-y-3">
                      {activeGoals.map((goal) => {
                        const pctNum = goal.pct.toNumber()
                        return (
                          <div key={goal.id}>
                            <div className="flex items-center justify-between text-xs mb-1.5">
                              <div className="flex items-center gap-1.5">
                                <span className="text-base">\uD83C\uDFAF</span>
                                <span className="font-medium" style={{color: 'var(--text-primary)'}}>{goal.name}</span>
                              </div>
                              <span style={{color: 'var(--text-tertiary)'}}>{pctNum.toFixed(0)}%</span>
                            </div>
                            <div className="w-full rounded-full h-2" style={{background: 'var(--border-default)'}}>
                              <div className="h-2 rounded-full progress-bar" style={{ width: `${Math.min(pctNum, 100)}%` }} />
                            </div>
                            <p className="text-[11px] mt-1" style={{color: 'var(--text-tertiary)'}}>
                              {formatMoney(goal.current)} / {formatMoney(goal.target)}
                            </p>
                          </div>
                        )
                      })}
                    </div>
                  </Card>
                )}

                {/* Budgets List */}
                {budgets.length > 0 && (
                  <Card>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>\u041b\u0438\u043c\u0438\u0442\u044b \u0431\u044e\u0434\u0436\u0435\u0442\u043e\u0432</h3>
                      <button onClick={() => navigate('/budgets')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>\u0412\u0441\u0435</button>
                    </div>
                    <div className="space-y-3">
                      {budgets.slice(0, 4).map((b) => (
                        <div key={b.id} className="flex items-center justify-between text-xs">
                          <span style={{color: 'var(--text-secondary)'}}>{b.category_name || '\u041f\u0440\u043e\u0447\u0435\u0435'}</span>
                          <span className="font-medium" style={{color: 'var(--text-primary)'}}>
                            {hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(toDecimal(b.limit_amount))}
                          </span>
                        </div>
                      ))}
                    </div>
                  </Card>
                )}

                {/* Recent Operations */}
                <Card>
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-sm font-semibold" style={{color: 'var(--text-primary)'}}>\u041f\u043e\u0441\u043b\u0435\u0434\u043d\u0438\u0435 \u043e\u043f\u0435\u0440\u0430\u0446\u0438\u0438</h3>
                    <button onClick={() => navigate('/operations')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>\u0412\u0441\u0435</button>
                  </div>
                  <div className="space-y-1">
                    {recentOps.length === 0 ? (
                      <p className="text-sm text-center py-6" style={{color: 'var(--text-tertiary)'}}>\u041d\u0435\u0442 \u043e\u043f\u0435\u0440\u0430\u0446\u0438\u0439</p>
                    ) : (
                      recentOps.slice(0, 5).map((op) => {
                        const amount = toDecimal(op.amount)
                        const isIncome = op.type === 'income'
                        return (
                          <div key={op.id} className="flex items-center gap-3 py-2.5" style={{borderBottom: '1px solid var(--border-subtle)'}}>
                            <div className="w-9 h-9 rounded-xl flex items-center justify-center text-sm"
                              style={isIncome ? {background: 'rgba(16,185,129,0.1)', color: 'var(--color-primary-600)'} : {background: 'var(--color-danger-50)', color: 'var(--color-danger-500)'}}>
                              {isIncome ? '\u2191' : '\u2193'}
                            </div>
                            <div className="flex-1 min-w-0">
                              <p className="text-sm font-medium truncate" style={{color: 'var(--text-primary)'}}>{op.description || '\u0411\u0435\u0437 \u043e\u043f\u0438\u0441\u0430\u043d\u0438\u044f'}</p>
                              <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>
                                {new Date(op.operation_date).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })}
                              </p>
                            </div>
                            <span className="font-mono-money font-semibold text-sm" style={{color: isIncome ? 'var(--color-primary-600)' : 'var(--color-danger-500)'}}>
                              {isIncome ? '+' : '\u2212'}{hideBalance ? '\u2022\u2022\u2022\u2022' : formatMoney(amount)}
                            </span>
                          </div>
                        )
                      })
                    )}
                  </div>
                </Card>
              </>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
