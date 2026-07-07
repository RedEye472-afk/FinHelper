import { apiRequest } from './client'
import type { DashboardData } from '../types'

export async function getDashboard(period = 'month'): Promise<DashboardData> {
  return apiRequest<DashboardData>('GET', `/api/v1/dashboard?period=${period}`)
}
