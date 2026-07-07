import { useState, useMemo, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Search, Plus, X, Inbox, Loader2 } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { useOperations, useDeleteOperation, useAccounts, useCategories } from '../api/queries'
import { useToast } from '../components/ui/Toast'
import { MoneyDisplay } from '../components/shared/MoneyDisplay'
import { Card } from '../components/ui/Card'
import { EmptyState } from '../components/ui/EmptyState'
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

  const handleDelete = (id: number) => {
    if (!confirm('Удалить операцию?')) return
    deleteOp.mutate(id, {
      onSuccess: () => { toast('success', 'Операция удалена'); setSelectedOp(null) },
      onError: () => toast('error', 'Ошибка удаления'),
    })
  }

  return (
    <div className="space-y-4">
      <div className="relative">
        <Search size={16} className="absolute left-3.5 top-1/2 -translate-y-1/2" style={{color: 'var(--text-tertiary)'}} />
        <input type="text" value={search} onChange={e => setSearchWithURL(e.target.value)}
          className="input pl-9 pr-9"
          placeholder="Поиск операций..." />
        {search && <button onClick={() => setSearchWithURL('')} className="absolute right-3 top-1/2 -translate-y-1/2" style={{color: 'var(--text-tertiary)'}}><X size={14} /></button>}
      </div>

      <div className="flex gap-2 overflow-x-auto scrollbar-hide -mx-4 px-4">
        <button onClick={() => setFilterWithURL('')} className={`flex-shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-colors btn-press ${!filterCat ? 'text-white' : ''}`}
          style={!filterCat ? {background: 'var(--color-primary-500)', color: 'white'} : {background: 'var(--bg-surface)', color: 'var(--text-secondary)'}}>Все</button>
        {cats.map(cat => (
          <button key={cat} onClick={() => setFilterWithURL(cat === filterCat ? '' : cat)}
            className={`flex-shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-colors btn-press ${filterCat === cat ? 'text-white' : ''}`}
            style={filterCat === cat ? {background: 'var(--color-primary-500)', color: 'white'} : {background: 'var(--bg-surface)', color: 'var(--text-secondary)'}}>
            {cat}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-48"><Loader2 className="animate-spin" style={{color: 'var(--color-primary-500)'}} size={24} /></div>
      ) : Object.entries(grouped).length === 0 ? (
        <EmptyState
          icon={Inbox}
          title="Нет операций"
          description={search || filterCat ? 'Измените параметры поиска' : 'Создайте первую операцию'}
          action={!search && !filterCat ? { label: 'Создать операцию', onClick: () => navigate('/operations/new') } : undefined}
        />
      ) : (
        <div className="space-y-4">
          <AnimatePresence>
            {Object.entries(grouped).map(([date, ops]) => {
              const dayTotal = ops.reduce((s, o) => s.plus(o.type === 'expense' ? toDecimal(o.amount).neg() : toDecimal(o.amount)), toDecimal('0'))
              return (
                <motion.div key={date} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}>
                  <div className="flex items-center justify-between mb-2 px-1">
                    <span className="text-xs font-medium" style={{color: 'var(--text-tertiary)'}}>{new Date(date).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })}</span>
                    <MoneyDisplay amount={dayTotal} type={dayTotal.gte(0) ? 'income' : 'expense'} size="sm" showSign />
                  </div>
                  <Card className="divide-y" style={{borderColor: 'var(--border-subtle)'}}>
                    {ops.map(op => (
                      <button key={op.id} onClick={() => setSelectedOp(op)}
                        className="flex items-center gap-3 py-2.5 w-full text-left hover:bg-gray-50 -mx-4 px-4 transition-colors first:rounded-t-xl last:rounded-b-xl"
                        style={{borderColor: 'var(--border-subtle)'}}>
                        <div className="w-9 h-9 rounded-xl flex items-center justify-center text-sm flex-shrink-0"
                          style={{background: op.type === 'income' ? 'rgba(16,185,129,0.1)' : 'var(--bg-surface)'}}>
                          <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: categoryColors[op.category] || '#6b7280' }} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate" style={{color: 'var(--text-primary)'}}>{op.description}</p>
                          <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{op.category}</p>
                        </div>
                        <div className="text-right flex-shrink-0">
                          <MoneyDisplay amount={op.amount} type={op.type as 'income' | 'expense'} size="sm" showSign />
                          <p className="text-[10px]" style={{color: 'var(--text-tertiary)'}}>{op.account}</p>
                        </div>
                      </button>
                    ))}
                  </Card>
                </motion.div>
              )
            })}
          </AnimatePresence>
        </div>
      )}

      <button onClick={() => navigate('/operations/new')} className="fixed bottom-20 right-5 w-12 h-12 rounded-full bg-gradient-primary text-white shadow-lg flex items-center justify-center transition-all btn-press z-30 animate-float">
        <Plus size={24} />
      </button>

      <AnimatePresence>
        {selectedOp && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/40 z-50 flex items-end sm:items-center justify-center p-4"
            onClick={() => setSelectedOp(null)}>
            <motion.div initial={{ y: 50, scale: 0.95 }} animate={{ y: 0, scale: 1 }} exit={{ y: 50, scale: 0.95 }}
              className="rounded-2xl p-5 w-full max-w-sm space-y-4"
              style={{background: 'var(--bg-elevated)', border: '1px solid var(--border-default)'}}
              onClick={e => e.stopPropagation()}>
              <div>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{selectedOp.category}</p>
                <p className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>{selectedOp.description}</p>
                <p className="text-xs" style={{color: 'var(--text-tertiary)'}}>{new Date(selectedOp.date).toLocaleDateString('ru-RU')}</p>
              </div>
              <div className="text-center py-3">
                <MoneyDisplay amount={selectedOp.amount} type={selectedOp.type as 'income' | 'expense'} size="lg" showSign />
              </div>
              <div className="text-sm space-y-1" style={{color: 'var(--text-secondary)'}}>
                <div className="flex justify-between"><span>Счёт</span><span className="font-medium">{selectedOp.account}</span></div>
                <div className="flex justify-between"><span>Сумма</span><span className="font-mono-money">{formatMoney(selectedOp.amount)}</span></div>
              </div>
              <div className="flex gap-3 pt-2">
                <button onClick={() => navigate('/operations/new')} className="flex-1 py-2.5 rounded-xl font-medium text-sm btn-secondary btn-press">
                  Повторить
                </button>
                <button onClick={() => handleDelete(selectedOp.id)} disabled={deleteOp.isPending}
                  className="flex-1 py-2.5 rounded-xl font-medium text-sm btn-danger btn-press disabled:opacity-50">
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