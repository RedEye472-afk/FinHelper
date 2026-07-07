import { apiRequest } from './client'
import type { Goal, GoalCreate, GoalProjection, ListResponse } from '../types'

export async function listGoals(): Promise<Goal[]> {
  const res = await apiRequest<ListResponse<Goal>>('GET', '/api/v1/goals')
  return res.items
}

export async function createGoal(data: GoalCreate): Promise<Goal> {
  return apiRequest<Goal>('POST', '/api/v1/goals', data)
}

export async function getGoal(id: number): Promise<Goal> {
  return apiRequest<Goal>('GET', `/api/v1/goals/${id}`)
}

export async function updateGoal(id: number, data: GoalCreate): Promise<Goal> {
  return apiRequest<Goal>('PATCH', `/api/v1/goals/${id}`, data)
}

export async function deleteGoal(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/api/v1/goals/${id}`)
}

export async function getProjection(goalId: number): Promise<GoalProjection> {
  return apiRequest<GoalProjection>('GET', `/api/v1/goals/${goalId}/projection`)
}

export async function simulate(data: {
  current_amount?: string
  target_amount: string
  monthly_contribution?: string
  target_date?: string
  expected_yield?: string
  inflation?: string
}): Promise<GoalProjection> {
  return apiRequest<GoalProjection>('POST', '/api/v1/calc/goal', data)
}

export async function simulateSaved(goalId: number, data: {
  current_amount?: string
  target_amount: string
  monthly_contribution?: string
  target_date?: string
  expected_yield?: string
  inflation?: string
}): Promise<GoalProjection> {
  return apiRequest<GoalProjection>('POST', `/api/v1/goals/${goalId}/simulate`, data)
}

export async function listContributions(goalId: number): Promise<ContributionResponse[]> {
  const res = await apiRequest<ListResponse<ContributionResponse>>('GET', `/api/v1/goals/${goalId}/contributions`)
  return res.items
}

export async function createContribution(goalId: number, data: {
  contribution_id: string
  amount: string
  date?: string
  comment?: string
}): Promise<ContributionResponse> {
  return apiRequest<ContributionResponse>('POST', `/api/v1/goals/${goalId}/contributions`, data)
}

export async function deleteContribution(goalId: number, cid: number): Promise<void> {
  return apiRequest<void>('DELETE', `/api/v1/goals/${goalId}/contributions/${cid}`)
}

export interface ContributionResponse {
  id: number
  goal_id: number
  contribution_id: string
  amount: string
  date: string
  comment?: string
  created_at: string
}
