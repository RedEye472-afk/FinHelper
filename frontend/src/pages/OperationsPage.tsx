/**
 * OperationsPage — banking-style transaction list (T-Bank inspired)
 * Premium Dark Neon: bg #0B1020, surface #141A2D, primary #6E56CF, radius 20px cards
 */
import { useState, useMemo, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Search, Plus, X, Inbox, Loader2,
  RotateCw, Trash2, Calendar,
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { useOperations, useDeleteOperation, useAccounts, useCategories } from '../api/queries'
import { useToast } from '../components/ui/Toast'
import { toDecimal, formatMoney } from '../lib/money'
import type { Operation, Category, Account } from '../types'

const categoryColors: Record<string, string> = {
  'Зарплата': '#10b981', 'Фриланс': '#34d399', 'Продукты': '#f59e0b',
  'Транспорт': '#3b82f6', 'Рестораны': '#f97316', 'Жильё': '#ef4444',
  'Развлечения': '#8b5cf6', 'Здоровье': '#ec4899', 'Связь': '#06b6d4',
  'Одежда': '#6366f1', 'Образование': '#14b8a6', 'Подарки': '#f43f5e',
  'Спорт': '#a855f7', 'Подписки': '#0ea5e9', 'Переводы': '#64748b', 'Прочее': '#6b7280',
}

interface OperationRow {
  id: number
  date: string
  category: string
  description: string
  amount: string
  type: string
  account: string
}

function toRow(op: Operation, accounts: Account[], categories: Category[]): OperationRow {
  const acc = accounts.find(a => a.id === op.account_id)
  const cat = categories.find(c => c.id === op.category_id)
  return {
    id: op.id,
    date: op.operation_date,
    category: cat?.name || 'Прочее',
    description: op.description || 'Без описания',
    amount: op.amount,
    type: op.type,
    account: acc?.name || 'Основной',
  }
}

// ── Animation variants ──
const containerVariants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.035 },
  },
}

const itemVariants = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.25, ease: 'easeOut' as const } },
}

const sheetOverlayVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.2 } },
  exit: { opacity: 0, transition: { duration: 0.15 } },
}

const sheetPanelVariants = {
  hidden: { y: '100%', opacity: 0 },
  visible: {
    y: 0,
    opacity: 1,
    transition: { type: 'spring' as const, damping: 28, stiffness: 300, mass: 0.9 },
  },
  exit: {
    y: '100%',
    opacity: 0,
    transition: { duration: 0.18, ease: 'easeIn' as const },
  },
} as const

export function OperationsPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { toast } = useToast()
  const [search, setSearch] = useState(searchParams.get('q') || '')
  const [filterCat, setFilterCat] = useState(searchParams.get('cat') || '')
  const [selectedOp, setSelectedOp] = useState<OperationRow | null>(null)

  const { data: opsData, isLoading } = useOperations(200)
  const { data: accounts = [] } = useAccounts()
  const { data: categories = [] } = useCategories()
  const deleteOp = useDeleteOperation()

  const operations = useMemo<OperationRow[]>(() => {
    if (!opsData?.items) return []
    return opsData.items.map(op => toRow(op, accounts, categories))
  }, [opsData, accounts, categories])

  const setSearchWithURL = useCallback((val: string) => {
    setSearch(val)
    const params = new URLSearchParams(searchParams)
    if (val) params.set('q', val); else params.delete('q')
    setSearchParams(params, { replace: true })
  }, [searchParams, setSearchParams])

  const setFilterWithURL = useCallback((cat: string) => {
    setFilterCat(cat)
    const params = new URLSearchParams(searchParams)
    if (cat) params.set('cat', cat); else params.delete('cat')
    setSearchParams(params, { replace: true })
  }, [searchParams, setSearchParams])

  const cats = useMemo(() => [...new Set(operations.map(o => o.category))], [operations])

  const filtered = useMemo(() => {
    return operations.filter(o => {
      if (search && !o.description.toLowerCase().includes(search.toLowerCase()) && !o.category.toLowerCase().includes(search.toLowerCase())) return false
      if (filterCat && o.category !== filterCat) return false
      return true
    })
  }, [operations, search, filterCat])

  const grouped = useMemo(() => {
    const g: Record<string, OperationRow[]> = {}
    filtered.forEach(o => { (g[o.date] ||= []).push(o) })
    return g
  }, [filtered])

  const handleDelete = useCallback((id: number) => {
    if (!confirm('Удалить операцию?')) return
    deleteOp.mutate(id, {
      onSuccess: () => { toast('success', 'Операция удалена'); setSelectedOp(null) },
      onError: () => toast('error', 'Ошибка удаления'),
    })
  }, [deleteOp, toast])

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
      }}
    >
      {/* ── Pill Search ── */}
      <div style={{ position: 'relative' }}>
        <Search
          size={16}
          style={{
            position: 'absolute',
            left: 16,
            top: '50%',
            transform: 'translateY(-50%)',
            color: 'var(--text-tertiary)',
            pointerEvents: 'none',
            zIndex: 1,
          }}
        />
        <input
          type="text"
          value={search}
          onChange={e => setSearchWithURL(e.target.value)}
          placeholder="Поиск операций..."
          style={{
            width: '100%',
            height: 44,
            padding: '0 40px',
            borderRadius: 22,
            border: '1px solid var(--border-default)',
            background: 'var(--bg-surface)',
            color: 'var(--text-primary)',
            fontSize: 14,
            outline: 'none',
            transition: 'border-color 0.2s, box-shadow 0.2s',
            boxSizing: 'border-box',
          }}
          onFocus={e => {
            e.currentTarget.style.borderColor = '#6E56CF'
            e.currentTarget.style.boxShadow = '0 0 0 3px rgba(110,86,207,0.15)'
          }}
          onBlur={e => {
            e.currentTarget.style.borderColor = 'var(--border-default)'
            e.currentTarget.style.boxShadow = 'none'
          }}
        />
        {search && (
          <button
            onClick={() => setSearchWithURL('')}
            style={{
              position: 'absolute',
              right: 12,
              top: '50%',
              transform: 'translateY(-50%)',
              color: 'var(--text-tertiary)',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              padding: 6,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: '50%',
              transition: 'background 0.15s',
            }}
            onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.06)' }}
            onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
          >
            <X size={14} />
          </button>
        )}
      </div>

      {/* ── Filter Chips ── */}
      <div
        style={{
          display: 'flex',
          gap: 8,
          overflowX: 'auto',
          scrollbarWidth: 'none',
          msOverflowStyle: 'none',
          paddingBottom: 2,
        }}
      >
        <button
          onClick={() => setFilterWithURL('')}
          style={{
            flexShrink: 0,
            padding: '7px 16px',
            borderRadius: 100,
            fontSize: 12,
            fontWeight: 600,
            border: 'none',
            cursor: 'pointer',
            transition: 'all 0.2s',
            background: !filterCat ? '#6E56CF' : 'var(--bg-surface)',
            color: !filterCat ? '#fff' : 'var(--text-secondary)',
            whiteSpace: 'nowrap',
          }}
        >
          Все
        </button>
        {cats.map(cat => {
          const active = filterCat === cat
          return (
            <button
              key={cat}
              onClick={() => setFilterWithURL(cat === filterCat ? '' : cat)}
              style={{
                flexShrink: 0,
                padding: '7px 16px',
                borderRadius: 100,
                fontSize: 12,
                fontWeight: 600,
                border: 'none',
                cursor: 'pointer',
                transition: 'all 0.2s',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                background: active ? '#6E56CF' : 'var(--bg-surface)',
                color: active ? '#fff' : 'var(--text-secondary)',
                whiteSpace: 'nowrap',
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  backgroundColor: categoryColors[cat] || '#6b7280',
                  flexShrink: 0,
                }}
              />
              {cat}
            </button>
          )
        })}
      </div>

      {/* ── Content ── */}
      {isLoading ? (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 200 }}>
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ repeat: Infinity, duration: 1, ease: 'linear' }}
          >
            <Loader2 size={24} style={{ color: '#6E56CF' }} />
          </motion.div>
        </div>
      ) : Object.entries(grouped).length === 0 ? (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: 280,
            gap: 12,
            color: 'var(--text-tertiary)',
          }}
        >
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: 16,
              background: 'var(--bg-surface)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Inbox size={24} style={{ color: 'var(--text-tertiary)' }} />
          </div>
          <p style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-secondary)', margin: 0 }}>
            Нет операций
          </p>
          <p style={{ fontSize: 13, margin: 0 }}>
            {search || filterCat ? 'Измените параметры поиска' : 'Создайте первую операцию'}
          </p>
          {!search && !filterCat && (
            <button
              onClick={() => navigate('/operations/new')}
              style={{
                marginTop: 8,
                padding: '10px 24px',
                borderRadius: 12,
                border: 'none',
                background: 'linear-gradient(135deg, #6E56CF 0%, #7C3AED 100%)',
                color: '#fff',
                fontSize: 13,
                fontWeight: 600,
                cursor: 'pointer',
                transition: 'opacity 0.15s',
              }}
              onMouseEnter={e => { e.currentTarget.style.opacity = '0.9' }}
              onMouseLeave={e => { e.currentTarget.style.opacity = '1' }}
            >
              Создать операцию
            </button>
          )}
        </div>
      ) : (
        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="show"
          style={{ display: 'flex', flexDirection: 'column', gap: 20 }}
        >
          <AnimatePresence mode="popLayout">
            {Object.entries(grouped).map(([date, ops]) => {
              const dayTotal = ops.reduce(
                (s, o) => s.plus(o.type === 'expense' ? toDecimal(o.amount).neg() : toDecimal(o.amount)),
                toDecimal('0'),
              )
              const isPositive = dayTotal.gte(0)
              return (
                <motion.div
                  key={date}
                  layout
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.25, ease: 'easeOut' }}
                >
                  {/* ── Date Header ── */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '0 4px 10px 4px',
                    }}
                  >
                    <span
                      style={{
                        fontSize: 12,
                        fontWeight: 600,
                        color: 'var(--text-tertiary)',
                        letterSpacing: '0.01em',
                      }}
                    >
                      {new Date(date + 'T00:00:00').toLocaleDateString('ru-RU', {
                        day: 'numeric',
                        month: 'long',
                      })}
                    </span>
                    <span
                      style={{
                        fontSize: 12,
                        fontWeight: 700,
                        fontFamily: 'system-ui, -apple-system, sans-serif',
                        color: isPositive ? '#22C55E' : '#F43F5E',
                      }}
                    >
                      {isPositive ? '+' : ''}
                      {formatMoney(dayTotal)}
                    </span>
                  </div>

                  {/* ── Day Card ── */}
                  <div
                    style={{
                      background: 'var(--bg-surface)',
                      borderRadius: 20,
                      border: '1px solid var(--border-default)',
                      overflow: 'hidden',
                    }}
                  >
                    {ops.map((op, idx) => (
                      <motion.button
                        key={op.id}
                        variants={itemVariants}
                        onClick={() => setSelectedOp(op)}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 12,
                          width: '100%',
                          padding: '12px 16px',
                          background: 'transparent',
                          border: 'none',
                          cursor: 'pointer',
                          textAlign: 'left',
                          transition: 'background 0.15s',
                          borderTop: idx > 0 ? '1px solid var(--border-default)' : 'none',
                          color: 'inherit',
                          fontFamily: 'inherit',
                        }}
                        onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.03)' }}
                        onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
                      >
                        {/* Category dot container */}
                        <div
                          style={{
                            width: 36,
                            height: 36,
                            borderRadius: 12,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            flexShrink: 0,
                            background:
                              op.type === 'income'
                                ? 'rgba(34,197,94,0.1)'
                                : 'rgba(255,255,255,0.04)',
                          }}
                        >
                          <div
                            style={{
                              width: 8,
                              height: 8,
                              borderRadius: '50%',
                              backgroundColor: categoryColors[op.category] || '#6b7280',
                            }}
                          />
                        </div>

                        {/* Info */}
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <p
                            style={{
                              fontSize: 13,
                              fontWeight: 600,
                              color: 'var(--text-primary)',
                              margin: 0,
                              whiteSpace: 'nowrap',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                            }}
                          >
                            {op.description}
                          </p>
                          <p
                            style={{
                              fontSize: 11,
                              color: 'var(--text-tertiary)',
                              margin: '2px 0 0 0',
                            }}
                          >
                            {op.category}
                          </p>
                        </div>

                        {/* Amount */}
                        <div style={{ textAlign: 'right', flexShrink: 0 }}>
                          <p
                            style={{
                              fontSize: 13,
                              fontWeight: 700,
                              fontFamily: 'system-ui, -apple-system, sans-serif',
                              margin: 0,
                              color: op.type === 'income' ? '#22C55E' : '#F43F5E',
                            }}
                          >
                            {op.type === 'income' ? '+' : '−'}
                            {formatMoney(op.amount)}
                          </p>
                          <p
                            style={{
                              fontSize: 10,
                              color: 'var(--text-tertiary)',
                              margin: '1px 0 0 0',
                            }}
                          >
                            {op.account}
                          </p>
                        </div>
                      </motion.button>
                    ))}
                  </div>
                </motion.div>
              )
            })}
          </AnimatePresence>
        </motion.div>
      )}

      {/* ── FAB ── */}
      <motion.button
        onClick={() => navigate('/operations/new')}
        whileHover={{ scale: 1.06 }}
        whileTap={{ scale: 0.94 }}
        style={{
          position: 'fixed',
          bottom: 80,
          right: 20,
          width: 52,
          height: 52,
          borderRadius: 16,
          border: 'none',
          background: 'linear-gradient(135deg, #6E56CF 0%, #7C3AED 100%)',
          color: '#fff',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          cursor: 'pointer',
          boxShadow: '0 4px 20px rgba(110,86,207,0.4)',
          zIndex: 30,
        }}
      >
        <Plus size={24} strokeWidth={2.5} />
      </motion.button>

      {/* ── Bottom Sheet ── */}
      <AnimatePresence>
        {selectedOp && (
          <motion.div
            key="sheet-overlay"
            variants={sheetOverlayVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
            style={{
              position: 'fixed',
              inset: 0,
              background: 'rgba(0,0,0,0.5)',
              zIndex: 50,
              display: 'flex',
              alignItems: 'flex-end',
              justifyContent: 'center',
            }}
            onClick={() => setSelectedOp(null)}
          >
            <motion.div
              key="sheet-panel"
              variants={sheetPanelVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
              onClick={e => e.stopPropagation()}
              style={{
                width: '100%',
                maxWidth: 420,
                background: 'var(--bg-elevated)',
                borderRadius: '20px 20px 0 0',
                border: '1px solid var(--border-default)',
                borderBottom: 'none',
                padding: '12px 20px 32px',
                display: 'flex',
                flexDirection: 'column',
                gap: 20,
              }}
            >
              {/* Sheet handle */}
              <div
                style={{
                  width: 32,
                  height: 3,
                  borderRadius: 2,
                  background: 'var(--text-tertiary)',
                  opacity: 0.3,
                  margin: '0 auto 4px',
                }}
              />

              {/* Header */}
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                  <span
                    style={{
                      width: 10,
                      height: 10,
                      borderRadius: '50%',
                      backgroundColor: categoryColors[selectedOp.category] || '#6b7280',
                      flexShrink: 0,
                    }}
                  />
                  <span
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--text-tertiary)',
                      letterSpacing: '0.03em',
                      textTransform: 'uppercase',
                    }}
                  >
                    {selectedOp.category}
                  </span>
                </div>
                <p
                  style={{
                    fontSize: 20,
                    fontWeight: 700,
                    color: 'var(--text-primary)',
                    margin: 0,
                    lineHeight: 1.2,
                  }}
                >
                  {selectedOp.description}
                </p>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    marginTop: 6,
                  }}
                >
                  <Calendar size={12} style={{ color: 'var(--text-tertiary)' }} />
                  <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                    {new Date(selectedOp.date + 'T00:00:00').toLocaleDateString('ru-RU', {
                      day: 'numeric',
                      month: 'long',
                      year: 'numeric',
                    })}
                  </span>
                </div>
              </div>

              {/* Amount display */}
              <div
                style={{
                  textAlign: 'center',
                  padding: '16px 0',
                  background: 'var(--bg-surface)',
                  borderRadius: 16,
                }}
              >
                <p
                  style={{
                    fontSize: 28,
                    fontWeight: 700,
                    fontFamily: 'system-ui, -apple-system, sans-serif',
                    margin: 0,
                    color: selectedOp.type === 'income' ? '#22C55E' : '#F43F5E',
                  }}
                >
                  {selectedOp.type === 'income' ? '+ ' : '− '}
                  {formatMoney(selectedOp.amount)}
                </p>
                <p
                  style={{
                    fontSize: 12,
                    color: 'var(--text-tertiary)',
                    margin: '4px 0 0 0',
                  }}
                >
                  {selectedOp.account}
                </p>
              </div>

              {/* Details */}
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 0,
                  fontSize: 13,
                  color: 'var(--text-secondary)',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    padding: '10px 0',
                    borderBottom: '1px solid var(--border-default)',
                  }}
                >
                  <span>Счёт</span>
                  <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                    {selectedOp.account}
                  </span>
                </div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    padding: '10px 0',
                  }}
                >
                  <span>Сумма</span>
                  <span
                    style={{
                      fontWeight: 600,
                      fontFamily: 'system-ui, -apple-system, sans-serif',
                      color: 'var(--text-primary)',
                    }}
                  >
                    {selectedOp.type === 'income' ? '' : '−'}
                    {formatMoney(selectedOp.amount)}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div style={{ display: 'flex', gap: 10 }}>
                <button
                  onClick={() => { setSelectedOp(null); navigate('/operations/new') }}
                  style={{
                    flex: 1,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 8,
                    padding: '13px 0',
                    borderRadius: 14,
                    border: '1px solid var(--border-default)',
                    background: 'var(--bg-surface)',
                    color: 'var(--text-primary)',
                    fontSize: 13,
                    fontWeight: 600,
                    cursor: 'pointer',
                    transition: 'background 0.15s',
                    fontFamily: 'inherit',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.06)' }}
                  onMouseLeave={e => { e.currentTarget.style.background = 'var(--bg-surface)' }}
                >
                  <RotateCw size={14} />
                  Повторить
                </button>
                <button
                  onClick={() => handleDelete(selectedOp.id)}
                  disabled={deleteOp.isPending}
                  style={{
                    flex: 1,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 8,
                    padding: '13px 0',
                    borderRadius: 14,
                    border: 'none',
                    background: 'rgba(244,63,94,0.12)',
                    color: '#F43F5E',
                    fontSize: 13,
                    fontWeight: 600,
                    cursor: deleteOp.isPending ? 'not-allowed' : 'pointer',
                    opacity: deleteOp.isPending ? 0.5 : 1,
                    transition: 'background 0.15s',
                    fontFamily: 'inherit',
                  }}
                  onMouseEnter={e => {
                    if (!deleteOp.isPending) e.currentTarget.style.background = 'rgba(244,63,94,0.2)'
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.background = 'rgba(244,63,94,0.12)'
                  }}
                >
                  <Trash2 size={14} />
                  {deleteOp.isPending ? 'Удаление...' : 'Удалить'}
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
