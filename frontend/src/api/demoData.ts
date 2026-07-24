/**
 * demoData.ts — Presentation/demo fallback.
 *
 * Зачем: на Vercel Go-λ может быть недоступен (холодный старт, таймаут БД —
 * см. PROGRESS.md «Блокер λ»). Чтобы приложение оставалось «живым» для презентации,
 * при ошибке/недоступности API мы показываем реалистичные демо-данные — в ТЕХ ЖЕ
 * типах, что ожидает реальный бэкенд. Ни одного parseFloat для денег: суммы — строки.
 *
 * Источник сырых данных — детерминированный генератор mockData.ts (seed=42).
 * Важно: это НЕ подменяет реальные данные — fallback срабатывает ТОЛЬКО когда
 * запрос упал (isError). При живом API пользователь видит свои настоящие данные.
 */
import Decimal from 'decimal.js'
import { getMockOperations } from '../mockData'
import type {
  Account, Operation, Category, Budget, DashboardData, Goal,
  ListResponse, GoalProjection,
} from '../types'

// ── Демо-счета (соответствуют account-типам из domain: cash|bank|savings|...) ──
const DEMO_ACCOUNTS: Account[] = [
  { id: 1, name: 'Наличные', account_type: 'cash', currency: 'RUB', balance: '45000.00', created_at: '2025-01-01T00:00:00Z' },
  { id: 2, name: 'Тинькофф Black', account_type: 'bank', currency: 'RUB', balance: '234500.00', created_at: '2025-01-01T00:00:00Z' },
  { id: 3, name: 'Сбер Вклад', account_type: 'savings', currency: 'RUB', balance: '500000.00', created_at: '2025-01-01T00:00:00Z' },
  { id: 4, name: 'Брокерский счёт', account_type: 'investment', currency: 'RUB', balance: '150000.00', created_at: '2025-01-01T00:00:00Z' },
]

// id категории → имя (совпадает с мок-генератором и categoryColors в store/pages)
const CATEGORY_BY_NAME: Record<string, { id: number; type: 'income' | 'expense' }> = {
  Зарплата: { id: 1, type: 'income' }, Фриланс: { id: 2, type: 'income' },
  Продукты: { id: 3, type: 'expense' }, Транспорт: { id: 4, type: 'expense' },
  Рестораны: { id: 5, type: 'expense' }, Жильё: { id: 6, type: 'expense' },
  Развлечения: { id: 7, type: 'expense' }, Здоровье: { id: 8, type: 'expense' },
  Связь: { id: 9, type: 'expense' }, Одежда: { id: 10, type: 'expense' },
  Образование: { id: 11, type: 'expense' }, Подарки: { id: 12, type: 'expense' },
  Спорт: { id: 13, type: 'expense' }, Подписки: { id: 14, type: 'expense' },
  Переводы: { id: 15, type: 'expense' }, Прочее: { id: 16, type: 'expense' },
}

const ACCOUNT_ID_BY_NAME: Record<string, number> = {
  'Наличные': 1, 'Тинькофф Black': 2, 'Сбер Вклад': 3,
}

/** Преобразовать мок-операции (float amount) → API-тип Operation (строки!). */
function buildDemoOperations(): Operation[] {
  const raw = getMockOperations()
  return raw.map((op, idx) => {
    // float → строка с 2 знаками через decimal.js (никаких parseFloat-потерь в UI-логике)
    const amountStr = new Decimal(op.amount).toFixed(2)
    const cat = CATEGORY_BY_NAME[op.category] ?? CATEGORY_BY_NAME['Прочее']
    const accountId = ACCOUNT_ID_BY_NAME[op.account] ?? 2
    return {
      id: idx + 1,
      calc_id: `demo-${op.id}`,
      type: op.type,
      amount: amountStr,
      currency: 'RUB',
      account_id: accountId,
      category_id: cat.id,
      category_confidence: 0.92,
      counterparty: undefined,
      description: op.description,
      operation_date: op.date,
      is_planned: false,
      created_at: `${op.date}T10:00:00Z`,
    }
  })
}

// Ленивый singleton — генерируется один раз за сессию
let _ops: Operation[] | null = null
function demoOperations(): Operation[] {
  if (!_ops) _ops = buildDemoOperations()
  return _ops
}

const DEMO_CATEGORIES: Category[] = Object.entries(CATEGORY_BY_NAME).map(([name, v]) => ({
  id: v.id, name, parent_id: null, is_system: v.id <= 2, type: v.type,
}))

const DEMO_GOALS: Goal[] = [
  { id: 1, name: 'Подушка безопасности', target_amount: '500000.00', current_amount: '250000.00', monthly_contribution: '20000.00', target_date: '2026-12-31', expected_yield: '0.08', created_at: '2025-01-01T00:00:00Z' },
  { id: 2, name: 'Новый ноутбук', target_amount: '150000.00', current_amount: '80000.00', monthly_contribution: '15000.00', target_date: '2026-09-30', expected_yield: '0.07', created_at: '2025-01-01T00:00:00Z' },
  { id: 3, name: 'Отпуск в Таиланде', target_amount: '200000.00', current_amount: '60000.00', monthly_contribution: '18000.00', target_date: '2026-11-30', expected_yield: '0.07', created_at: '2025-01-01T00:00:00Z' },
  { id: 4, name: 'Инвестиционный портфель', target_amount: '1000000.00', current_amount: '150000.00', monthly_contribution: '25000.00', target_date: '2027-06-30', expected_yield: '0.12', created_at: '2025-01-01T00:00:00Z' },
]

const DEMO_BUDGETS: Budget[] = [
  { id: 1, user_id: 10, category_id: 3, limit_amount: '35000.00', rollover_policy: 'none', is_active: true, created_at: '2025-01-01T00:00:00Z', category_name: 'Продукты' },
  { id: 2, user_id: 10, category_id: 4, limit_amount: '7000.00', rollover_policy: 'none', is_active: true, created_at: '2025-01-01T00:00:00Z', category_name: 'Транспорт' },
  { id: 3, user_id: 10, category_id: 5, limit_amount: '12000.00', rollover_policy: 'none', is_active: true, created_at: '2025-01-01T00:00:00Z', category_name: 'Рестораны' },
  { id: 4, user_id: 10, category_id: 7, limit_amount: '10000.00', rollover_policy: 'none', is_active: true, created_at: '2025-01-01T00:00:00Z', category_name: 'Развлечения' },
  { id: 5, user_id: 10, category_id: 6, limit_amount: '45000.00', rollover_policy: 'none', is_active: true, created_at: '2025-01-01T00:00:00Z', category_name: 'Жильё' },
]

/** Построить сводку дашборда за текущий месяц из демо-операций (строки!). */
function buildDemoDashboard(period: string): DashboardData {
  const now = new Date()
  const monthOps = demoOperations().filter(op => {
    const d = new Date(op.operation_date)
    return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear()
  })
  // Если в текущем месяце пусто (начало месяца) — берём последний непустой месяц
  const source = monthOps.length > 0 ? monthOps : demoOperations().filter(op => {
    const d = new Date(op.operation_date)
    const lm = new Date(now.getFullYear(), now.getMonth() - 1, 1)
    return d.getMonth() === lm.getMonth() && d.getFullYear() === lm.getFullYear()
  })

  let income = new Decimal(0)
  let expense = new Decimal(0)
  const byCat = new Map<number, { name: string; total: Decimal }>()
  for (const op of source) {
    const amt = new Decimal(op.amount)
    if (op.type === 'income') {
      income = income.plus(amt)
    } else {
      expense = expense.plus(amt)
      const cat = DEMO_CATEGORIES.find(c => c.id === op.category_id)
      const name = cat?.name ?? 'Прочее'
      const prev = byCat.get(op.category_id ?? 0)
      byCat.set(op.category_id ?? 0, prev ? { name, total: prev.total.plus(amt) } : { name, total: amt })
    }
  }

  const assets = DEMO_ACCOUNTS
    .filter(a => a.account_type !== 'debt')
    .reduce((s, a) => s.plus(new Decimal(a.balance)), new Decimal(0))

  return {
    period,
    from: source.at(-1)?.operation_date ?? now.toISOString(),
    to: source[0]?.operation_date ?? now.toISOString(),
    income: income.toFixed(2),
    expense: expense.toFixed(2),
    net: income.minus(expense).toFixed(2),
    by_category: Array.from(byCat.entries())
      .map(([category_id, v]) => ({ category_id, category_name: v.name, total: v.total.toFixed(2) }))
      .sort((a, b) => new Decimal(b.total).cmp(new Decimal(a.total))),
    net_worth: {
      assets: assets.toFixed(2),
      debts: '0.00',
      net: assets.toFixed(2),
    },
    goals: DEMO_GOALS.map(g => {
      const cur = new Decimal(g.current_amount)
      const tgt = new Decimal(g.target_amount)
      return { id: g.id, name: g.name, target: g.target_amount, current: g.current_amount, progress: tgt.gt(0) ? cur.div(tgt).toFixed(4) : '0' }
    }),
  }
}

// ── Публичный API fallback'а ──
// Каждая функция возвращает данные в ТОЧНО тех же типах, что реальный бэкенд.

export function demoAccounts(): Account[] {
  return DEMO_ACCOUNTS
}

export function demoCategories(): Category[] {
  return DEMO_CATEGORIES
}

export function demoGoals(): Goal[] {
  return DEMO_GOALS
}

export function demoBudgets(): Budget[] {
  return DEMO_BUDGETS
}

export function demoDashboard(period: string): DashboardData {
  return buildDemoDashboard(period)
}

export function demoOperationsList(limit: number, before?: number): ListResponse<Operation> {
  let ops = demoOperations()
  if (before) ops = ops.filter(op => op.id < before)
  // Свежие сверху — как ожидает UI
  const sorted = [...ops].sort((a, b) => b.operation_date.localeCompare(a.operation_date) || b.id - a.id)
  return { items: sorted.slice(0, limit), more: sorted.length > limit }
}

/**
 * Демо-проекция цели — детерминированная математика sinking fund
 * (Копнова Гл. 3.3.3), без обращения к бэку. Используется при падении API.
 *
 *   FV = PV·(1+i)^n + PMT·((1+i)^n − 1)/i
 *   Решаем для n (месяцев до цели) и PMT (требуемый взнос).
 * i = годовая доходность / 12 (месячная ставка).
 */
export function demoProjection(goalId: number): GoalProjection {
  const goal = DEMO_GOALS.find(g => g.id === goalId) ?? DEMO_GOALS[0]
  const current = new Decimal(goal.current_amount)
  const target = new Decimal(goal.target_amount)
  const pmt = goal.monthly_contribution ? new Decimal(goal.monthly_contribution) : new Decimal(0)
  const annualYield = new Decimal(goal.expected_yield ?? '0')
  const monthlyRate = annualYield.div(12) // i

  const remaining = target.minus(current)
  let monthsLeft = 0
  let requiredMonthly = new Decimal(0)
  let status = 'on_track'

  if (remaining.lte(0)) {
    status = 'achieved'
  } else if (monthlyRate.gt(0)) {
    const onePlusI = monthlyRate.plus(1)
    // Требуемый месячный взнос для достижения за целевого срока:
    // PMT = remaining · i / ((1+i)^n − 1). Срок — из target_date.
    const tgtDate = goal.target_date ? new Date(goal.target_date) : null
    if (tgtDate) {
      const now = new Date()
      monthsLeft = Math.max(0, Math.round((tgtDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24 * 30.44)))
      if (monthsLeft > 0) {
        const powN = onePlusI.pow(monthsLeft) // (1+i)^n
        requiredMonthly = remaining.mul(monthlyRate).div(powN.minus(1))
      }
    }
    // Достаточен ли текущий взнос? Считаем прогнозируемые месяцы по PMT.
    if (pmt.gt(0)) {
      // remaining = PMT·((1+i)^n − 1)/i  →  (1+i)^n = remaining·i/PMT + 1
      const base = remaining.mul(monthlyRate).div(pmt).plus(1)
      if (base.gt(1)) {
        const ln = base.ln()
        const ln1plI = onePlusI.ln()
        const estMonths = ln.div(ln1plI).toNumber()
        if (isFinite(estMonths) && estMonths > 0) {
          status = pmt.gte(requiredMonthly) ? 'on_track' : 'behind'
        }
      }
    }
  } else {
    // Нулевая доходность — простая арифметика
    monthsLeft = goal.target_date
      ? Math.max(0, Math.round((new Date(goal.target_date).getTime() - Date.now()) / (1000 * 60 * 60 * 24 * 30.44)))
      : 0
    requiredMonthly = monthsLeft > 0 ? remaining.div(monthsLeft) : remaining
    if (pmt.gt(0)) status = pmt.gte(requiredMonthly) ? 'on_track' : 'behind'
  }

  const progress = target.gt(0) ? current.div(target).toFixed(4) : '0'
  const targetEff = target // без инфляции в демо

  return {
    goal,
    effective_current: current.toFixed(2),
    target_effective: targetEff.toFixed(2),
    progress,
    months_left: Math.max(0, Math.round(monthsLeft)),
    required_monthly: requiredMonthly.gt(0) ? requiredMonthly.toFixed(2) : undefined,
    estimated_months: monthsLeft > 0 ? Math.round(monthsLeft) : undefined,
    status,
    as_of: new Date().toISOString(),
  }
}
