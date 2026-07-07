import { useMemo, useState } from 'react'
import { useQueries } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { Plus, AlertTriangle, Loader2, Trash2 } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { BottomSheet } from '../components/ui/BottomSheet'
import { useBudgets, useCreateBudget, useDeleteBudget, useBudgetStatus, useCategories } from '../api/queries'
import { formatMoney, toDecimal, M } from '../lib/money'
import type { Decimal } from '../lib/money'
import { useSettings } from '../hooks/useSettings'
import type { Budget, BudgetStatus } from '../types'

const MONTHS_RU = ['янв', 'фев', 'мар', 'апр', 'май', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек']

/** Local palette — not imported from mockData (see task constraints). */
const CATEGORY_COLORS: Record<string, string> = {
  'Продукты': '#f59e0b', 'Транспорт': '#3b82f6', 'Рестораны': '#f97316', 'Жильё': '#ef4444',
  'Развлечения': '#8b5cf6', 'Здоровье': '#ec4899', 'Связь': '#06b6d4', 'Одежда': '#6366f1',
  'Образование': '#14b8a6', 'Подарки': '#f43f5e', 'Спорт': '#a855f7', 'Подписки': '#0ea5e9',
  'Переводы': '#64748b', 'Прочее': '#6b7280',
}
function colorFor(cat: string): string { return CATEGORY_COLORS[cat] || '#6b7280' }

/** Show "₽" symbol for the current currency, preserved across the page. */
function MoneySpan({ amount, size = 'md', className = '', withSymbol = true }: {
  amount: string; size?: 'sm' | 'md' | 'lg'; className?: string; withSymbol?: boolean
}) {
  const { hideBalance, symbol } = useSettings()
  if (hideBalance) return <span className={`font-mono-money ${className}`} style={{ color: 'var(--text-tertiary)' }}>••••</span>
  const sizes = { sm: 'text-sm', md: 'text-base', lg: 'text-xl' }
  return <span className={`font-mono-money font-semibold ${sizes[size]} ${className}`}>{formatMoney(amount)}{withSymbol ? ` ${symbol}` : ''}</span>
}

export function BudgetsPage() {
  const { data: budgets, isLoading: budgetsLoading } = useBudgets()
  const { data: categories } = useCategories()
  const createBudget = useCreateBudget()
  const deleteBudget = useDeleteBudget()
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth())

  // Fetch each budget's status in parallel. budgets?.id share the ['budgetStatus', id] queryKey with useBudgetStatus, so caches are reused and there's no duplicate network.
  const list = budgets ?? []
  const statusQueries = useQueries({
    queries: list.map(b => ({
      queryKey: ['budgetStatus', b.id],
      queryFn: () => {/* imported lazily below */ return import('../api/budgets').then(m => m.getBudgetStatus(b.id))},
      staleTime: 30_000,
    })),
  })

  // Map budget_id → BudgetStatus (only ready ones)
  const statusMap = useMemo(() => {
    const m = new Map<number, BudgetStatus>()
    statusQueries.forEach((q, i) => {
      if (q.data && list[i]) m.set(list[i].id, q.data)
    })
    return m
  }, [statusQueries, list])

  const totalLimit = useMemo(() => list.reduce((acc: Decimal, b) => acc.plus(toDecimal(b.limit_amount)), M.zero()), [list])
  const totalSpent = useMemo(() => {
    return list.reduce((acc: Decimal, b) => {
      const st = statusMap.get(b.id)
      return acc.plus(st ? toDecimal(st.spent) : M.zero())
    }, M.zero())
  }, [list, statusMap])

  // Add form
  const [showAdd, setShowAdd] = useState(false)
  const [categoryId, setCategoryId] = useState<number | null>(null)
  const [limit, setLimit] = useState('')

  const reset = () => { setCategoryId(null); setLimit('') }
  const openAdd = () => { reset(); setShowAdd(true) }

  const handleSave = async () => {
    const limDec = M.fromInput(limit)
    if (categoryId === null || !M.isPositive(limDec)) return
    if (createBudget.isPending) return
    try {
      await createBudget.mutateAsync({ category_id: categoryId, limit_amount: limDec.toFixed(2), rollover_policy: 'none' })
      setShowAdd(false); reset()
    } catch (e) {
      console.error('createBudget failed', e)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Удалить бюджет?')) return
    if (deleteBudget.isPending) return
    try { await deleteBudget.mutateAsync(id) } catch (e) { console.error('deleteBudget failed', e) }
  }

  // Category picker options: union of budget categories and categories list
  const budgetCats = useMemo(() => {
    const used = new Set<string>()
    list.forEach(b => { if (b.category_name) used.add(b.category_name) })
    const fromCats = (categories ?? []).filter(c => c.type === 'expense').map(c => c.name)
    fromCats.forEach(n => used.add(n))
    return Array.from(used).sort()
  }, [list, categories])

  // Resolve category name for a budget (prefer server-provided, fallback category list lookup)
  const categoryName = (b: Budget): string => {
    if (b.category_name) return b.category_name
    const cat = (categories ?? []).find(c => c.id === b.category_id)
    return cat?.name || 'Прочее'
  }

  // Inline style for progress bar color based on pct
  const barStyleFor = (pct: number): React.CSSProperties => {
    if (pct > 100) return { background: 'var(--color-danger-500)' }
    if (pct > 80) return { background: 'var(--color-warning-500)' }
    return { background: 'var(--color-primary-500)' }
  }

  return (
    <div className="space-y-4">
      {/* Month selector */}
      <div className="flex gap-1 overflow-x-auto scrollbar-hide -mx-4 px-4">
        {Array.from({ length: 12 }).map((_, i) => (
          <button key={i} onClick={() => setSelectedMonth(i)}
            className={`flex-shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${selectedMonth === i ? 'text-white' : ''}`}
            style={selectedMonth === i
              ? { background: 'var(--color-primary-500)', color: 'white' }
              : { background: 'var(--bg-surface)', color: 'var(--text-secondary)' }}>
            {MONTHS_RU[i]}
          </button>
        ))}
      </div>

      {/* Summary */}
      <div className="bg-gradient-primary rounded-2xl p-5 text-white">
        <p className="text-sm text-white/80 mb-1">Бюджет {MONTHS_RU[selectedMonth]}</p>
        <p className="text-2xl font-bold font-mono-money">{formatMoney(totalSpent)}</p>
        <div className="w-full bg-white/20 rounded-full h-2 mt-3">
          <div className="bg-white h-2 rounded-full transition-all" style={{ width: `${Math.min(totalLimit.gt(0) ? Number(totalSpent.div(totalLimit).mul(100)) : 0, 100)}%` }} />
        </div>
        <p className="text-xs text-white/70 mt-2">из {formatMoney(totalLimit)} • {totalLimit.gt(0) ? Math.round(Number(totalSpent.div(totalLimit).mul(100))) : 0}%</p>
        {totalSpent.gt(totalLimit) && (
          <div className="flex items-center gap-1 mt-2 text-xs text-red-200"><AlertTriangle size={12} /> Превышение бюджета</div>
        )}
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
          Категории{budgetsLoading && <Loader2 className="inline ml-1 animate-spin" size={12} />}
        </h2>
        <button onClick={openAdd} className="flex items-center gap-1 text-sm font-medium" style={{ color: 'var(--color-primary-600)' }}><Plus size={16} /> Добавить</button>
      </div>

      <div className="space-y-2">
        {list.length === 0 && !budgetsLoading ? (
          <Card className="py-8 text-center"><p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Нет бюджетов</p></Card>
        ) : (
          <AnimatePresence mode="popLayout">
            {list.map(b => {
              const st = statusMap.get(b.id)
              const catName = categoryName(b)
              const catColor = colorFor(catName)
              const pct = st ? (Number(st.effective_limit) > 0 ? (Number(st.spent) / Number(st.effective_limit)) * 100 : 0) : 0
              const spentStr = st ? st.spent : '0'
              const remaining = st ? M.sub(toDecimal(st.effective_limit), toDecimal(st.spent)) : toDecimal(b.limit_amount)
              return (
                <motion.div
                  key={b.id}
                  layout
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, x: -16 }}
                  transition={{ duration: 0.2 }}
                  className="card p-4 w-full"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ backgroundColor: catColor }} />
                      <span className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>{catName}</span>
                    </div>
                    <div className="text-right flex-shrink-0 ml-2">
                      <MoneySpan amount={spentStr} size="sm" className={pct > 100 ? 'text-red-500' : ''} />
                      <p className="text-[10px]" style={{ color: 'var(--text-tertiary)' }}>из {formatMoney(b.limit_amount)}</p>
                    </div>
                  </div>
                  <div className="w-full rounded-full h-2" style={{ background: 'var(--border-default)' }}>
                    <div className="h-2 rounded-full transition-all" style={{ width: `${Math.min(pct, 100)}%`, ...barStyleFor(pct) }} />
                  </div>
                  <div className="flex justify-between text-[10px] mt-1" style={{ color: 'var(--text-tertiary)' }}>
                    <span>{pct > 0 ? `~${Math.round(pct)}%` : '—'}{st && st.period_to ? ` • ${Math.max(0, Math.ceil((new Date(st.period_to).getTime() - Date.now()) / 86400000))} дн.` : ''}</span>
                    {pct > 100 && st && <span style={{ color: 'var(--color-danger-500)' }}>+{formatMoney(M.sub(toDecimal(st.spent), toDecimal(b.limit_amount)))}</span>}
                  </div>
                  <div className="flex gap-2 mt-2">
                    <button onClick={() => handleDelete(b.id)}
                      disabled={deleteBudget.isPending}
                      className="ml-auto py-1 px-2 rounded-lg transition-colors disabled:opacity-50"
                      style={{ color: 'var(--text-tertiary)' }}
                      onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = 'var(--color-danger-500)'; (e.currentTarget as HTMLButtonElement).style.background = 'var(--color-danger-50)' }}
                      onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-tertiary)'; (e.currentTarget as HTMLButtonElement).style.background = 'transparent' }}>
                      <Trash2 size={14} />
                    </button>
                  </div>
                </motion.div>
              )
            })}
          </AnimatePresence>
        )}
      </div>

      <BottomSheet open={showAdd} onClose={() => { setShowAdd(false); reset() }} title="Новый бюджет">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2" style={{ color: 'var(--text-primary)' }}>Категория</label>
            {budgetCats.length === 0 ? (
              <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>Нет категорий. Создайте операции с категориями или загрузите категории.</p>
            ) : (
              <div className="flex flex-wrap gap-1.5 max-h-40 overflow-y-auto">
                {budgetCats.slice(0, 30).map(cat => {
                  const catId = (categories ?? []).find(c => c.name === cat)?.id || null
                  const active = categoryId === catId && catId !== null
                  return (
                    <button key={cat} onClick={() => setCategoryId(catId)}
                      className="px-3 py-1.5 rounded-full text-xs font-medium transition-colors"
                      style={active
                        ? { background: 'var(--color-primary-500)', color: 'white' }
                        : { background: 'var(--bg-surface)', color: 'var(--text-secondary)' }}>
                      {cat}
                    </button>
                  )
                })}
              </div>
            )}
          </div>
          <Input label="Месячный лимит (₽)" type="number" step="0.01" min="0.01" value={limit} onChange={e => setLimit(e.target.value)} placeholder="30000" />
          <div className="flex gap-3 pt-2">
            <button onClick={handleSave} disabled={createBudget.isPending || categoryId === null || !M.isPositive(M.fromInput(limit))}
              className="flex-1 py-2.5 rounded-xl bg-primary-500 text-white font-medium text-sm hover:bg-primary-600 transition-colors btn-press disabled:opacity-50">
              {createBudget.isPending ? 'Добавление…' : 'Добавить'}
            </button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}