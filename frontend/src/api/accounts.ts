import { apiRequest } from './client'
import type { Account, AccountCreate, ListResponse } from '../types'

export async function listAccounts(): Promise<Account[]> {
  const res = await apiRequest<ListResponse<Account>>('GET', '/api/v1/accounts')
  return res.items
}

export async function getAccount(id: number): Promise<Account> {
  return apiRequest<Account>('GET', `/api/v1/accounts/${id}`)
}

export async function createAccount(data: AccountCreate): Promise<Account> {
  return apiRequest<Account>('POST', '/api/v1/accounts', { name: data.name, type: data.type, currency: data.currency })
}

export async function updateAccount(id: number, data: Partial<AccountCreate>): Promise<Account> {
  return apiRequest<Account>('PATCH', `/api/v1/accounts/${id}`, data)
}

export async function deleteAccount(id: number): Promise<void> {
  return apiRequest<void>('DELETE', `/api/v1/accounts/${id}`)
}
