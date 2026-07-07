import { apiRequest } from './client'
import type { Budget, BudgetCreate, BudgetStatus, ListResponse } from '../types'

export async function listBudgets(): Promise<Budget[]> {
  const res = await apiRequest<ListResponse<Budget>>('GET', '/api/v1/budgets')
  return res.items
}

export async function createBudget(data: BudgetCreate): Promise<Budget> {
  return apiRequest<Budget>('POST', '/api/v1/budgets', data)
}

export async function deleteBudget(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/api/v1/budgets/${id}`)
}

export async function getBudgetStatus(id: number): Promise<BudgetStatus> {
  return apiRequest<BudgetStatus>('GET', `/api/v1/budgets/${id}/status`)
}
