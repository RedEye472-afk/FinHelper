import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Loader2, AlertCircle } from 'lucide-react'
import { useCreateOperation, useAccounts, useCategories } from '../api/queries'
import { safeParse } from '../lib/money'
import type { OperationCreate } from '../types'

type OpType = 'income' | 'expense' | 'transfer' | 'fee' | 'correction'

const OP_TYPES: { value: OpType; label: string }[] = [
  { value: 'expense', label: 'Расход' },
  { value: 'income', label: 'Доход' },
  { value: 'transfer', label: 'Перевод' },
  { value: 'fee', label: 'Комиссия' },
  { value: 'correction', label: 'Коррекция' },
]

interface FormState {
  account_id: string
  type: OpType
  amount: string
  date: string
  description: string
  category_id: string
}

export function OperationsNewPage() {
  const navigate = useNavigate()
  const createOp = useCreateOperation()
  const accountsQuery = useAccounts()
  const categoriesQuery = useCategories()

  const accounts = accountsQuery.data ?? []
  const categories = categoriesQuery.data ?? []

  const [form, setForm] = useState<FormState>({
    account_id: '',
    type: 'expense',
    amount: '',
    date: new Date().toISOString().split('T')[0],
    description: '',
    category_id: '',
  })
  const [error, setError] = useState('')

  const update = (field: keyof FormState, value: string) =>
    setForm(f => ({ ...f, [field]: value }))

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!form.account_id) {
      setError('Выберите счёт')
      return
    }
    // Валидация суммы через decimal.js — без parseFloat.
    const amount = safeParse(form.amount)
    if (!amount.isPositive()) {
      setError('Введите корректную сумму')
      return
    }

    const payload: OperationCreate = {
      account_id: parseInt(form.account_id, 10),
      type: form.type,
      amount: form.amount,
      operation_date: form.date,
      description: form.description,
      category_id: form.category_id ? parseInt(form.category_id, 10) : undefined,
      calc_id: crypto.randomUUID(),
    }

    createOp.mutate(payload, {
      onSuccess: () => navigate('/operations'),
      onError: (err: unknown) => {
        const message =
          err instanceof Error && err.message ? err.message : 'Не удалось создать операцию'
        setError(message)
      },
    })
  }

  const submitting = createOp.isPending

  const inputStyle: React.CSSProperties = {
    background: 'var(--bg-surface)',
    borderColor: 'var(--border-default)',
    color: 'var(--text-primary)',
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <h1 className="text-lg font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>Новая операция</h1>

      <form onSubmit={handleSubmit} className="card p-5 space-y-4">
        {error && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            className="flex items-start gap-2 text-sm border rounded-xl px-3 py-2.5"
            style={{
              color: 'var(--color-danger-500)',
              background: 'var(--color-danger-50)',
              borderColor: 'var(--color-danger-200)',
            }}
          >
            <AlertCircle size={16} className="mt-0.5 flex-shrink-0" />
            <span>{error}</span>
          </motion.div>
        )}

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Счёт</label>
          <select
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.account_id}
            onChange={e => update('account_id', e.target.value)}
            required
          >
            <option value="">Выберите счёт</option>
            {accounts.map(a => (
              <option key={a.id} value={a.id}>
                {a.name} ({a.balance} ₽)
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Тип</label>
          <select
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.type}
            onChange={e => update('type', e.target.value)}
          >
            {OP_TYPES.map(t => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Сумма (₽)</label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            autoFocus
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.amount}
            onChange={e => update('amount', e.target.value)}
            placeholder="1000.00"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Дата</label>
          <input
            type="date"
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.date}
            onChange={e => update('date', e.target.value)}
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Описание</label>
          <input
            type="text"
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.description}
            onChange={e => update('description', e.target.value)}
            placeholder="Продукты, аренда..."
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-secondary)' }}>Категория</label>
          <select
            className="input w-full rounded-xl px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/30"
            style={inputStyle}
            value={form.category_id}
            onChange={e => update('category_id', e.target.value)}
          >
            <option value="">Без категории</option>
            {categories.map(c => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>

        <motion.button
          type="submit"
          disabled={submitting}
          whileTap={{ scale: 0.98 }}
          className="w-full py-2.5 rounded-xl bg-primary-500 text-white font-medium text-sm hover:bg-primary-600 transition-colors btn-press disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {submitting && <Loader2 size={16} className="animate-spin" />}
          {submitting ? 'Сохранение...' : 'Создать'}
        </motion.button>
      </form>
    </motion.div>
  )
}