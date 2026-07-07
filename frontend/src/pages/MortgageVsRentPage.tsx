import { useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import { Home, TrendingUp, Building2, Award } from 'lucide-react'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { FormulaTooltip } from '../components/shared/FormulaTooltip'
import { AiExplanation } from '../components/shared/AiExplanation'
import { Disclaimer } from '../components/shared/Disclaimer'
import { Decimal, safeParse, formatNumber } from '../lib/money'

interface MortgateVsRentResult {
  monthlyMortgage: Decimal
  totalMortgagePayment: Decimal
  overpayment: Decimal
  futurePropertyValue: Decimal
  netMortgage: Decimal
  netRent: Decimal
  winner: 'mortgage' | 'rent'
  advantage: Decimal
  downPaymentAmt: Decimal
  loanAmt: Decimal
  totalRentPaid: Decimal
  hiddenCosts: Decimal
  propertyTax: Decimal
  insurance: Decimal
  utilities: Decimal
  monthlyDiff: Decimal
  price: Decimal
  rent: Decimal
}

const TAX_RATE = '0.003'      // 0.3% / год
const INSURANCE_RATE = '0.005' // 0.5% / год
const UTILITIES_MONTHLY = '5000'

export default function MortgageVsRentPage() {
  const [price, setPrice] = useState('8000000')
  const [downPayment, setDownPayment] = useState('20')
  const [mortgageRate, setMortgageRate] = useState('21')
  const [mortgageTerm, setMortgageTerm] = useState('240')
  const [monthlyRent, setMonthlyRent] = useState('40000')
  const [rentGrowth, setRentGrowth] = useState('5')
  const [propertyGrowth, setPropertyGrowth] = useState('6')
  const [investmentReturn, setInvestmentReturn] = useState('12')

  const result = useMemo<MortgateVsRentResult | null>(() => {
    const P = safeParse(price)
    if (!P.isPositive()) return null

    // term — это количество месяцев (count), parseInt разрешён.
    const n = Number.parseInt(mortgageTerm, 10) || 1
    const years = new Decimal(n).div(12)

    // Проценты → Decimal (в долях единицы)
    const dpPct = safeParse(downPayment).div(100)
    const annualRate = safeParse(mortgageRate).div(100)
    const monthlyRate = annualRate.div(12)
    const rentMonthly = safeParse(monthlyRent)
    const rg = safeParse(rentGrowth).div(100)
    const pg = safeParse(propertyGrowth).div(100)
    const ir = safeParse(investmentReturn).div(100)

    const downPaymentAmt = P.mul(dpPct)
    const loanAmt = P.sub(downPaymentAmt)

    // Аннуитетный коэффициент k = r*(1+r)^n / ((1+r)^n - 1)
    // monthlyMortgage = loanAmt * k = S * r * (1+r)^n / ((1+r)^n - 1)
    let monthlyMortgage: Decimal
    if (monthlyRate.isZero()) {
      monthlyMortgage = loanAmt.div(n)
    } else {
      const one = new Decimal(1)
      const growth = one.plus(monthlyRate).pow(n)
      const k = monthlyRate.mul(growth).div(growth.sub(one))
      monthlyMortgage = loanAmt.mul(k)
    }

    const totalMortgagePayment = monthlyMortgage.mul(n)
    const overpayment = totalMortgagePayment.sub(loanAmt)

    // Будущая стоимость квартиры: P * (1 + pg)^years
    const futurePropertyValue = P.mul(new Decimal(1).plus(pg).pow(years))

    // Будущая стоимость первого взноса, если инвестировать: downPayment * (1 + ir)^years
    const investedDownPayment = downPaymentAmt.mul(new Decimal(1).plus(ir).pow(years))

    const monthlyDiff = monthlyMortgage.sub(rentMonthly)

    // Аренда за весь срок (растёт ежегодно)
    let totalRentPaid = new Decimal(0)
    let annualRent = rentMonthly.mul(12)
    const yearsInt = Math.round(years.toNumber())
    for (let y = 0; y < yearsInt; y++) {
      totalRentPaid = totalRentPaid.plus(annualRent)
      annualRent = annualRent.mul(new Decimal(1).plus(rg))
    }

    // Скрытые расходы homeowners
    const propertyTax = P.mul(TAX_RATE).mul(years)
    const insurance = P.mul(INSURANCE_RATE).mul(years)
    const utilities = new Decimal(UTILITIES_MONTHLY).mul(n)
    const hiddenCosts = propertyTax.plus(insurance).plus(utilities)

    const mortgageTotalWithCosts = totalMortgagePayment.plus(hiddenCosts)

    // Честное сравнение: ежемесячная разница инвестируется под ir/12
    const monthlyInvRate = ir.div(12)
    let fvFactor: Decimal
    if (monthlyInvRate.isZero()) {
      fvFactor = new Decimal(n)
    } else {
      fvFactor = new Decimal(1).plus(monthlyInvRate).pow(n).sub(1).div(monthlyInvRate)
    }
    const renterExtra = monthlyDiff.isPositive() ? monthlyDiff.mul(fvFactor) : new Decimal(0)
    const homeownerExtra = monthlyDiff.isNegative() ? monthlyDiff.negated().mul(fvFactor) : new Decimal(0)

    const netMortgage = futurePropertyValue.plus(homeownerExtra).sub(mortgageTotalWithCosts)
    const netRent = investedDownPayment.plus(renterExtra).sub(totalRentPaid)

    const winner = netMortgage.gt(netRent) ? 'mortgage' : 'rent'
    const advantage = netMortgage.sub(netRent).abs()

    return {
      monthlyMortgage, totalMortgagePayment, overpayment, futurePropertyValue,
      netMortgage, netRent, winner, advantage, downPaymentAmt, loanAmt,
      totalRentPaid, hiddenCosts, propertyTax, insurance, utilities,
      monthlyDiff, price: P, rent: rentMonthly,
    }
  }, [price, downPayment, mortgageRate, mortgageTerm, monthlyRent, rentGrowth, propertyGrowth, investmentReturn])

  const fmt = (d: Decimal): string => formatNumber(d)

  return (
    <div className="space-y-4">
      {result && (
        <motion.div layout initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.25 }}>
          <Card className="relative overflow-hidden">
            <div className={`absolute inset-0 opacity-10 ${result.winner === 'mortgage' ? 'bg-gradient-to-r from-emerald-500 to-teal-500' : 'bg-gradient-to-r from-blue-500 to-violet-500'}`} />
            <div className="relative">
              <div className="flex items-center gap-2 mb-1">
                {result.winner === 'mortgage' ? <Building2 size={18} className="text-emerald-500" /> : <TrendingUp size={18} className="text-blue-500" />}
                <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                  {result.winner === 'mortgage' ? 'Выгоднее ипотека' : 'Выгоднее аренда'}
                </p>
              </div>
              <p className="text-2xl font-bold font-mono-money gradient-text">+{fmt(result.advantage)} ₽</p>
              <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                Разница в пользу {result.winner === 'mortgage' ? 'ипотеки' : 'аренды'} за {Math.round(Number.parseInt(mortgageTerm, 10) / 12)} лет
              </p>
            </div>
          </Card>
        </motion.div>
      )}

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Недвижимость</h3>
        <Input label="Стоимость квартиры (₽)" type="number" value={price} onChange={e => setPrice(e.target.value)} />
        <Input label="Первоначальный взнос (%)" type="number" value={downPayment} onChange={e => setDownPayment(e.target.value)} />
        <Input label="Рост стоимости жилья (%/год)" type="number" step="0.5" value={propertyGrowth} onChange={e => setPropertyGrowth(e.target.value)} />
      </Card>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Ипотека</h3>
        <Input label="Ставка (%)" type="number" step="0.1" value={mortgageRate} onChange={e => setMortgageRate(e.target.value)} />
        <Input label="Срок (мес.)" type="number" value={mortgageTerm} onChange={e => setMortgageTerm(e.target.value)} />
      </Card>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Аренда и инвестиции</h3>
        <Input label="Аренда в месяц (₽)" type="number" value={monthlyRent} onChange={e => setMonthlyRent(e.target.value)} />
        <Input label="Рост аренды (%/год)" type="number" step="0.5" value={rentGrowth} onChange={e => setRentGrowth(e.target.value)} />
        <Input label="Доходность инвестиций (%/год)" type="number" step="0.5" value={investmentReturn} onChange={e => setInvestmentReturn(e.target.value)} hint="На сколько вырастет первый взнос, если его инвестировать" />
      </Card>

      {result && (
        <motion.div layout className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <Card>
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Платёж по ипотеке</p>
              <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmt(result.monthlyMortgage)} ₽</p>
            </Card>
            <Card>
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Аренда сейчас</p>
              <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmt(result.rent)} ₽</p>
            </Card>
            <Card>
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Переплата по ипотеке</p>
              <p className="text-base font-semibold font-mono-money text-red-500">{fmt(result.overpayment)} ₽</p>
            </Card>
            <Card>
              <p className="text-xs mb-1" style={{ color: 'var(--text-secondary)' }}>Аренда за весь срок</p>
              <p className="text-base font-semibold font-mono-money" style={{ color: 'var(--text-primary)' }}>{fmt(result.totalRentPaid)} ₽</p>
            </Card>
          </div>

          <Card>
            <h4 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>Скрытые расходы ипотеки</h4>
            <div className="space-y-1.5 text-xs" style={{ color: 'var(--text-secondary)' }}>
              <div className="flex justify-between"><span>Налог на недвижимость (0.3%/год)</span><span className="font-mono-money">{fmt(result.propertyTax)} ₽</span></div>
              <div className="flex justify-between"><span>Страхование (0.5%/год)</span><span className="font-mono-money">{fmt(result.insurance)} ₽</span></div>
              <div className="flex justify-between"><span>Коммунальные платежи</span><span className="font-mono-money">{fmt(result.utilities)} ₽</span></div>
              <div className="flex justify-between font-semibold pt-1" style={{ borderTop: '1px solid var(--border-subtle)', color: 'var(--text-primary)' }}>
                <span>Всего скрытых</span><span className="font-mono-money">{fmt(result.hiddenCosts)} ₽</span>
              </div>
            </div>
          </Card>

          <div className="grid grid-cols-2 gap-3">
            <Card>
              <div className="flex items-center gap-1.5 mb-1">
                <Building2 size={14} style={{ color: 'var(--color-emerald-500)' }} />
                <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Итог ипотеки</p>
              </div>
              <p className={`text-lg font-bold font-mono-money ${result.winner === 'mortgage' ? 'text-emerald-600' : ''}`}>
                {fmt(result.netMortgage)} ₽
              </p>
            </Card>
            <Card>
              <div className="flex items-center gap-1.5 mb-1">
                <TrendingUp size={14} style={{ color: 'var(--color-blue-500)' }} />
                <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Итог аренды + инвестиции</p>
              </div>
              <p className={`text-lg font-bold font-mono-money ${result.winner === 'rent' ? 'text-emerald-600' : ''}`}>
                {fmt(result.netRent)} ₽
              </p>
            </Card>
          </div>

          <Card className="bg-gradient-primary text-white">
            <div className="flex items-center gap-2">
              <Award size={20} />
              <p className="text-sm font-semibold">
                {result.winner === 'mortgage'
                  ? 'Ипотека выгоднее — квартира растёт в цене быстрее, чем аренда съедает разницу'
                  : 'Аренда выгоднее — разумнее инвестировать разницу и первый взнос'}
              </p>
            </div>
          </Card>
        </motion.div>
      )}

      <FormulaTooltip
        title="Сравнение ипотеки и аренды"
        formula="Net_{mortgage} = V_{future} - (P_{total} + C_{hidden})"
        explanation="V — стоимость квартиры в будущем, P — все платежи по ипотеке, C — скрытые расходы"
      />
      <AiExplanation>
        <p>Калькулятор сравнивает два сценария на срок ипотеки с учётом всех расходов.</p>
        <p><b>Ипотека:</b> вы платите кредит + коммуналка + налог + страховка. В конце у вас квартира, которая выросла в цене.</p>
        <p><b>Аренда:</b> вы платите аренду (растёт каждый год), а первый взнос и разницу с ипотекой инвестируете под заданный %.</p>
        <p>Ключевые переменные: рост цен на жильё и доходность альтернативных инвестиций. При доходности 12%+ аренда почти всегда выгоднее.</p>
      </AiExplanation>
      <Disclaimer />
    </div>
  )
}