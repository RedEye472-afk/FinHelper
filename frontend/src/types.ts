// ===== API Types — matching real Go backend =====

// === User ===
export interface User {
  id: number
  email: string
  created_at: string
}

// === Accounts ===
export interface Account {
  id: number
  name: string
  account_type: 'cash' | 'bank' | 'savings' | 'investment' | 'crypto' | 'debt'
  currency: string
  balance: string
  created_at: string
  updated_at?: string
}

export interface AccountCreate {
  name: string
  type: string
  currency: string
}

// === Operations ===
export interface Operation {
  id: number
  calc_id: string
  type: string
  amount: string
  currency?: string
  account_id: number
  account_dst_id?: number | null
  category_id?: number | null
  income_subtype?: string | null
  category_confidence?: number | null
  counterparty?: string
  description?: string
  operation_date: string
  is_planned: boolean
  created_at: string
  updated_at?: string
}

export interface OperationCreate {
  type: string
  amount: string
  account_id: number
  category_id?: number
  counterparty?: string
  description?: string
  operation_date?: string
  calc_id: string // UUID for idempotency
}

// === Generic list response (backend returns {items: T[], more: boolean}) ===
export interface ListResponse<T> {
  items: T[]
  more: boolean
}

// === Dashboard ===
export interface DashboardData {
  period: string
  from: string
  to: string
  income: string
  expense: string
  net: string
  by_category: ByCategoryEntry[]
  net_worth: NetWorth
  goals: GoalProgress[]
}

export interface ByCategoryEntry {
  category_id: number
  category_name: string
  total: string
}

export interface NetWorth {
  assets: string
  debts: string
  net: string
}

export interface GoalProgress {
  id: number
  name: string
  target: string
  current: string
  progress: string
}

// === Budgets ===
export interface Budget {
  id: number
  user_id: number
  category_id: number
  limit_amount: string
  rollover_policy: 'none' | 'unlimited' | 'months_3'
  is_active: boolean
  created_at: string
  updated_at?: string
  // Computed on frontend
  category_name?: string
}

export interface BudgetCreate {
  category_id: number
  limit_amount: string
  rollover_policy?: string
}

export interface BudgetStatus {
  budget: Budget
  period_from: string
  period_to: string
  spent: string
  rollover: string
  effective_limit: string
}

// === Categories ===
export interface Category {
  id: number
  name: string
  parent_id?: number | null
  is_system: boolean
  // type is NOT returned by backend — frontend may derive it
  type?: 'income' | 'expense'
}

// === Goals ===
export interface Goal {
  id: number
  name: string
  target_amount: string
  current_amount: string
  monthly_contribution: string | null
  target_date: string | null
  expected_yield: string
  created_at: string
  updated_at?: string
}

export interface GoalCreate {
  name: string
  target_amount: string
  target_date?: string
  monthly_contribution?: string
  expected_yield?: string
}

export interface GoalResponse {
  id: number
  name: string
  target_amount: string
  current_amount: string
  monthly_contribution: string | null
  target_date: string | null
  expected_yield: string
  created_at: string
  updated_at?: string
}

export interface ContributionCreate {
  goal_id: number
  amount: string
  contribution_id: string
  contribution_date?: string
  comment?: string
}

export interface GoalProjection {
  goal: Goal
  effective_current: string
  target_effective: string
  progress: string
  months_left: number
  required_monthly?: string
  estimated_months?: number
  status: string
  as_of: string
}

// ===== Local UI Types (unchanged) =====
export interface AuthTokens {
  access_token: string
  refresh_token: string
}

export type OperationType = 'income' | 'expense'
export type ThemeMode = 'emerald' | 'blue' | 'purple' | 'dark'
export type Currency = 'RUB' | 'USD' | 'EUR'

export interface LocalOperation {
  id: string
  date: string
  category: string
  description: string
  amount: number
  type: OperationType
  account: string
}

export interface LocalBudget {
  id: string
  category: string
  limit: number
  spent: number
  period: string
}

export interface LocalGoal {
  id: string
  name: string
  target: number
  current: number
  deadline: string
  icon: string
  status?: string
}

export interface LocalAccount {
  id: string
  name: string
  balance: number
  type: string
}

export interface KPI {
  totalBalance: number
  monthIncome: number
  monthExpense: number
  savingsRate: number
}

export interface CategorySpending {
  category: string
  amount: number
  color: string
}

export interface Settings {
  theme: ThemeMode
  currency: Currency
  hideBalance: boolean
  name: string
  initials: string
  email: string
}

export interface ApiError {
  type?: string
  title?: string
  status?: number
  detail?: string
  message?: string
}

// Legacy PaginatedResponse — kept for backward compat, use ListResponse for new code
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
}
