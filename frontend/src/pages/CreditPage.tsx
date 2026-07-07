import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { CreditCard, ChevronDown, ChevronUp, Loader2, AlertCircle } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { FormulaTooltip } from '../components/shared/FormulaTooltip'
import { AiExplanation } from '../components/shared/AiExplanation'
import { Disclaimer } from '../components/shared/Disclaimer'
import { useCreditCalc } from '../api/queries'
import { formatMoney, formatNumber } from '../lib/money'
import type { CreditResponse } from '../api/calculators'

export default function CreditPage() {
  // Форма — входные строки. Деньги/ставки — строки, без parseFloat/Number.
  const [amount, setAmount] = useState('2000000')
  const [rate, setRate] = useState('21')
  const [term, setTerm] = useState('24')
  const [payType, setPayType] = useState<'annuity' | 'diff'>('annuity')
  const [showSchedule, setShowSchedule] = useState(false)

  const calc = useCreditCalc()
  const result: CreditResponse | undefined = calc.data
  const isLoading = calc.isPending
  const error = calc.error

  const handleCalculate = () => {
    // term_months — это количество месяцев (count), parseInt разрешён.
    const termNum = Number.parseInt(term, 10) || 1
    calc.mutate({
      principal: amount,
      annual_rate: rate,
      term_months: termNum,
      payment_type: payType,
    })
  }

  const fmtMoney = (s: string) => formatMoney(s)
  const fmtNum = (s: string) => formatNumber(s)

  return (
    <div className="space-y-4">
      <motion.div layout>
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 bg-gradient-primary opacity-10" />
          <div className="relative">
            <div className="flex items-center gap-2 mb-1">
              <CreditCard size={18} style={{ color: 'var(--color-danger-500)' }} />
              <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>Ежемесячный платёж</p>
            </div>
            <div className="flex items-baseline gap-2">
              <p className="text-3xl font-bold font-mono-money gradient-text">
                {result ? fmtNum(result.monthly_payment) : '—'}{result ? ' ₽' : ''}
              </p>
            </div>
            <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
              {payType === 'annuity' ? 'Аннуитетный' : 'Дифференцированный'} платёж
            </p>
          </div>
        </Card>
      </motion.div>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Параметры кредита</h3>
        <Input label="Сумма кредита (₽)" type="number" value={amount} onChange={e => setAmount(e.target.value)} />
        <Input label="Годовая ставка (%)" type="number" step="0.1" value={rate} onChange={e => setRate(e.target.value)} />
        <Input label="Срок (мес.)" type="number" value={term} onChange={e => setTerm(e.target.value)} />
        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-primary)' }}>Тип платежа</label>
          <div className="flex gap-2">
            {(['annuity', 'diff'] as const).map(t => (
              <button key={t} onClick={() => setPayType(t)}
                className={`flex-1 py-2.5 rounded-xl text-xs font-medium transition-all ${payType === t ? 'bg-gradient-primary text-white shadow-sm' : ''}`}
                style={payType !== t ? { background: 'var(--bg-surface)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' } : {}}>
                {t === 'annuity' ? 'Аннуитет' : 'Дифференцированный'}
              </button>
            ))}
          </div>
        </div>
        <Button variant="primary" className="w-full" onClick={handleCalculate} disabled={isLoading}>
          {isLoading ? <><Loader2 size={16} className="animate-spin mr-1.5" /> Расчёт…</> : 'Рассчитать'}
        </Button>
      </Card>

      <AnimatePresence mode="wait">
        {error && (
          <motion.div
            key="error"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.2 }}
          >
            <div className="flex items-start gap-2 p-3 border rounded-xl text-xs"
              style={{
                background: 'var(--color-danger-50)',
                borderColor: 'var(--color-danger-200)',
                color: 'var(--color-danger-500)',
              }}>
              <AlertCircle size={14} className="mt-0.5 flex-shrink-0" />
              <p>Не удалось выполнить расчёт. {(error as Error)?.message || 'Проверьте введённые данные и попробуйте снова.'}</p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <AnimatePresence mode="wait">
        {result && !error && (
          <motion.div
            key="result"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -12 }}
            transition={{ duration: 0.25, ease: 'easeOut' }}
            className="space-y-4"
          >
            <div className="grid grid-cols-2 gap-3">
              <motion.div layout><Card><p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Сумма кредита</p><p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtMoney(amount)}</p></Card></motion.div>
              <motion.div layout><Card><p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Платёж в месяц</p><p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtMoney(result.monthly_payment)}</p></Card></motion.div>
              <motion.div layout><Card><p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Переплата</p><p className="text-base font-semibold font-mono-money" style={{ color: 'var(--color-danger-500)' }}>{fmtMoney(result.overpayment)}</p></Card></motion.div>
              <motion.div layout><Card><p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>ПСК (ФЗ-353)</p><p className="text-base font-semibold font-mono-money" style={{ color: 'var(--color-warning-500)' }}>{fmtNum(result.psk)}%</p></Card></motion.div>
            </div>

            <Button variant="secondary" onClick={() => setShowSchedule(!showSchedule)} className="w-full justify-between">
              <span>График платежей</span>
              {showSchedule ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </Button>

            <AnimatePresence>
              {showSchedule && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.2 }}
                  className="overflow-hidden"
                >
                  <div className="overflow-x-auto rounded-xl" style={{ border: '1px solid var(--border-default)' }}>
                    <table className="w-full text-xs">
                      <thead>
                        <tr style={{ background: 'var(--bg-surface)' }}>
                          <th className="px-3 py-2 text-left font-medium" style={{ color: 'var(--text-secondary)' }}>№</th>
                          <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>Платёж</th>
                          <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>Основной долг</th>
                          <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>Проценты</th>
                          <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>Остаток</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y" style={{ borderColor: 'var(--border-subtle)' }}>
                        {result.schedule.map(p => (
                          <tr key={p.month} className="transition-colors hover:bg-gray-50">
                            <td className="px-3 py-1.5" style={{ color: 'var(--text-tertiary)' }}>{p.month}</td>
                            <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtNum(p.payment)}</td>
                            <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtNum(p.principal_part)}</td>
                            <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--color-danger-500)' }}>{fmtNum(p.interest_part)}</td>
                            <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtNum(p.remaining)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            {result.disclaimer && (
              <div className="flex items-start gap-2 p-3 border rounded-xl text-xs"
                style={{
                  background: 'rgba(245,158,11,0.08)',
                  borderColor: 'var(--color-warning-500)',
                  color: 'var(--color-warning-500)',
                }}>
                <AlertCircle size={14} className="mt-0.5 flex-shrink-0" />
                <p>{result.disclaimer}</p>
              </div>
            )}

            <FormulaTooltip
              title="Формула аннуитетного платежа"
              formula="P = S \cdot \frac{r(1+r)^n}{(1+r)^n - 1}"
              explanation="P — ежемесячный платёж, S — сумма кредита, r — месячная ставка, n — число месяцев"
            />
            <AiExplanation>
              <p>Кредитный калькулятор рассчитывает график платежей двумя способами.</p>
              <p><b>Аннуитет:</b> равные платежи весь срок. Сначала большая часть — проценты, к концу — основной долг.</p>
              <p><b>Дифференцированный:</b> платежи уменьшаются. Основной долг делится поровну, проценты начисляются на остаток.</p>
              <p>ПСК (ФЗ-353) включает все расходы по кредиту. Важно сравнивать именно ПСК, а не номинальную ставку.</p>
            </AiExplanation>
          </motion.div>
        )}
      </AnimatePresence>

      <Disclaimer />
    </div>
  )
}