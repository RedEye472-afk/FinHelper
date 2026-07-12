import { useMemo, useState } from 'react'
import { useQueries } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Plus,
  Trash2,
  AlertTriangle,
  Loader2,
} from 'lucide-react'
import { BottomSheet } from '../components/ui/BottomSheet'
import { Input } from '../components/ui/Input'
import {
  useBudgets,
  useCreateBudget,
  useDeleteBudget,
  useCategories,
} from '../api/queries'
import { getBudgetStatus } from '../api/budgets'
import { formatMoney, toDecimal, M } from '../lib/money'
import type { Decimal } from '../lib/money'
import type { Budget, BudgetStatus } from '../types'

/* ─────────────────────────────────────────── */
/*  Local palette — Banking dark neon accents   */
/* ─────────────────────────────────────────── */
const CATEGORY_COLORS: Record<string, string> = {
  'Продукты': '#f59e0b',
  'Транспорт': '#3b82f6',
  'Рестораны': '#f97316',
  'Жильё': '#ef4444',
  'Развлечения': '#8b5cf6',
  'Здоровье': '#ec4899',
  'Связь': '#06b6d4',
  'Одежда': '#6366f1',
  'Образование': '#14b8a6',
  'Подарки': '#f43f5e',
  'Спорт': '#a855f7',
  'Подписки': '#0ea5e9',
  'Переводы': '#64748b',
  'Прочее': '#6b7280',
}

function colorFor(cat: string): string {
  return CATEGORY_COLORS[cat] || '#6b7280'
}

/* ─────────────────────────────────────────── */
/*  Month labels                                */
/* ─────────────────────────────────────────── */
const MONTHS_RU = [
  'янв', 'фев', 'мар', 'апр', 'май', 'июн',
  'июл', 'авг', 'сен', 'окт', 'ноя', 'дек',
]

/* ─────────────────────────────────────────── */
/*  Progress helpers                            */
/* ─────────────────────────────────────────── */
type ProgressTier = 'low' | 'mid' | 'high' | 'over'

function progressTier(pct: number): ProgressTier {
  if (pct > 100) return 'over'
  if (pct > 80) return 'high'
  if (pct > 50) return 'mid'
  return 'low'
}

const PROGRESS_COLORS: Record<ProgressTier, { track: string; fill: string; glow: string }> = {
  low:  { track: 'rgba(16,185,129,0.15)', fill: '#10b981', glow: 'rgba(16,185,129,0.35)' },
  mid:  { track: 'rgba(245,158,11,0.15)', fill: '#f59e0b', glow: 'rgba(245,158,11,0.35)' },
  high: { track: 'rgba(244,63,94,0.15)',  fill: '#f43f5e', glow: 'rgba(244,63,94,0.35)' },
  over: { track: 'rgba(244,63,94,0.25)',  fill: '#e11d48', glow: 'rgba(244,63,94,0.45)' },
}

function pctOf(b: Budget, st: BudgetStatus | undefined): number {
  if (!st) return 0
  const eff = Number(st.effective_limit)
  return eff > 0 ? (Number(st.spent) / eff) * 100 : 0
}

/* ─────────────────────────────────────────── */
/*  BudgetsPage                                 */
/* ─────────────────────────────────────────── */
export function BudgetsPage() {
  const { data: budgets, isLoading: budgetsLoading } = useBudgets()
  const { data: categories } = useCategories()
  const createBudget = useCreateBudget()
  const deleteBudget = useDeleteBudget()

  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth())

  /* ── budget statuses ── */
  const list = budgets ?? []
  const statusQueries = useQueries({
    queries: list.map(b => ({
      queryKey: ['budgetStatus', b.id],
      queryFn: () => getBudgetStatus(b.id),
      staleTime: 30_000,
    })),
  })

  const statusMap = useMemo(() => {
    const m = new Map<number, BudgetStatus>()
    statusQueries.forEach((q, i) => {
      if (q.data && list[i]) m.set(list[i].id, q.data)
    })
    return m
  }, [statusQueries, list])

  /* ── aggregated totals ── */
  const totalLimit = useMemo(
    () => list.reduce((acc: Decimal, b) => acc.plus(toDecimal(b.limit_amount)), M.zero()),
    [list],
  )
  const totalSpent = useMemo(
    () =>
      list.reduce((acc: Decimal, b) => {
        const st = statusMap.get(b.id)
        return acc.plus(st ? toDecimal(st.spent) : M.zero())
      }, M.zero()),
    [list, statusMap],
  )
  const totalPct = totalLimit.gt(0)
    ? Math.round(Number(totalSpent.div(totalLimit).mul(100)))
    : 0

  /* ── add budget form ── */
  const [showAdd, setShowAdd] = useState(false)
  const [categoryId, setCategoryId] = useState<number | null>(null)
  const [limit, setLimit] = useState('')

  const resetForm = () => {
    setCategoryId(null)
    setLimit('')
  }
  const openAdd = () => {
    resetForm()
    setShowAdd(true)
  }

  const handleSave = async () => {
    const limDec = M.fromInput(limit)
    if (categoryId === null || !M.isPositive(limDec)) return
    if (createBudget.isPending) return
    try {
      await createBudget.mutateAsync({
        category_id: categoryId,
        limit_amount: limDec.toFixed(2),
        rollover_policy: 'none',
      })
      setShowAdd(false)
      resetForm()
    } catch (e) {
      console.error('createBudget failed', e)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Удалить бюджет?')) return
    if (deleteBudget.isPending) return
    try {
      await deleteBudget.mutateAsync(id)
    } catch (e) {
      console.error('deleteBudget failed', e)
    }
  }

  /* ── derived category list for picker ── */
  const budgetCats = useMemo(() => {
    const used = new Set<string>()
    list.forEach(b => {
      if (b.category_name) used.add(b.category_name)
    })
    ;(categories ?? [])
      .filter(c => c.type === 'expense')
      .forEach(c => used.add(c.name))
    return Array.from(used).sort()
  }, [list, categories])

  const categoryName = (b: Budget): string => {
    if (b.category_name) return b.category_name
    const cat = (categories ?? []).find(c => c.id === b.category_id)
    return cat?.name || 'Прочее'
  }

  /* ── render helpers ── */

  const now = Date.now()

  /* ─────────────────────────────────────────── */
  /*  RENDER                                      */
  /* ─────────────────────────────────────────── */
  return (
    <div className="relative min-h-screen px-4 pb-28">
      {/* ── Decorative glow ── */}
      <div
        className="hero-circle-glow"
        style={{
          top: '-5%',
          left: '-10%',
          width: '55%',
          height: '55%',
          background: 'radial-gradient(circle, rgba(110,86,207,0.18) 0%, transparent 70%)',
        }}
      />

      {/* ── Hero header card ── */}
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: [0.16, 1, 0.3, 1] }}
        className="relative overflow-hidden rounded-2xl p-6 mt-4"
        style={{ background: 'var(--bg-gradient)' }}
      >
        {/* Shine overlay */}
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            background:
              'linear-gradient(135deg, rgba(255,255,255,0.12) 0%, transparent 50%, rgba(255,255,255,0.04) 100%)',
          }}
        />

        <div className="relative z-10">
          <p className="text-xs font-medium tracking-wide uppercase opacity-70">
            Бюджет {MONTHS_RU[selectedMonth]}
          </p>
          <p className="text-3xl font-bold font-mono-money mt-1 tracking-tight">
            {formatMoney(totalSpent)}
          </p>

          {/* Overall progress */}
          <div className="mt-4">
            <div
              className="w-full rounded-full overflow-hidden"
              style={{ background: 'rgba(255,255,255,0.15)' }}
            >
              <motion.div
                className="h-2 rounded-full"
                initial={{ width: 0 }}
                animate={{ width: `${Math.min(totalPct, 100)}%` }}
                transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
                style={{ background: 'rgba(255,255,255,0.85)' }}
              />
            </div>
          </div>

          <div className="flex items-center justify-between mt-2">
            <p className="text-xs opacity-65">
              из {formatMoney(totalLimit)}
              <span className="ml-1.5">•</span>
              <span className="ml-1.5">{totalPct}%</span>
            </p>

            {totalSpent.gt(totalLimit) && (
              <div className="flex items-center gap-1 text-xs font-medium text-red-200 bg-black/20 rounded-full px-2.5 py-0.5">
                <AlertTriangle size={11} />
                Превышение
              </div>
            )}
          </div>
        </div>
      </motion.div>

      {/* ── Month chips ── */}
      <div className="flex gap-1.5 overflow-x-auto scrollbar-hide mt-5 -mx-4 px-4">
        {Array.from({ length: 12 }).map((_, i) => (
          <button
            key={i}
            onClick={() => setSelectedMonth(i)}
            className="flex-shrink-0 px-3.5 py-2 rounded-full text-xs font-medium transition-all duration-200 border"
            style={
              selectedMonth === i
                ? {
                    background: 'var(--color-primary-500)',
                    color: '#fff',
                    borderColor: 'var(--color-primary-500)',
                    boxShadow: '0 0 12px rgba(139,92,246,0.35)',
                  }
                : {
                    background: 'var(--bg-surface)',
                    color: 'var(--text-secondary)',
                    borderColor: 'var(--border-default)',
                  }
            }
          >
            {MONTHS_RU[i]}
          </button>
        ))}
      </div>

      {/* ── Section header ── */}
      <div className="flex items-center justify-between mt-6 mb-3">
        <h2 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
          Категории
          {budgetsLoading && <Loader2 className="inline ml-1.5 animate-spin" size={12} />}
        </h2>
        <button
          onClick={openAdd}
          className="flex items-center gap-1 text-sm font-medium transition-opacity hover:opacity-80"
          style={{ color: 'var(--color-primary-400)' }}
        >
          <Plus size={15} />
          Добавить
        </button>
      </div>

      {/* ── Category budget cards ── */}
      <div className="space-y-2.5">
        {list.length === 0 && !budgetsLoading ? (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="rounded-2xl py-12 text-center border border-dashed"
            style={{
              background: 'var(--bg-surface)',
              borderColor: 'var(--border-default)',
            }}
          >
            <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
              Нет бюджетов
            </p>
            <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
              Нажмите «Добавить», чтобы установить лимит на категорию
            </p>
          </motion.div>
        ) : (
          <AnimatePresence mode="popLayout">
            {list.map(b => {
              const st = statusMap.get(b.id)
              const catName = categoryName(b)
              const catColor = colorFor(catName)
              const pct = pctOf(b, st)

              const tier = progressTier(pct)
              const barColors = PROGRESS_COLORS[tier]
              const spentStr = st ? st.spent : '0'
              const remaining = st
                ? M.sub(toDecimal(st.effective_limit), toDecimal(st.spent))
                : toDecimal(b.limit_amount)

              const daysLeft =
                st && st.period_to
                  ? Math.max(0, Math.ceil((new Date(st.period_to).getTime() - now) / 86400000))
                  : null

              return (
                <motion.div
                  key={b.id}
                  layout
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, x: -20, scale: 0.96 }}
                  transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
                  className="card card-hover p-4"
                >
                  {/* Top row: dot + name + spent */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span
                        className="w-3 h-3 rounded-full flex-shrink-0 ring-2"
                        style={{
                          backgroundColor: catColor,
                          borderColor: `${catColor}40`,
                        }}
                      />
                      <span
                        className="text-sm font-semibold truncate"
                        style={{ color: 'var(--text-primary)' }}
                      >
                        {catName}
                      </span>
                    </div>

                    <div className="flex items-center gap-2 flex-shrink-0 ml-2">
                      <span
                        className="text-sm font-bold font-mono-money"
                        style={{
                          color:
                            tier === 'over'
                              ? 'var(--color-danger-400)'
                              : 'var(--text-primary)',
                        }}
                      >
                        {formatMoney(spentStr)}
                      </span>
                      <span className="text-[10px]" style={{ color: 'var(--text-tertiary)' }}>
                        / {formatMoney(b.limit_amount)}
                      </span>
                    </div>
                  </div>

                  {/* Progress bar */}
                  <div className="mt-2.5">
                    <div
                      className="w-full rounded-full overflow-hidden"
                      style={{ background: barColors.track }}
                    >
                      <motion.div
                        className="h-2 rounded-full relative"
                        initial={{ width: 0 }}
                        animate={{ width: `${Math.min(pct, 100)}%` }}
                        transition={{ duration: 0.6, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
                        style={{
                          background: barColors.fill,
                          boxShadow: `0 0 8px ${barColors.glow}`,
                        }}
                      />
                    </div>
                  </div>

                  {/* Bottom row: metadata + delete */}
                  <div className="flex items-center justify-between mt-1.5">
                    <div className="flex items-center gap-2 text-[10px]" style={{ color: 'var(--text-tertiary)' }}>
                      {pct > 0 ? (
                        <span>~{Math.round(pct)}%</span>
                      ) : (
                        <span>—</span>
                      )}
                      {daysLeft !== null && (
                        <>
                          <span>•</span>
                          <span>{daysLeft} дн.</span>
                        </>
                      )}
                      {tier === 'over' && st && (
                        <>
                          <span>•</span>
                          <span style={{ color: 'var(--color-danger-400)' }}>
                            +{formatMoney(M.sub(toDecimal(st.spent), toDecimal(b.limit_amount)))}
                          </span>
                        </>
                      )}
                      {tier !== 'over' && remaining.gt(0) && (
                        <>
                          <span>•</span>
                          <span style={{ color: barColors.fill }}>
                            осталось {formatMoney(remaining)}
                          </span>
                        </>
                      )}
                    </div>

                    <button
                      onClick={() => handleDelete(b.id)}
                      disabled={deleteBudget.isPending}
                      className="flex-shrink-0 p-1.5 rounded-lg transition-all duration-200 opacity-40 hover:opacity-100 disabled:opacity-20"
                      style={{ color: 'var(--text-tertiary)' }}
                      onMouseEnter={e => {
                        e.currentTarget.style.color = '#fb7185'
                        e.currentTarget.style.background = 'rgba(244,63,94,0.12)'
                      }}
                      onMouseLeave={e => {
                        e.currentTarget.style.color = 'var(--text-tertiary)'
                        e.currentTarget.style.background = 'transparent'
                      }}
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </motion.div>
              )
            })}
          </AnimatePresence>
        )}
      </div>

      {/* ── FAB ── */}
      <motion.button
        onClick={openAdd}
        initial={{ scale: 0, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ delay: 0.3, type: 'spring', stiffness: 300, damping: 20 }}
        whileHover={{ scale: 1.06 }}
        whileTap={{ scale: 0.92 }}
        className="fixed bottom-6 right-6 z-40 w-14 h-14 rounded-full flex items-center justify-center shadow-2xl"
        style={{ background: 'var(--bg-gradient)' }}
      >
        <Plus size={24} className="text-white" />
      </motion.button>

      {/* ── Add budget bottom sheet ── */}
      <BottomSheet
        open={showAdd}
        onClose={() => {
          setShowAdd(false)
          resetForm()
        }}
        title="Новый бюджет"
      >
        <div className="space-y-5">
          {/* Category picker */}
          <div>
            <label
              className="block text-sm font-semibold mb-3"
              style={{ color: 'var(--text-primary)' }}
            >
              Категория
            </label>

            {budgetCats.length === 0 ? (
              <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                Нет категорий. Создайте операции с категориями или загрузите категории.
              </p>
            ) : (
              <div className="flex flex-wrap gap-2 max-h-44 overflow-y-auto scrollbar-hide">
                {budgetCats.slice(0, 30).map(cat => {
                  const catId = (categories ?? []).find(c => c.name === cat)?.id ?? null
                  const active = categoryId === catId && catId !== null
                  const dotColor = colorFor(cat)

                  return (
                    <button
                      key={cat}
                      onClick={() => setCategoryId(catId)}
                      className="flex items-center gap-1.5 px-3.5 py-2 rounded-full text-xs font-medium transition-all duration-200"
                      style={
                        active
                          ? {
                              background: 'rgba(110,86,207,0.2)',
                              color: '#a78bfa',
                              border: '1px solid rgba(110,86,207,0.4)',
                            }
                          : {
                              background: 'var(--bg-surface)',
                              color: 'var(--text-secondary)',
                              border: '1px solid var(--border-default)',
                            }
                      }
                    >
                      <span
                        className="w-2 h-2 rounded-full flex-shrink-0"
                        style={{ background: dotColor }}
                      />
                      {cat}
                    </button>
                  )
                })}
              </div>
            )}
          </div>

          {/* Limit input */}
          <Input
            label="Месячный лимит (₽)"
            type="number"
            step="0.01"
            min="0.01"
            value={limit}
            onChange={e => setLimit(e.target.value)}
            placeholder="30000"
          />

          {/* Actions */}
          <div className="flex gap-3 pt-1">
            <button
              onClick={() => {
                setShowAdd(false)
                resetForm()
              }}
              className="flex-1 py-2.5 rounded-xl text-sm font-medium btn-secondary btn-press"
            >
              Отмена
            </button>
            <button
              onClick={handleSave}
              disabled={
                createBudget.isPending || categoryId === null || !M.isPositive(M.fromInput(limit))
              }
              className="flex-1 py-2.5 rounded-xl text-white text-sm font-semibold btn-press disabled:opacity-40"
              style={{ background: 'var(--bg-gradient)' }}
            >
              {createBudget.isPending ? (
                <span className="flex items-center justify-center gap-1.5">
                  <Loader2 size={14} className="animate-spin" />
                  Добавление…
                </span>
              ) : (
                'Добавить'
              )}
            </button>
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}
