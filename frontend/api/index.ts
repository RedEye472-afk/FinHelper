import type { VercelRequest, VercelResponse } from '@vercel/node'
import crypto from 'node:crypto'

// ── In-memory data store ──

type User = { id: string; email: string; created_at: string }
type Account = { id: number; name: string; account_type: string; currency: string; balance: string; created_at: string; updated_at?: string }
type Operation = { id: number; calc_id: string; operation_type: string; amount: string; account_id: number; category_id?: number | null; counterparty?: string; description?: string; operation_date: string; is_planned: boolean; created_at: string; deleted_at?: string | null }
type Category = { id: number; name: string; parent_id: number | null; is_system: boolean }
type Budget = { id: number; user_id: string; category_id: number; limit_amount: string; rollover_policy: string; is_active: boolean; created_at: string }
type Goal = { id: number; user_id: string; name: string; target_amount: string; current_amount: string; monthly_contribution: string | null; target_date: string | null; expected_yield: string; created_at: string }

let nextId = 100
const userId = 'demo-user-0001'
const user: User = { id: userId, email: 'demo@finhelper.ru', created_at: '2026-01-01T00:00:00.000Z' }
const users: Record<string, User> = { [userId]: user }
const tokens: Record<string, { userId: string; refresh?: string }> = {}
const accounts: Account[] = [
  { id: ++nextId, name: 'Основной счёт', account_type: 'bank', currency: 'RUB', balance: '150000.00', created_at: '2026-01-01T00:00:00Z' },
  { id: ++nextId, name: 'Наличные', account_type: 'cash', currency: 'RUB', balance: '35000.00', created_at: '2026-01-01T00:00:00Z' },
]
const operations: Operation[] = [
  { id: ++nextId, calc_id: 'srv:demo:1', operation_type: 'income', amount: '85000.00', account_id: 101, category_id: 1, counterparty: '', description: 'Зарплата за июль', operation_date: '2026-07-01', is_planned: false, created_at: '2026-07-01T10:00:00Z' },
  { id: ++nextId, calc_id: 'srv:demo:2', operation_type: 'expense', amount: '3200.50', account_id: 102, category_id: 2, counterparty: 'Пятёрочка', description: '', operation_date: '2026-07-03', is_planned: false, created_at: '2026-07-03T12:30:00Z' },
  { id: ++nextId, calc_id: 'srv:demo:3', operation_type: 'expense', amount: '15000.00', account_id: 101, category_id: 4, counterparty: '', description: 'Аренда квартиры', operation_date: '2026-07-05', is_planned: false, created_at: '2026-07-05T09:00:00Z' },
  { id: ++nextId, calc_id: 'srv:demo:4', operation_type: 'expense', amount: '850.00', account_id: 102, category_id: 3, counterparty: 'Метро', description: '', operation_date: '2026-07-06', is_planned: false, created_at: '2026-07-06T08:15:00Z' },
  { id: ++nextId, calc_id: 'srv:demo:5', operation_type: 'income', amount: '15000.00', account_id: 101, category_id: 6, counterparty: '', description: 'Фриланс-проект', operation_date: '2026-07-10', is_planned: false, created_at: '2026-07-10T14:00:00Z' },
]
const categories: Category[] = [
  { id: 1, name: 'Зарплата', parent_id: null, is_system: true },
  { id: 2, name: 'Продукты', parent_id: null, is_system: true },
  { id: 3, name: 'Транспорт', parent_id: null, is_system: true },
  { id: 4, name: 'Жильё', parent_id: null, is_system: true },
  { id: 5, name: 'Развлечения', parent_id: null, is_system: true },
  { id: 6, name: 'Фриланс', parent_id: null, is_system: false },
]
const budgets: Budget[] = [
  { id: ++nextId, user_id: userId, category_id: 2, limit_amount: '25000.00', rollover_policy: 'none', is_active: true, created_at: '2026-07-01T00:00:00Z' },
  { id: ++nextId, user_id: userId, category_id: 4, limit_amount: '35000.00', rollover_policy: 'none', is_active: true, created_at: '2026-07-01T00:00:00Z' },
]
const goals: Goal[] = [
  { id: ++nextId, user_id: userId, name: 'Накопить на машину', target_amount: '1500000.00', current_amount: '250000.00', monthly_contribution: '50000.00', target_date: '2027-06-01', expected_yield: '0.00', created_at: '2026-01-15T00:00:00Z' },
  { id: ++nextId, user_id: userId, name: 'Подушка безопасности', target_amount: '500000.00', current_amount: '150000.00', monthly_contribution: '25000.00', target_date: '2026-12-31', expected_yield: '0.08', created_at: '2026-02-01T00:00:00Z' },
]

// ── Helpers ──

function makeTokens(uid: string) {
  const access = crypto.randomUUID() + crypto.randomUUID()
  const refresh = crypto.randomUUID() + crypto.randomUUID()
  tokens[access] = { userId: uid, refresh }
  tokens[refresh] = { userId: uid }
  return { access_token: access, refresh_token: refresh, token_type: 'bearer', expires_in: 900 }
}
function getUserId(req: VercelRequest): string | null {
  const auth = req.headers['authorization']
  if (!auth) return null
  const token = String(auth).replace('Bearer ', '')
  const t = tokens[token]
  return t ? t.userId : null
}
function makeList<T>(items: T[]): { items: T[]; more: boolean } {
  return { items, more: false }
}
function catName(id: number): string {
  return categories.find(c => c.id === id)?.name ?? 'Unknown'
}
function categoryExpenses(catId: number): string {
  const total = operations
    .filter(o => o.category_id === catId && o.operation_type === 'expense' && !o.deleted_at)
    .reduce((s, o) => s + parseFloat(o.amount), 0)
  return total.toFixed(2)
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  res.setHeader('Access-Control-Allow-Origin', '*')
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PATCH,DELETE,OPTIONS')
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization')
  if (req.method === 'OPTIONS') { res.status(204).end(); return }

  const path = (req.query.path as string) || '/'
  const method = req.method || 'GET'

  try {
    // ── Auth (no token required) ──
    if (path === '/api/v1/auth/register' && method === 'POST') {
      const body = req.body || {}
      if (!body.email || !body.password) return res.status(400).json({ detail: 'email and password required' })
      if (Object.values(users).find(u => u.email === body.email)) return res.status(409).json({ detail: 'User already exists' })
      const newUser: User = { id: crypto.randomUUID(), email: body.email, created_at: new Date().toISOString() }
      users[newUser.id] = newUser
      return res.status(201).json(makeTokens(newUser.id))
    }
    if (path === '/api/v1/auth/login' && method === 'POST') {
      const body = req.body || {}
      const found = Object.values(users).find(u => u.email === body.email)
      if (!found) return res.status(401).json({ detail: 'Invalid credentials' })
      return res.status(200).json(makeTokens(found.id))
    }
    if (path === '/api/v1/auth/refresh' && method === 'POST') {
      const body = req.body || {}
      if (!body.refresh_token) return res.status(400).json({ detail: 'refresh_token required' })
      const t = tokens[body.refresh_token]
      if (!t) return res.status(401).json({ detail: 'Invalid refresh token' })
      delete tokens[body.refresh_token]
      return res.status(200).json(makeTokens(t.userId))
    }

    // ── Authenticated routes ──
    const uid = getUserId(req)
    if (!uid || !users[uid]) return res.status(401).json({ detail: 'Unauthorized' })

    if (path === '/api/v1/me' && method === 'GET') return res.status(200).json(users[uid])

    // ── Dashboard ──
    if (path === '/api/v1/dashboard' && method === 'GET') {
      const income = operations.filter(o => o.operation_type === 'income' && !o.deleted_at).reduce((s, o) => s + parseFloat(o.amount), 0)
      const expense = operations.filter(o => o.operation_type === 'expense' && !o.deleted_at).reduce((s, o) => s + parseFloat(o.amount), 0)
      const byCategory = categories.map(c => {
        const total = categoryExpenses(c.id)
        return { category_id: c.id, category_name: c.name, total }
      }).filter(c => parseFloat(c.total) > 0)
      return res.status(200).json({
        period: 'month', from: '2026-07-01', to: '2026-07-31',
        income: income.toFixed(2), expense: expense.toFixed(2), net: (income - expense).toFixed(2),
        by_category: byCategory,
        net_worth: { assets: '185000.00', debts: '0.00', net: '185000.00' },
        goals: goals.map(g => ({
          id: g.id, name: g.name,
          target: g.target_amount, current: g.current_amount,
          progress: parseFloat(g.target_amount) > 0 ? Math.round((parseFloat(g.current_amount) / parseFloat(g.target_amount)) * 100) : 0,
        })),
      })
    }

    // ── Accounts ──
    if (path === '/api/v1/accounts' && method === 'GET') return res.status(200).json(makeList(accounts))
    if (path === '/api/v1/accounts' && method === 'POST') {
      const body = req.body || {}
      const acc: Account = {
        id: ++nextId, name: body.name, account_type: body.type || body.account_type || 'bank',
        currency: body.currency || 'RUB', balance: '0.00', created_at: new Date().toISOString(),
      }
      accounts.push(acc)
      return res.status(201).json(acc)
    }
    const accMatch = path.match(/^\/api\/v1\/accounts\/(\d+)$/)
    if (accMatch && method === 'GET') {
      const acc = accounts.find(a => a.id === parseInt(accMatch[1]))
      if (!acc) return res.status(404).json({ detail: 'Account not found' })
      return res.status(200).json(acc)
    }
    if (accMatch && method === 'PATCH') {
      const acc = accounts.find(a => a.id === parseInt(accMatch[1]))
      if (!acc) return res.status(404).json({ detail: 'Account not found' })
      const body = req.body || {}
      if (body.name) acc.name = body.name
      if (body.type || body.account_type) acc.account_type = body.type || body.account_type
      acc.updated_at = new Date().toISOString()
      return res.status(200).json(acc)
    }
    if (accMatch && method === 'DELETE') {
      const idx = accounts.findIndex(a => a.id === parseInt(accMatch[1]))
      if (idx === -1) return res.status(404).json({ detail: 'Account not found' })
      accounts.splice(idx, 1)
      return res.status(204).end()
    }

    // ── Operations ──
    if (path === '/api/v1/operations' && method === 'GET') {
      const limit = parseInt(req.query.limit as string) || 50
      const before = parseInt(req.query.before as string) || 0
      let filtered = operations.filter(o => !o.deleted_at).sort((a, b) => b.id - a.id)
      if (before > 0) filtered = filtered.filter(o => o.id < before)
      const page = filtered.slice(0, limit)
      return res.status(200).json({ items: page, more: filtered.length > limit })
    }
    if (path === '/api/v1/operations' && method === 'POST') {
      const body = req.body || {}
      const op: Operation = {
        id: ++nextId, calc_id: body.calc_id || `srv:${uid}:${Date.now()}`,
        operation_type: body.operation_type, amount: body.amount,
        account_id: body.account_id, category_id: body.category_id || null,
        counterparty: body.counterparty || '', description: body.description || '',
        operation_date: body.operation_date || new Date().toISOString().split('T')[0],
        is_planned: false, created_at: new Date().toISOString(),
      }
      operations.push(op)
      return res.status(201).json(op)
    }
    const opDel = path.match(/^\/api\/v1\/operations\/(-?\d+)$/)
    if (opDel && method === 'DELETE') {
      const op = operations.find(o => o.id === parseInt(opDel[1]))
      if (!op) return res.status(404).json({ detail: 'Operation not found' })
      op.deleted_at = new Date().toISOString()
      return res.status(204).end()
    }

    // ── Categories ──
    if (path === '/api/v1/categories' && method === 'GET') return res.status(200).json(makeList(categories))
    if (path === '/api/v1/categories' && method === 'POST') {
      const body = req.body || {}
      const cat: Category = { id: ++nextId, name: body.name, parent_id: body.parent_id ?? null, is_system: false }
      categories.push(cat)
      return res.status(201).json(cat)
    }

    // ── Budgets ──
    if (path === '/api/v1/budgets' && method === 'GET') {
      const list = budgets.filter(b => b.user_id === uid).map(b => ({ ...b, category_name: catName(b.category_id) }))
      return res.status(200).json(makeList(list))
    }
    if (path === '/api/v1/budgets' && method === 'POST') {
      const body = req.body || {}
      const b: Budget = {
        id: ++nextId, user_id: uid, category_id: body.category_id,
        limit_amount: body.limit_amount, rollover_policy: body.rollover_policy || 'none',
        is_active: true, created_at: new Date().toISOString(),
      }
      budgets.push(b)
      return res.status(201).json({ ...b, category_name: catName(b.category_id) })
    }
    const budStat = path.match(/^\/api\/v1\/budgets\/(-?\d+)\/status$/)
    if (budStat && method === 'GET') {
      const b = budgets.find(x => x.id === parseInt(budStat[1]) && x.user_id === uid)
      if (!b) return res.status(404).json({ detail: 'Budget not found' })
      const spent = parseFloat(categoryExpenses(b.category_id))
      const limit = parseFloat(b.limit_amount)
      return res.status(200).json({
        budget_id: b.id, spent: spent.toFixed(2), remaining: (limit - spent).toFixed(2),
        spent_pct: limit > 0 ? (spent / limit) * 100 : 0, days_remaining: 15,
      })
    }
    const budDel = path.match(/^\/api\/v1\/budgets\/(-?\d+)$/)
    if (budDel && method === 'DELETE') {
      const idx = budgets.findIndex(x => x.id === parseInt(budDel[1]) && x.user_id === uid)
      if (idx === -1) return res.status(404).json({ detail: 'Budget not found' })
      budgets.splice(idx, 1)
      return res.status(204).end()
    }

    // ── Goals ──
    if (path === '/api/v1/goals' && method === 'GET') return res.status(200).json(makeList(goals.filter(g => g.user_id === uid)))
    if (path === '/api/v1/goals' && method === 'POST') {
      const body = req.body || {}
      const g: Goal = {
        id: ++nextId, user_id: uid, name: body.name,
        target_amount: body.target_amount, current_amount: body.current_amount || '0',
        monthly_contribution: body.monthly_contribution || null,
        target_date: body.target_date || null, expected_yield: body.expected_yield || '0',
        created_at: new Date().toISOString(),
      }
      goals.push(g)
      return res.status(201).json(g)
    }
    const goalProj = path.match(/^\/api\/v1\/goals\/(-?\d+)\/project$/)
    if (goalProj && method === 'POST') {
      const g = goals.find(x => x.id === parseInt(goalProj[1]) && x.user_id === uid)
      if (!g) return res.status(404).json({ detail: 'Goal not found' })
      return res.status(200).json({
        monthly_contribution: g.monthly_contribution || '0',
        months_remaining: g.target_date ? Math.max(0, Math.ceil((new Date(g.target_date).getTime() - Date.now()) / 2592000000)) : 0,
        total_contributed: g.current_amount, final_amount: (parseFloat(g.current_amount) + 0).toFixed(2),
      })
    }

    // ── Contributions ──
    if (path === '/api/v1/contributions' && method === 'POST') {
      const body = req.body || {}
      const g = goals.find(x => x.id === body.goal_id && x.user_id === uid)
      if (!g) return res.status(404).json({ detail: 'Goal not found' })
      g.current_amount = (parseFloat(g.current_amount) + parseFloat(body.amount)).toFixed(2)
      return res.status(201).json({
        id: ++nextId, goal_id: g.id, amount: body.amount,
        contribution_date: body.contribution_date || new Date().toISOString().split('T')[0],
        contribution_id: body.contribution_id || crypto.randomUUID(),
      })
    }

    return res.status(404).json({ detail: 'Not found', path, method })
  } catch (err: unknown) {
    return res.status(500).json({ detail: err instanceof Error ? err.message : 'Internal error' })
  }
}
