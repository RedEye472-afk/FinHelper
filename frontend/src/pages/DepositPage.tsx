import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { PiggyBank, ChevronDown, ChevronUp, Loader2, AlertCircle } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { FormulaTooltip } from '../components/shared/FormulaTooltip'
import { AiExplanation } from '../components/shared/AiExplanation'
import { Disclaimer } from '../components/shared/Disclaimer'
import { useDepositCalc } from '../api/queries'
import { formatMoney, formatNumber } from '../lib/money'
import type { DepositResponse } from '../api/calculators'

export default function DepositPage() {
  const [amount, setAmount] = useState('500000')
  const [rate, setRate] = useState('18')
  const [term, setTerm] = useState('12')
  const [capType, setCapType] = useState<'monthly' | 'quarterly' | 'annually' | 'maturity'>('monthly')
  const [showProjection, setShowProjection] = useState(false)

  const calc = useDepositCalc()
  const result: DepositResponse | undefined = calc.data
  const isLoading = calc.isPending
  const error = calc.error

  const handleCalculate = () => {
    calc.mutate({
      principal: amount,
      annual_rate: rate,
      term_months: Number.parseInt(term, 10) || 1,
      capitalization: capType,
      tax_year: new Date().getFullYear(),
    })
  }

  const fmtMoney = (s: string) => formatMoney(s)
  const fmtNum = (s: string) => formatNumber(s)

  return (
    <div className="space-y-4">
      <motion.div layout>
        <Card className="bg-gradient-primary text-white">
          <div className="flex items-center gap-2 mb-1">
            <PiggyBank size={18} />
            <p className="text-sm text-white/80">Итог при размещении</p>
          </div>
          <motion.p
            layout
            className="text-2xl font-bold font-mono-money gradient-text"
            key={result ? result.maturity_amount : 'empty'}
          >
            {result ? fmtMoney(result.maturity_amount) : '—'}
          </motion.p>
          <p className="text-xs text-white/70 mt-1">
            Начислено %: {result ? fmtMoney(result.total_interest) : '—'}
          </p>
        </Card>
      </motion.div>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Параметры вклада</h3>
        <Input label="Сумма вклада (₽)" type="number" value={amount} onChange={e => setAmount(e.target.value)} />
        <Input label="Годовая ставка (%)" type="number" step="0.1" value={rate} onChange={e => setRate(e.target.value)} />
        <Input label="Срок (мес.)" type="number" value={term} onChange={e => setTerm(e.target.value)} />
        <div>
          <label className="block text-sm font-medium mb-1" style={{ color: 'var(--text-primary)' }}>Капитализация</label>
          <div className="flex gap-2">
            {(['monthly', 'quarterly', 'annually', 'maturity'] as const).map(c => (
              <button key={c} onClick={() => setCapType(c)}
                className={`flex-1 py-2.5 rounded-xl text-xs font-medium transition-all ${capType === c ? 'bg-gradient-primary text-white shadow-sm' : ''}`}
                style={capType !== c ? { background: 'var(--bg-surface)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' } : {}}>
                {c === 'monthly' ? 'Ежемес.' : c === 'quarterly' ? 'Ежекв.' : c === 'annually' ? 'Ежегод.' : 'Без'}
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
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Начислено %</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                    {fmtMoney(result.total_interest)}
                  </p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Эфф. ставка</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                    {fmtNum(result.effective_rate)}%
                  </p>
                </Card>
              </motion.div>
              {result.tax_amount && (
                <motion.div layout>
                  <Card>
                    <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Налог (НДФЛ 13%)</p>
                    <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--color-danger-500)' }}>
                      {fmtMoney(result.tax_amount)}
                    </p>
                  </Card>
                </motion.div>
              )}
              {result.real_return && (
                <motion.div layout>
                  <Card>
                    <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Реальная доходность</p>
                    <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                      {fmtNum(result.real_return)}%
                    </p>
                  </Card>
                </motion.div>
              )}
            </div>

            {result.projection && result.projection.length > 0 && (
              <>
                <Button variant="secondary" onClick={() => setShowProjection(!showProjection)} className="w-full justify-between">
                  <span>График начислений</span>
                  {showProjection ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </Button>

                <AnimatePresence>
                  {showProjection && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: 'auto' }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={{ duration: 0.2 }}
                      className="overflow-hidden"
                    >
                      <div className="overflow-x-auto rounded-xl max-h-64 overflow-y-auto" style={{ border: '1px solid var(--border-default)' }}>
                        <table className="w-full text-xs">
                          <thead className="sticky top-0" style={{ background: 'var(--bg-surface)' }}>
                            <tr>
                              <th className="px-3 py-2 text-left font-medium" style={{ color: 'var(--text-secondary)' }}>№</th>
                              <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>Остаток</th>
                              <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>% за месяц</th>
                              <th className="px-3 py-2 text-right font-medium" style={{ color: 'var(--text-secondary)' }}>% накоплено</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y" style={{ borderColor: 'var(--border-subtle)' }}>
                            {result.projection.map(p => (
                              <tr key={p.month} className="transition-colors hover:bg-gray-50">
                                <td className="px-3 py-1.5" style={{ color: 'var(--text-tertiary)' }}>{p.month}</td>
                                <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtNum(p.balance)}</td>
                                <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmtNum(p.interest)}</td>
                                <td className="px-3 py-1.5 text-right font-mono-money" style={{ color: 'var(--color-success-500)' }}>{fmtNum(p.cumulative_interest)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </>
            )}

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
              title="Формула сложного процента"
              formula="S = P \cdot (1 + \frac{r}{n})^{n \cdot t}"
              explanation="S — итоговая сумма, P — начальный вклад, r — годовая ставка, n — число периодов капитализации в год, t — срок в годах"
            />
            <AiExplanation>
              <p>Депозитный калькулятор рассчитывает доходность вклада с учётом капитализации процентов и налогов.</p>
              <p>Капитализация — это начисление процентов на уже накопленные проценты, что даёт эффект «снежного кома». Чем чаще капитализация, тем выше эффективная ставка.</p>
              <p>Дополнительно рассчитывается реальная доходность (с поправкой на инфляцию по уравнению Фишера) и налог на доход по вкладам (НДФЛ 13%) в соответствии с ФЗ-382.</p>
            </AiExplanation>
          </motion.div>
        )}
      </AnimatePresence>
      <Disclaimer />
    </div>
  )
}
