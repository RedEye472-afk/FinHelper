/**
 * DashboardPage — реальные данные с бэкенда, без моков.
 * Нет фейковых чисел. Пустые состояния для новых пользователей.
 */
import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Target, Calculator, PiggyBank, Eye, EyeOff, ArrowUpRight, ArrowDownRight, TrendingUp, Wallet } from 'lucide-react'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts'
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

  const activeGoals = useMemo(() => goals.filter(g => toDecimal(g.current_amount).lt(toDecimal(g.target_amount))).slice(0, 3), [goals])
  const budgetChart = useMemo(() => budgets.slice(0, 8).map(b => ({ name: (b.category_name || 'Прочее').length > 8 ? (b.category_name || 'Прочее').slice(0, 8) + '…' : (b.category_name || 'Прочее'), limit: toDecimal(b.limit_amount).toNumber() })), [budgets])
  const pieData = useMemo(() => expensesByCategory.slice(0, 6).map(e => ({ category: e.category, color: e.color, value: e.amount.toNumber() })), [expensesByCategory])
  const savedThisMonth = useMemo(() => M.sub(kpi.monthIncome, kpi.monthExpense), [kpi])
  const hasData = kpi.totalBalance.gt(0) || recentOps.length > 0 || budgets.length > 0

  if (isLoading) return (
    <div className="space-y-5">
      <Skeleton className="h-40 rounded-2xl" />
      <Skeleton className="h-24 rounded-2xl" />
      <Skeleton className="h-52 rounded-2xl" />
    </div>
  )

  return (
    <div className="space-y-5">
      {/* Empty state for new users */}
      {!hasData && (
        <div className="space-y-6">
          <Card className="bg-gradient-primary text-white shadow-glow">
            <div className="p-8 text-center">
              <div className="w-20 h-20 rounded-3xl bg-white/20 flex items-center justify-center mx-auto mb-5">
                <Wallet size={40} className="text-white" />
              </div>
              <h2 className="text-2xl font-bold mb-2">Добро пожаловать в FinHelper!</h2>
              <p className="text-white/70 max-w-sm mx-auto mb-6">
                Ваш личный финансовый навигатор. Добавьте первую операцию, и мы покажем красивую аналитику.
              </p>
              <button onClick={() => navigate('/operations/new')} className="inline-flex items-center gap-2 bg-white text-primary-700 font-semibold px-6 py-3 rounded-xl hover:bg-white/90 transition-all shadow-lg">
                <Plus size={18} />
                Добавить первую операцию
              </button>
            </div>
          </Card>

          <div className="grid grid-cols-3 gap-4">
            <button onClick={() => navigate('/operations/new')} className="card-hover p-5 rounded-2xl text-center border-2 border-dashed border-primary-300/50">
              <Plus size={24} className="text-primary-500 mx-auto mb-2" />
              <p className="font-medium text-sm">Добавить операцию</p>
              <p className="text-[11px] text-tertiary mt-1">Доход или расход</p>
            </button>
            <button onClick={() => navigate('/goals')} className="card-hover p-5 rounded-2xl text-center border-2 border-dashed border-primary-300/50">
              <Target size={24} className="text-primary-500 mx-auto mb-2" />
              <p className="font-medium text-sm">Создать цель</p>
              <p className="text-[11px] text-tertiary mt-1">Накопления на мечту</p>
            </button>
            <button onClick={() => navigate('/budgets')} className="card-hover p-5 rounded-2xl text-center border-2 border-dashed border-primary-300/50">
              <PiggyBank size={24} className="text-primary-500 mx-auto mb-2" />
              <p className="font-medium text-sm">Завести бюджет</p>
              <p className="text-[11px] text-tertiary mt-1">Контроль лимитов</p>
            </button>
          </div>

          <Card>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold">Быстрые действия</h3>
            </div>
            <div className="grid grid-cols-4 gap-3">
              {[
                { icon: Plus, label: 'Операция', href: '/operations/new' },
                { icon: Target, label: 'Цель', href: '/goals' },
                { icon: Calculator, label: 'Калькулятор', href: '/deposit' },
                { icon: PiggyBank, label: 'Бюджет', href: '/budgets' },
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
        </div>
      )}

      {/* Dashboard with data */}
      {hasData && (
        <>
          {/* Hero */}
          <div className="relative -mx-4 -mt-4 px-4 pt-6 pb-8 bg-gradient-primary overflow-hidden">
            <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1)' }} />
            <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2)' }} />
            <div className="relative">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <p className="text-sm text-white/70">Добро пожаловать,</p>
                  <p className="text-lg font-semibold text-white">Алексей</p>
                </div>
                <button onClick={() => setSettings({ hideBalance: !hideBalance })} className="p-2 rounded-xl bg-white/15 text-white/80 hover:bg-white/25 backdrop-blur-sm">
                  {hideBalance ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
              <div className="mb-5">
                <p className="text-xs text-white/60 font-medium mb-1">Общий баланс</p>
                <p className="text-3xl font-bold font-mono-money text-white tracking-tight">
                  {hideBalance ? '••••' : formatMoney(kpi.totalBalance)}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="glass p-4 rounded-2xl">
                  <div className="flex items-center gap-1.5 text-xs text-white/70 mb-1">
                    <ArrowUpRight size={12} className="text-emerald-300" /> Доходы
                  </div>
                  <p className="text-base font-semibold font-mono-money text-emerald-300">
                    {hideBalance ? '••••' : formatMoney(kpi.monthIncome)}
                  </p>
                </div>
                <div className="glass p-4 rounded-2xl">
                  <div className="flex items-center gap-1.5 text-xs text-white/70 mb-1">
                    <ArrowDownRight size={12} className="text-red-300" /> Расходы
                  </div>
                  <p className="text-base font-semibold font-mono-money text-red-300">
                    {hideBalance ? '••••' : formatMoney(kpi.monthExpense)}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-4 gap-3">
            {[
              { icon: Plus, label: 'Операция', href: '/operations/new' },
              { icon: Target, label: 'Цель', href: '/goals' },
              { icon: Calculator, label: 'Калькулятор', href: '/deposit' },
              { icon: PiggyBank, label: 'Бюджет', href: '/budgets' },
            ].map(({ icon: Icon, label, href }) => (
              <button key={label} onClick={() => navigate(href)} className="flex flex-col items-center gap-2 p-3 rounded-2xl card-hover btn-press">
                <div className="w-10 h-10 rounded-xl bg-gradient-primary flex items-center justify-center shadow-lg">
                  <Icon size={18} className="text-white" />
                </div>
                <span className="text-[11px] font-medium" style={{color: 'var(--text-secondary)'}}>{label}</span>
              </button>
            ))}
          </div>

          {/* Savings Rate */}
          <Card className="relative overflow-hidden">
            <div className="absolute inset-0 bg-gradient-primary opacity-10" />
            <div className="relative">
              <div className="flex items-center justify-between mb-3">
                <p className="text-sm font-medium" style={{color: 'var(--text-secondary)'}}>Норма сбережений</p>
                <span className="text-2xl font-bold font-mono-money gradient-text">{kpi.savingsRate}%</span>
              </div>
              <div className="w-full rounded-full h-2.5" style={{background: 'var(--border-default)'}}>
                <div className="h-2.5 rounded-full progress-bar" style={{ width: `${Math.min(kpi.savingsRate, 100)}%` }} />
              </div>
              <p className="text-xs mt-2" style={{color: 'var(--text-tertiary)'}}>
                Накоплено: {hideBalance ? '••••' : formatMoney(savedThisMonth)} в этом месяце
              </p>
            </div>
          </Card>

          {/* Budget Chart */}
          {budgetChart.length > 0 && (
            <Card>
              <h3 className="text-sm font-semibold mb-3">Бюджеты</h3>
              <div className="h-52">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={budgetChart}>
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--border-default)" />
                    <XAxis dataKey="name" tick={{ fontSize: 10, fill: 'var(--text-tertiary)' }} stroke="var(--border-default)" />
                    <YAxis tick={{ fontSize: 11, fill: 'var(--text-tertiary)' }} stroke="var(--border-default)" tickFormatter={v => formatCompact(new Decimal(v))} />
                    <Tooltip contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', borderRadius: '12px' }} formatter={(value: any) => [formatMoney(new Decimal(value))]} />
                    <Bar dataKey="limit" fill="var(--color-primary-300)" name="Лимит" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          )}

          {/* Expenses Pie */}
          {pieData.length > 0 && (
            <Card>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-semibold">Расходы по категориям</h3>
                <button onClick={() => navigate('/operations')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>Все</button>
              </div>
              <div className="flex items-center gap-4">
                <div className="w-36 h-36 flex-shrink-0">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie data={pieData} cx="50%" cy="50%" innerRadius={35} outerRadius={65} dataKey="value" paddingAngle={3}>
                        {pieData.map((entry) => <Cell key={entry.category} fill={entry.color || categoryColors[entry.category] || '#6b7280'} />)}
                      </Pie>
                      <Tooltip contentStyle={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-default)', borderRadius: '12px' }} formatter={(value: any) => [formatMoney(new Decimal(value))]} />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                <div className="flex-1 space-y-2">
                  {expensesByCategory.slice(0, 5).map((item) => (
                    <div key={item.category} className="flex items-center gap-2 text-xs">
                      <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: item.color || categoryColors[item.category] || '#6b7280' }} />
                      <span style={{color: 'var(--text-secondary)'}} className="flex-1 truncate">{item.category}</span>
                      <span className="font-medium" style={{color: 'var(--text-primary)'}}>{hideBalance ? '••••' : formatCompact(item.amount)}</span>
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
                <h3 className="text-sm font-semibold">Финансовые цели</h3>
                <button onClick={() => navigate('/goals')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>Все</button>
              </div>
              <div className="space-y-3">
                {activeGoals.slice(0, 3).map((goal) => {
                  const target = toDecimal(goal.target_amount)
                  const current = toDecimal(goal.current_amount)
                  const pct = target.gt(0) ? current.div(target).mul(100).toNumber() : 0
                  return (
                    <div key={goal.id}>
                      <div className="flex items-center justify-between text-xs mb-1.5">
                        <div className="flex items-center gap-1.5">
                          <span className="text-base">🎯</span>
                          <span className="font-medium" style={{color: 'var(--text-primary)'}}>{goal.name}</span>
                        </div>
                        <span style={{color: 'var(--text-tertiary)'}}>{pct.toFixed(0)}%</span>
                      </div>
                      <div className="w-full rounded-full h-2" style={{background: 'var(--border-default)'}}>
                        <div className="h-2 rounded-full progress-bar" style={{ width: `${Math.min(pct, 100)}%` }} />
                      </div>
                      <p className="text-[11px] mt-1" style={{color: 'var(--text-tertiary)'}}>
                        {formatMoney(current)} / {formatMoney(target)}
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
                <h3 className="text-sm font-semibold">Лимиты бюджетов</h3>
                <button onClick={() => navigate('/budgets')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>Все</button>
              </div>
              <div className="space-y-3">
                {budgets.slice(0, 4).map((b) => (
                  <div key={b.id} className="flex items-center justify-between text-xs">
                    <span style={{color: 'var(--text-secondary)'}}>{b.category_name || 'Прочее'}</span>
                    <span className="font-medium" style={{color: 'var(--text-primary)'}}>
                      {hideBalance ? '••••' : formatMoney(toDecimal(b.limit_amount))}
                    </span>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* Recent Operations */}
          <Card>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold">Последние операции</h3>
              <button onClick={() => navigate('/operations')} className="text-xs font-medium" style={{color: 'var(--color-primary-600)'}}>Все</button>
            </div>
            <div className="space-y-1">
              {recentOps.length === 0 ? (
                <p className="text-sm text-center py-6" style={{color: 'var(--text-tertiary)'}}>
                  Нет операций. <button onClick={() => navigate('/operations/new')} className="font-medium underline">Добавить</button>
                </p>
              ) : (
                recentOps.slice(0, 5).map((op) => {
                  const amount = toDecimal(op.amount)
                  const isIncome = op.type === 'income'
                  return (
                    <div key={op.id} className="flex items-center gap-3 py-2.5" style={{borderBottom: '1px solid var(--border-subtle)'}}>
                      <div className="w-9 h-9 rounded-xl flex items-center justify-center text-sm"
                        style={isIncome ? {background: 'rgba(16,185,129,0.1)', color: 'var(--color-primary-600)'} : {background: 'var(--color-danger-50)', color: 'var(--color-danger-500)'}}>
                        {isIncome ? '↑' : '↓'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate" style={{color: 'var(--text-primary)'}}>{op.description || 'Без описания'}</p>
                        <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>
                          {new Date(op.operation_date).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })}
                        </p>
                      </div>
                      <span className="font-mono-money font-semibold text-sm" style={{color: isIncome ? 'var(--color-primary-600)' : 'var(--color-danger-500)'}}>
                        {isIncome ? '+' : '−'}{hideBalance ? '••••' : formatMoney(amount)}
                      </span>
                    </div>
                  )
                })
              )}
            </div>
          </Card>
        </>
      )}
    </div>
  )
}
