/**
 * store.tsx — FinanceProvider питается от бэкенда через React Query.
 * Деньги обрабатываем через decimal.js (см. lib/money.ts).
 * Нет фоллбек-данных — новые пользователи видят нули.
 */
import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { useDashboard } from './api/queries'
import { toDecimal, M } from './lib/money'
import type Decimal from 'decimal.js'

interface KPI {
  totalBalance: Decimal
  monthIncome: Decimal
  monthExpense: Decimal
  savingsRate: number
}

interface CategorySpending {
  category: string
  amount: Decimal
  color: string
}

interface FinanceContextValue {
  kpi: KPI
  expensesByCategory: CategorySpending[]
  isLoading: boolean
  isMockData: boolean
}

const FinanceContext = createContext<FinanceContextValue | null>(null)

const categoryColors: Record<string, string> = {
  'Зарплата': '#10b981', 'Фриланс': '#34d399', 'Продукты': '#f59e0b',
  'Транспорт': '#3b82f6', 'Рестораны': '#f97316', 'Жильё': '#ef4444',
  'Развлечения': '#8b5cf6', 'Здоровье': '#ec4899', 'Связь': '#06b6d4',
  'Одежда': '#6366f1', 'Образование': '#14b8a6', 'Подарки': '#f43f5e',
  'Спорт': '#a855f7', 'Подписки': '#0ea5e9', 'Переводы': '#64748b', 'Прочее': '#6b7280',
}

export function FinanceProvider({ children }: { children: ReactNode }) {
  const { data: dashData, isLoading } = useDashboard()

  const kpi = useMemo<KPI>(() => {
    if (!dashData) return { totalBalance: M.zero(), monthIncome: M.zero(), monthExpense: M.zero(), savingsRate: 0 }
    const income = toDecimal(dashData.income)
    const expenses = toDecimal(dashData.expense)
    const netWorth = toDecimal(dashData.net_worth.net)
    const savingsRate = income.gt(0) ? income.minus(expenses).div(income).mul(100).toNumber() : 0
    return { totalBalance: netWorth, monthIncome: income, monthExpense: expenses, savingsRate: Math.round(savingsRate) }
  }, [dashData])

  const expensesByCategory = useMemo<CategorySpending[]>(() => {
    if (!dashData?.by_category || dashData.by_category.length === 0) return []
    return dashData.by_category.map(ec => ({
      category: ec.category_name,
      amount: toDecimal(ec.total),
      color: categoryColors[ec.category_name] || '#6b7280',
    }))
  }, [dashData])

  const isMockData = !dashData

  return (
    <FinanceContext.Provider value={{ kpi, expensesByCategory, isLoading, isMockData }}>
      {children}
    </FinanceContext.Provider>
  )
}

export function useFinance() {
  const ctx = useContext(FinanceContext)
  if (!ctx) throw new Error('useFinance must be used within FinanceProvider')
  return ctx
}
