/**
 * queries.ts — React Query hooks for FinHelper.
 * All requests go through API client (client.ts). Money values are strings.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as accountsApi from './accounts'
import * as operationsApi from './operations'
import * as dashboardApi from './dashboard'
import * as budgetsApi from './budgets'
import * as goalsApi from './goalsApi'
import * as categoriesApi from './categories'
import * as calculatorsApi from './calculators'
import type {
  Account, AccountCreate, Operation, OperationCreate, ListResponse,
  DashboardData, Budget, BudgetCreate, BudgetStatus,
  Category, Goal, GoalCreate, GoalProjection,
} from '../types'

// ── Accounts ──
export function useAccounts() {
  return useQuery({
    queryKey: ['accounts'],
    queryFn: () => accountsApi.listAccounts(),
  })
}

export function useCreateAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: AccountCreate) => accountsApi.createAccount(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts'] }),
  })
}

export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => accountsApi.deleteAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts'] }),
  })
}

// ── Operations ──
export function useOperations(limit = 50, before?: number) {
  return useQuery({
    queryKey: ['operations', limit, before],
    queryFn: () => operationsApi.listOperations(limit, before),
  })
}

export function useCreateOperation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: OperationCreate) => operationsApi.createOperation(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['operations'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

export function useDeleteOperation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => operationsApi.deleteOperation(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['operations'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}

// ── Dashboard ──
export function useDashboard(period = 'month') {
  return useQuery({
    queryKey: ['dashboard', period],
    queryFn: () => dashboardApi.getDashboard(period),
  })
}

// ── Budgets ──
export function useBudgets() {
  return useQuery({
    queryKey: ['budgets'],
    queryFn: () => budgetsApi.listBudgets(),
  })
}

export function useCreateBudget() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: BudgetCreate) => budgetsApi.createBudget(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['budgets'] }),
  })
}

export function useDeleteBudget() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => budgetsApi.deleteBudget(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['budgets'] }),
  })
}

export function useBudgetStatus(id: number) {
  return useQuery({
    queryKey: ['budgetStatus', id],
    queryFn: () => budgetsApi.getBudgetStatus(id),
    enabled: id > 0,
  })
}

// ── Categories ──
export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.listCategories(),
  })
}

// ── Goals ──
export function useGoals() {
  return useQuery({
    queryKey: ['goals'],
    queryFn: () => goalsApi.listGoals(),
  })
}

export function useCreateGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: GoalCreate) => goalsApi.createGoal(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })
}

export function useContributeGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { goalId: number; data: { contribution_id: string; amount: string; date?: string; comment?: string } }) =>
      goalsApi.createContribution(input.goalId, input.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })
}

export function useGoalProjection(goalId: number, enabled = false) {
  return useQuery({
    queryKey: ['goalProjection', goalId],
    queryFn: () => goalsApi.getProjection(goalId),
    enabled: goalId > 0 && enabled,
  })
}

// ── Calculators (unchanged) ──
export function useCreditCalc() {
  return useMutation({
    mutationFn: calculatorsApi.calculateCredit,
  })
}

export function useAffordabilityCalc() {
  return useMutation({
    mutationFn: calculatorsApi.calculateAffordability,
  })
}
