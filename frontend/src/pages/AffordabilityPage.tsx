import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Scale, ThumbsUp, ThumbsDown, AlertTriangle, Loader2, AlertCircle } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { FormulaTooltip } from '../components/shared/FormulaTooltip'
import { AiExplanation } from '../components/shared/AiExplanation'
import { Disclaimer } from '../components/shared/Disclaimer'
import { useAffordabilityCalc } from '../api/queries'
import { formatMoney, formatNumber, toDecimal } from '../lib/money'
import type { AffordabilityResponse } from '../api/calculators'

// Карта рисков → цвета/подписи
const riskStyle: Record<string, { badge: string; text: string; label: string }> = {
  low: { badge: 'text-emerald-700 bg-emerald-100', text: '#10b981', label: 'Низкий риск' },
  medium: { badge: 'text-amber-700 bg-amber-100', text: '#f59e0b', label: 'Средний риск' },
  high: { badge: 'text-red-700 bg-red-100', text: '#ef4444', label: 'Высокий риск' },
}
const riskOf = (r: string) => riskStyle[r] ?? riskStyle.medium

const stressColor = (r: string) => (r === 'low' ? '#10b981' : r === 'high' ? '#ef4444' : '#f59e0b')

export default function AffordabilityPage() {
  // Форма — входные строки. Деньги/ставки — строки, без parseFloat/Number.
  const [income, setIncome] = useState('150000')
  const [expenses, setExpenses] = useState('50000')
  const [obligations, setObligations] = useState('15000')
  const [cushion, setCushion] = useState('')
  const [loanAmount, setLoanAmount] = useState('3000000')
  const [loanRate, setLoanRate] = useState('21')
  const [loanTerm, setLoanTerm] = useState('60')

  const calc = useAffordabilityCalc()
  const result: AffordabilityResponse | undefined = calc.data
  const isLoading = calc.isPending
  const error = calc.error

  const handleCalculate = () => {
    const termNum = Number.parseInt(loanTerm, 10) || 1
    calc.mutate({
      monthly_incomes: [income],
      mandatory_expenses: expenses || undefined,
      existing_loan_payments: obligations || undefined,
      cushion: cushion || undefined,
      new_loan: {
        principal: loanAmount,
        annual_rate: loanRate,
        term_months: termNum,
      },
    })
  }

  return (
    <div className="space-y-4">
      <motion.div layout>
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 bg-gradient-to-r from-violet-500 to-purple-500 opacity-10" />
          <div className="relative">
            <div className="flex items-center gap-2 mb-1">
              <Scale size={18} style={{ color: 'var(--color-violet-500)' }} />
              <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>Кредитная нагрузка</p>
            </div>
            <div className="flex items-baseline gap-3">
              <p className="text-3xl font-bold font-mono-money gradient-text">
                {result ? `${formatNumber(result.safe_payment_range.low)} – ${formatNumber(result.safe_payment_range.high)}` : '—'}
              </p>
              {result && (
                <span className={`text-sm font-medium px-2 py-0.5 rounded-full ${riskOf(result.risk).badge}`}>
                  {riskOf(result.risk).label}
                </span>
              )}
            </div>
            <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
              {result ? `Безопасный диапазон платежа, риск: ${result.scenario}` : 'Диапазон безопасного платежа'}
            </p>
          </div>
        </Card>
      </motion.div>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Доходы и расходы</h3>
        <Input label="Ежемесячный доход (₽)" type="number" value={income} onChange={e => setIncome(e.target.value)} />
        <Input label="Обязательные расходы (₽)" type="number" value={expenses} onChange={e => setExpenses(e.target.value)} />
        <Input label="Текущие обязательства (₽)" type="number" value={obligations} onChange={e => setObligations(e.target.value)} hint="Кредиты, алименты и т.д." />
        <Input label="Финансовая подушка (₽, опционально)" type="number" value={cushion} onChange={e => setCushion(e.target.value)} hint="Сбережения на счёте" />
      </Card>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Параметры кредита</h3>
        <Input label="Сумма кредита (₽)" type="number" value={loanAmount} onChange={e => setLoanAmount(e.target.value)} />
        <Input label="Ставка (%)" type="number" step="0.1" value={loanRate} onChange={e => setLoanRate(e.target.value)} />
        <Input label="Срок (мес.)" type="number" value={loanTerm} onChange={e => setLoanTerm(e.target.value)} />
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
            <div className="flex items-start gap-2 p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700">
              <AlertCircle size={14} className="mt-0.5 flex-shrink-0" />
              <p>Не удалось выполнить оценку. {(error as Error)?.message || 'Проверьте введённые данные и попробуйте снова.'}</p>
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
            {result.refused_reason && (
              <div className="flex items-start gap-2 p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700">
                <AlertCircle size={14} className="mt-0.5 flex-shrink-0" />
                <p><b>Кредит не рекомендован:</b> {result.refused_reason}</p>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Стабильный доход</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{formatMoney(result.stable_income)}</p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Свободный остаток</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{formatMoney(result.free_remainder)}</p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Платёж по новому кредиту</p>
                  <p className="text-base font-semibold font-mono-money text-red-500">{formatMoney(result.new_payment)}</p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Подушка (мес.)</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{result.cushion_months}</p>
                </Card>
              </motion.div>
            </div>

            <Card>
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Безопасный диапазон платежа</p>
              <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                {formatMoney(result.safe_payment_range.low)} – {formatMoney(result.safe_payment_range.high)}
              </p>
            </Card>

            {/* Стресс-сценарии из ответа */}
            {result.stress.length > 0 && (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Стресс-сценарии</h3>
                <AnimatePresence>
                  {result.stress.map((s, i) => {
                    const risk = riskOf(s.risk)
                    const pass = toDecimal(s.free_remainder).gte(0)
                    return (
                      <motion.div
                        key={s.scenario + i}
                        layout
                        initial={{ opacity: 0, x: -12 }}
                        animate={{ opacity: 1, x: 0 }}
                        exit={{ opacity: 0, x: 12 }}
                        transition={{ duration: 0.2, delay: i * 0.04 }}
                        className="relative overflow-hidden card p-4"
                        style={{ borderColor: stressColor(s.risk) }}
                      >
                        <div className="absolute inset-0" style={{ background: stressColor(s.risk), opacity: 0.05 }} />
                        <div className="relative flex items-start gap-3">
                          {pass
                            ? <ThumbsUp size={20} className="mt-0.5" style={{ color: '#10b981' }} />
                            : <ThumbsDown size={20} className="mt-0.5" style={{ color: stressColor(s.risk) }} />}
                          <div className="flex-1">
                            <div className="flex items-center justify-between gap-2">
                              <p className="text-sm font-semibold" style={{ color: pass ? '#059669' : stressColor(s.risk) }}>
                                {s.scenario}
                              </p>
                              <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${risk.badge}`}>
                                {risk.label}
                              </span>
                            </div>
                            <div className="grid grid-cols-2 gap-x-3 gap-y-1 mt-2 text-xs">
                              <div>
                                <span style={{ color: 'var(--text-tertiary)' }}>Падение дохода: </span>
                                <span className="font-mono-money" style={{ color: 'var(--text-primary)' }}>{formatNumber(s.income_drop_pct)}%</span>
                              </div>
                              <div>
                                <span style={{ color: 'var(--text-tertiary)' }}>Эфф. доход: </span>
                                <span className="font-mono-money" style={{ color: 'var(--text-primary)' }}>{formatMoney(s.effective_income)}</span>
                              </div>
                              <div>
                                <span style={{ color: 'var(--text-tertiary)' }}>Платёж: </span>
                                <span className="font-mono-money text-red-500">{formatMoney(s.new_payment)}</span>
                              </div>
                              <div>
                                <span style={{ color: 'var(--text-tertiary)' }}>Доля платежа: </span>
                                <span className="font-mono-money" style={{ color: stressColor(s.risk) }}>{formatNumber(s.payment_share_pct)}%</span>
                              </div>
                              <div className="col-span-2">
                                <span style={{ color: 'var(--text-tertiary)' }}>Свободный остаток: </span>
                                <span className="font-mono-money" style={{ color: pass ? '#10b981' : '#ef4444' }}>{formatMoney(s.free_remainder)}</span>
                              </div>
                            </div>
                            {pass ? (
                              <p className="text-xs mt-2" style={{ color: '#059669' }}>
                                <ThumbsUp size={12} className="inline mr-1" /> Сценарий пройден
                              </p>
                            ) : (
                              <p className="text-xs mt-2" style={{ color: '#dc2626' }}>
                                <AlertTriangle size={12} className="inline mr-1" /> Сценарий не пройден
                              </p>
                            )}
                          </div>
                        </div>
                      </motion.div>
                    )
                  })}
                </AnimatePresence>
              </div>
            )}

            <FormulaTooltip
              title="DTI (Debt-to-Income)"
              formula="DTI = \frac{P + O}{I} \cdot 100\%"
              explanation="P — платеж по кредиту, O — текущие обязательства, I — доход. Норма — до 30-40%."
            />
            <AiExplanation>
              <p>Оценка доступности кредита анализирует вашу способность обслуживать долг.</p>
              <p><b>DTI &lt; 30%:</b> комфортная нагрузка. Банки одобряют такой кредит.</p>
              <p><b>DTI 30-50%:</b> средняя нагрузка. Возможны отказы или повышенная ставка.</p>
              <p><b>DTI &gt; 50%:</b> критическая нагрузка. Рекомендуется снизить сумму или увеличить срок.</p>
              <p>Стресс-тест моделирует потерю дохода — базовая проверка финансовой устойчивости.</p>
            </AiExplanation>
          </motion.div>
        )}
      </AnimatePresence>

      <Disclaimer />
    </div>
  )
}