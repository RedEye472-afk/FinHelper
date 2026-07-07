import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { PiggyBank } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { FormulaTooltip } from '../components/shared/FormulaTooltip'
import { AiExplanation } from '../components/shared/AiExplanation'
import { Disclaimer } from '../components/shared/Disclaimer'
import { toDecimal, M, formatMoney, formatNumber, Decimal } from '../lib/money'

interface DepositResults {
  total: Decimal
  interest: Decimal
  effectiveRate: Decimal
  tax: Decimal
  afterTax: Decimal
  monthlyIncome: Decimal
  principal: Decimal
}

export default function DepositPage() {
  const [amount, setAmount] = useState('500000')
  const [rate, setRate] = useState('18')
  const [term, setTerm] = useState('12')
  const [capType, setCapType] = useState<'none' | 'monthly' | 'quarterly'>('monthly')
  const [submitted, setSubmitted] = useState(false)

  const results = useMemo<DepositResults | null>(() => {
    // Все вводы — строки; конвертация через toDecimal/safeParse, НЕ parseFloat.
    const P = toDecimal(amount)
    // r — годовая ставка в долях: 18% → 0.18
    const r = toDecimal(rate).div(100)
    // n — срок в месяцах (count). parseInt разрешён для количества месяцев.
    const n = new Decimal(Number.parseInt(term, 10) || 0)
    if (!M.isPositive(P) || !M.isPositive(r) || !M.isPositive(n)) return null

    // periods — число периодов капитализации в год
    const periods = capType === 'monthly' ? 12 : capType === 'quarterly' ? 4 : 1
    const periodsDec = new Decimal(periods)

    // t — срок в годах = n/12
    const t = n.div(12)

    // Сложный процент: S = P * (1 + r/n)^(n*t)
    // Для capType === 'none' — простые проценты: S = P * (1 + r * t)
    let total: Decimal
    if (capType === 'none') {
      total = M.mul(P, M.add(new Decimal(1), M.mul(r, t)))
    } else {
      const ratePerPeriod = M.div(r, periodsDec)
      const totalPeriods = M.mul(periodsDec, t)
      const base = M.add(new Decimal(1), ratePerPeriod)
      total = M.mul(P, base.pow(totalPeriods))
    }

    const interest = M.sub(total, P)

    // Эффективная годовая ставка: ((S/P)^(12/n) - 1) * 100
    const ratio = M.div(total, P)
    const effectiveRate = ratio.pow(new Decimal(12).div(n)).minus(1).mul(100)

    // Налог НДФЛ 13% с превышения над 190000 * (n/12)
    const taxRate = new Decimal(0.13)
    const taxFreeThreshold = new Decimal(190000)
    const taxFree = M.mul(taxFreeThreshold, M.div(n, new Decimal(12)))
    const taxableInterest = Decimal.max(new Decimal(0), M.sub(interest, taxFree))
    const tax = M.mul(taxableInterest, taxRate)

    const afterTax = M.sub(total, tax)
    const monthlyIncome = M.div(interest, n)

    return { total, interest, effectiveRate, tax, afterTax, monthlyIncome, principal: P }
  }, [amount, rate, term, capType])

  const handleCalculate = () => {
    setSubmitted(true)
  }

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
            key={results ? results.total.toString() : 'empty'}
          >
            {results ? formatMoney(results.total) : '—'}
          </motion.p>
          <p className="text-xs text-white/70 mt-1">
            Чистый доход: {results ? formatMoney(results.interest) : '—'}
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
            {(['monthly', 'quarterly', 'none'] as const).map(c => (
              <button key={c} onClick={() => setCapType(c)}
                className={`flex-1 py-2.5 rounded-xl text-xs font-medium transition-all ${capType === c ? 'bg-gradient-primary text-white shadow-sm' : ''}`}
                style={capType !== c ? { background: 'var(--bg-surface)', color: 'var(--text-secondary)', border: '1px solid var(--border-default)' } : {}}>
                {c === 'monthly' ? 'Ежемес.' : c === 'quarterly' ? 'Ежекв.' : 'Без'}
              </button>
            ))}
          </div>
        </div>
        <Button variant="primary" className="w-full" onClick={handleCalculate}>
          Рассчитать
        </Button>
      </Card>

      <AnimatePresence mode="wait">
        {submitted && results && (
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
                    {formatMoney(results.interest)}
                  </p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Эфф. ставка</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                    {formatNumber(results.effectiveRate)}%
                  </p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Налог (НДФЛ 13%)</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--color-danger-500)' }}>
                    {formatMoney(results.tax)}
                  </p>
                </Card>
              </motion.div>
              <motion.div layout>
                <Card>
                  <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Доход в месяц</p>
                  <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>
                    {formatMoney(results.monthlyIncome)}
                  </p>
                </Card>
              </motion.div>
            </div>

            <FormulaTooltip
              title="Формула сложного процента"
              formula="S = P \cdot (1 + \frac{r}{n})^{n \cdot t}"
              explanation="S — итоговая сумма, P — начальный вклад, r — годовая ставка, n — число периодов капитализации в год, t — срок в годах"
            />
            <AiExplanation>
              <p>Депозитный калькулятор рассчитывает доходность вклада с учётом капитализации процентов и налогов.</p>
              <p>Капитализация — это начисление процентов на уже накопленные проценты, что даёт эффект «снежного кома». Чем чаще капитализация, тем выше эффективная ставка.</p>
              <p>Налог на доход по вкладам (НДФЛ 13%) платится с суммы превышения необлагаемого лимита (190 000 ₽ × (n/12), где n — срок в месяцах).</p>
            </AiExplanation>
          </motion.div>
        )}
      </AnimatePresence>
      <Disclaimer />
    </div>
  )
}