import { useState } from 'react'
import { Wallet, Plus, ArrowRightLeft, Building2, PiggyBank, Landmark, Loader2 } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { useAccounts, useCreateAccount, useDeleteAccount } from '../api/queries'
import { useToast } from '../components/ui/Toast'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { BottomSheet } from '../components/ui/BottomSheet'
import { MoneyDisplay } from '../components/shared/MoneyDisplay'
import { toDecimal, sumMoney, formatMoney } from '../lib/money'
import type { Account } from '../types'

const accountIcons: Record<string, typeof Wallet> = {
  cash: Wallet, bank: Building2, savings: PiggyBank, investment: Landmark, checking: Building2,
}

export function AccountsPage() {
  const { data: accounts = [], isLoading } = useAccounts()
  const createAcc = useCreateAccount()
  const deleteAcc = useDeleteAccount()
  const { toast } = useToast()
  const [showAdd, setShowAdd] = useState(false)
  const [editA, setEditA] = useState<Account | null>(null)
  const [name, setName] = useState('')
  const [type, setType] = useState('bank')

  const reset = () => { setName(''); setType('bank') }

  const handleSave = () => {
    if (!name) return
    createAcc.mutate({ name, type, currency: 'RUB' }, {
      onSuccess: () => { toast('success', 'Счёт создан'); setShowAdd(false); setEditA(null); reset() },
      onError: (err) => toast('error', err instanceof Error ? err.message : 'Ошибка создания'),
    })
  }

  const handleDelete = (id: number) => {
    if (!confirm('Удалить счёт?')) return
    deleteAcc.mutate(id, { onSuccess: () => toast('success', 'Счёт удалён'), onError: (err) => toast('error', err instanceof Error ? err.message : 'Ошибка удаления') })
  }

  const totalBalance = sumMoney(accounts.map(a => a.balance))

  const typeLabel = (t: string) => t === 'cash' ? 'Наличные' : t === 'savings' ? 'Сберегательный' : t === 'investment' ? 'Инвестиционный' : t === 'debt' ? 'Долговой' : 'Расчётный'

  return (
    <div className="space-y-4">
      <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}
        className="bg-gradient-primary rounded-2xl p-5 text-white">
        <p className="text-sm text-white/80 mb-1">Общий баланс</p>
        <p className="text-2xl font-bold font-mono-money">{formatMoney(totalBalance)}</p>
        <p className="text-xs text-white/70 mt-1">{accounts.length} счетов</p>
      </motion.div>

      <button onClick={() => { reset(); setShowAdd(true) }}
        className="flex items-center gap-1.5 w-full py-2.5 rounded-xl font-medium text-sm transition-colors btn-press"
        style={{ background: 'var(--color-primary-50)', color: 'var(--color-primary-600)' }}>
        <Plus size={16} /> Новый счёт
      </button>

      {isLoading ? (
        <div className="flex items-center justify-center h-48"><Loader2 className="animate-spin" size={24} style={{ color: 'var(--color-primary-500)' }} /></div>
      ) : accounts.length === 0 ? (
        <Card className="text-center py-12">
          <Wallet size={32} className="mx-auto mb-2" style={{ color: 'var(--text-tertiary)' }} />
          <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Нет счетов. Создайте первый!</p>
        </Card>
      ) : (
        <div className="space-y-2">
          <AnimatePresence>
            {accounts.map((a, i) => {
              const Icon = accountIcons[a.account_type] || Wallet
              return (
                <motion.div key={a.id}
                  initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 20 }}
                  transition={{ delay: i * 0.05 }}>
                  <button onClick={() => { setEditA(a); setName(a.name); setType(a.account_type) }}
                    className="card p-4 w-full text-left card-hover">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl flex items-center justify-center"
                        style={{ background: 'var(--color-primary-50)', color: 'var(--color-primary-600)' }}>
                        <Icon size={20} />
                      </div>
                      <div className="flex-1">
                        <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{a.name}</p>
                        <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{typeLabel(a.account_type)}</p>
                      </div>
                      <MoneyDisplay amount={a.balance} size="md" />
                    </div>
                  </button>
                </motion.div>
              )
            })}
          </AnimatePresence>
        </div>
      )}

      <BottomSheet open={showAdd || !!editA} onClose={() => { setShowAdd(false); setEditA(null); reset() }} title={editA ? 'Редактировать счёт' : 'Новый счёт'}>
        <div className="space-y-4">
          <Input label="Название" value={name} onChange={e => setName(e.target.value)} placeholder="Наличные" />
          <div>
            <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-primary)' }}>Тип</label>
            <select value={type} onChange={e => setType(e.target.value)}
              className="input"
              style={{background: 'var(--bg-surface)', color: 'var(--text-primary)'}}>
              <option value="cash" style={{ background: 'var(--bg-elevated)', color: 'var(--text-primary)' }}>Наличные</option>
              <option value="bank" style={{ background: 'var(--bg-elevated)', color: 'var(--text-primary)' }}>Банковский</option>
              <option value="savings" style={{ background: 'var(--bg-elevated)', color: 'var(--text-primary)' }}>Сберегательный</option>
              <option value="investment" style={{ background: 'var(--bg-elevated)', color: 'var(--text-primary)' }}>Инвестиционный</option>
              <option value="debt" style={{ background: 'var(--bg-elevated)', color: 'var(--text-primary)' }}>Долговой</option>
            </select>
          </div>
          <div className="flex gap-3 pt-2">
            <button onClick={handleSave} disabled={createAcc.isPending}
              className="flex-1 py-2.5 rounded-xl bg-primary-500 text-white font-medium text-sm hover:bg-primary-600 transition-colors btn-press disabled:opacity-50">
              {createAcc.isPending ? 'Создание...' : editA ? 'Сохранить' : 'Добавить'}
            </button>
            {editA && <button onClick={() => handleDelete(editA.id)} className="py-2.5 px-4 rounded-xl font-medium text-sm transition-colors btn-press"
              style={{ background: 'var(--color-danger-50)', color: 'var(--color-danger-600)' }}>Удалить</button>}
          </div>
        </div>
      </BottomSheet>
    </div>
  )
}