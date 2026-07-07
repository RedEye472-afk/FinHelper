import { apiRequest } from './client'

export interface CreditRequest {
  principal: string
  annual_rate?: string
  term_months: number
  payment_type?: string
  first_payment_date?: string
  upfront_fees?: string[]
  monthly_fee?: string
  early?: { paid_months: number; amount: string; mode?: string }
}

export interface CreditScheduleItem {
  month: number
  payment: string
  principal_part: string
  interest_part: string
  remaining: string
}

export interface CreditResponse {
  payment_type: string
  monthly_payment: string
  psk: string
  overpayment: string
  schedule: CreditScheduleItem[]
  disclaimer: string
  early?: unknown
}

export interface AffordabilityRequest {
  monthly_incomes: string[]
  mandatory_expenses?: string
  existing_loan_payments?: string
  cushion?: string
  new_loan: {
    principal: string
    annual_rate?: string
    term_months: number
  }
  config?: {
    min_share?: string
    max_share?: string
    stress_drop?: string
    min_cushion_months?: number
    existing_debt_refusal_share?: string
  }
}

export interface AffordabilityStressItem {
  scenario: string
  income_drop_pct: string
  effective_income: string
  free_remainder: string
  new_payment: string
  payment_share_pct: string
  risk: string
}

export interface AffordabilityResponse {
  stable_income: string
  free_remainder: string
  safe_payment_range: { low: string; high: string }
  new_payment: string
  cushion_months: number
  stress: AffordabilityStressItem[]
  risk: string
  scenario: string
  refused_reason?: string
}

export async function calculateCredit(data: CreditRequest): Promise<CreditResponse> {
  return apiRequest<CreditResponse>('POST', '/api/v1/calc/credit', data)
}

export async function calculateAffordability(data: AffordabilityRequest): Promise<AffordabilityResponse> {
  return apiRequest<AffordabilityResponse>('POST', '/api/v1/calc/affordability', data)
}