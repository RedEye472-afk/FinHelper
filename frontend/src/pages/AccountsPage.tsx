import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Wallet, Plus, Building2, PiggyBank, Landmark, CreditCard, Coins, Trash2, Loader2 } from 'lucide-react'
import { useAccounts, useCreateAccount, useDeleteAccount } from '../api/queries'
import { useToast } from '../components/ui/Toast'
import { Input } from '../components/ui/Input'
import { BottomSheet } from '../components/ui/BottomSheet'
import { formatMoney, sumMoney, toDecimal } from '../lib/money'
import type { Account } from '../types'

const accountConfig: Record<string, { icon: typeof Wallet; label: string; color: string }> = {
  cash: { icon: Wallet, label: 'Наличные', color: '#22C55E' },
  bank: { icon: Building2, label: 'Расчётный', color: '#3B82F6' },
  savings: { icon: PiggyBank, label: 'Сберегательный', color: '#F59E0B' },
  investment: { icon: Landmark, label: 'Инвестиционный', color: '#8B5CF6' },
  crypto: { icon: CreditCard, label: 'Крипто', color: '#F97316' },
  debt: { icon: Coins, label: 'Долговой', color: '#F43F5E' },
}

const cardVariant = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.25, ease: 'easeOut' as const } },
}

export function AccountsPage() {
  const { data: accounts = [], isLoading } = useAccounts()
  const createAcc = useCreateAccount()
  const deleteAcc = useDeleteAccount()
  const { toast } = useToast()

  const [showAdd, setShowAdd] = useState(false)
  const [name, setName] = useState('')
  const [type, setType] = useState('bank')
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const reset = () => { setName(''); setType('bank') }
  const totalBalance = sumMoney(accounts.map(a => a.balance))

  const handleSave = () => {
    if (!name.trim()) { toast('error', 'Введите название счёта'); return }
    createAcc.mutate({ name: name.trim(), type, currency: 'RUB' }, {
      onSuccess: () => { toast('success', 'Счёт создан'); setShowAdd(false); reset() },
      onError: (err) => toast('error', err instanceof Error ? err.message : 'Ошибка создания'),
    })
  }

  const handleDelete = (id: number) => {
    setDeleteId(id)
    deleteAcc.mutate(id, {
      onSuccess: () => { toast('success', 'Счёт удалён'); setDeleteId(null) },
      onError: (err) => { toast('error', err instanceof Error ? err.message : 'Ошибка удаления'); setDeleteId(null) },
    })
  }

  return (
    <motion.div className="space-y-4" initial="hidden" animate="show" variants={{ show: { transition: { staggerChildren: 0.05 } } }}>
      {/* Summary */}
      <motion.div variants={cardVariant}>
        <div className="relative overflow-hidden rounded-[20px] border p-5"
          style={{
            background: 'linear-gradient(135deg, #141A2D, #1E293B)',
            borderColor: 'rgba(255,255,255,0.06)',
          }}>
          <div className="relative">
            <p className="text-xs font-medium mb-1" style={{ color: '#94A3B8' }}>Общий баланс</p>
            <p className="text-2xl font-bold font-mono-money tracking-tight">{formatMoney(totalBalance)}</p>
            <p className="text-xs mt-1" style={{ color: '#64748B' }}>{accounts.length} {accounts.length === 1 ? 'счёт' : 'счетов'}</p>
          </div>
        </div>
      </motion.div>

      {/* Add button */}
      <motion.div variants={cardVariant}>
        <button onClick={() => { reset(); setShowAdd(true) }}
          className="w-full py-3 rounded-[16px] font-medium text-sm flex items-center justify-center gap-2 transition-all btn-press"
          style={{
            background: 'rgba(110,86,207,0.1)',
            color: '#A78BFA',
            border: '1px dashed rgba(110,86,207,0.3)',
          }}>
          <Plus size={18} /> Новый счёт
        </button>
      </motion.div>

      {/* Account list */}
      {isLoading ? (
        <div className="flex items-center justify-center h-32">
          <Loader2 className="animate-spin" size={24} style={{ color: '#A78BFA' }} />
        </div>
      ) : accounts.length === 0 ? (
        <motion.div variants={cardVariant} className="rounded-[20px] border p-8 text-center"
          style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <div className="w-12 h-12 rounded-2xl flex items-center justify-center mx-auto mb-3"
            style={{ background: 'rgba(110,86,207,0.1)' }}>
            <Wallet size={24} style={{ color: '#A78BFA' }} />
          </div>
          <p className="text-sm mb-1" style={{ color: '#94A3B8' }}>Нет счетов</p>
          <p className="text-xs" style={{ color: '#64748B' }}>Создайте первый счёт чтобы начать</p>
        </motion.div>
      ) : (
        <div className="space-y-2">
          <AnimatePresence mode="popLayout">
            {accounts.map((a, i) => {
              const cfg = accountConfig[a.account_type] || accountConfig.bank
              const Icon = cfg.icon
              return (
                <motion.div key={a.id} layout
                  initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 40, scale: 0.95 }}
                  transition={{ delay: i * 0.03 }}
                  className="rounded-[16px] border p-4 card-hover"
                  style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
                      style={{ background: `${cfg.color}18`, color: cfg.color }}>
                      <Icon size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>{a.name}</p>
                      <p className="text-xs" style={{ color: '#64748B' }}>{cfg.label}</p>
                    </div>
                    <div className="text-right shrink-0">
                      <p className="text-base font-bold font-mono-money tracking-tight">
                        {formatMoney(toDecimal(a.balance))}
                      </p>
                    </div>
                    <button onClick={() => handleDelete(a.id)}
                      disabled={deleteId === a.id}
                      className="p-2 rounded-lg transition-all btn-press disabled:opacity-50"
                      style={{ color: '#64748B' }}>
                      {deleteId === a.id ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                    </button>
                  </div>
                </motion.div>
              )
            })}
          </AnimatePresence>
        </div>
      )}

      {/* Add account sheet */}
      <BottomSheet open={showAdd} onClose={() => { setShowAdd(false); reset() }} title="Новый счёт">
        <div className="space-y-4">
          <Input
            label="Название счёта"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Наличные, Зарплатная карта..."
            autoFocus
          />

          <div>
            <label className="block text-sm font-medium mb-2" style={{ color: 'var(--text-primary)' }}>Тип счёта</label>
            <div className="grid grid-cols-2 gap-2">
              {Object.entries(accountConfig).map(([key, cfg]) => {
                const Icon = cfg.icon
                const active = type === key
                return (
                  <button key={key} onClick={() => setType(key)}
                    className="flex items-center gap-2.5 p-3 rounded-xl text-sm font-medium transition-all btn-press"
                    style={active
                      ? { background: `${cfg.color}18`, border: `1px solid ${cfg.color}40`, color: cfg.color }
                      : { background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)', color: '#94A3B8' }
                    }>
                    <Icon size={18} />
                    {cfg.label}
                  </button>
                )
              })}
            </div>
          </div>

          <button onClick={handleSave} disabled={createAcc.isPending || !name.trim()}
            className="w-full py-3 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 disabled:opacity-50 btn-press"
            style={{
              background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
              color: '#fff',
            }}>
            {createAcc.isPending ? <><Loader2 size={16} className="animate-spin" /> Создание...</> : 'Создать счёт'}
          </button>
        </div>
      </BottomSheet>
    </motion.div>
  )
}
