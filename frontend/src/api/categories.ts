import { apiRequest } from './client'
import type { Category, ListResponse } from '../types'

export async function listCategories(): Promise<Category[]> {
  const res = await apiRequest<ListResponse<Category>>('GET', '/api/v1/categories')
  return res.items
}

export async function createCategory(name: string, parentId?: number): Promise<Category> {
  return apiRequest<Category>('POST', '/api/v1/categories', { name, parent_id: parentId })
}
