import { apiRequest } from './client'
import type { Operation, OperationCreate, ListResponse } from '../types'

export async function listOperations(limit = 50, before?: number): Promise<ListResponse<Operation>> {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  if (before) params.set('before', String(before))
  return apiRequest<ListResponse<Operation>>('GET', `/api/v1/operations?${params}`)
}

export async function createOperation(data: OperationCreate): Promise<Operation> {
  return apiRequest<Operation>('POST', '/api/v1/operations', data)
}

export async function deleteOperation(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/api/v1/operations/${id}`)
}
