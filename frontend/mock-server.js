import http from 'node:http'
import crypto from 'node:crypto'

const users = [
  { id: crypto.randomUUID(), email: 'demo@finhelper.ru', created_at: '2026-01-01T00:00:00.000Z' },
]
const tokens = {}
// Pre-login the demo user
const demoUser = users[0]
const demoRefresh = crypto.randomUUID() + crypto.randomUUID()
tokens[crypto.randomUUID() + crypto.randomUUID()] = { userId: demoUser.id, refresh: demoRefresh }
tokens[demoRefresh] = { userId: demoUser.id, kind: 'refresh' }
const accounts = [
  { id: 'a1', user_id: 'u1', name: 'Основной счёт', type: 'bank', balance: '150000.00', currency: 'RUB', created_at: new Date().toISOString() },
  { id: 'a2', user_id: 'u1', name: 'Наличные', type: 'cash', balance: '35000.00', currency: 'RUB', created_at: new Date().toISOString() },
]
const operations = []
const categories = [
  { id: 'cat1', name: 'Зарплата', type: 'income', parent_id: null, is_system: true },
  { id: 'cat2', name: 'Продукты', type: 'expense', parent_id: null, is_system: true },
  { id: 'cat3', name: 'Транспорт', type: 'expense', parent_id: null, is_system: true },
  { id: 'cat4', name: 'Жильё', type: 'expense', parent_id: null, is_system: true },
  { id: 'cat5', name: 'Развлечения', type: 'expense', parent_id: null, is_system: true },
  { id: 'cat6', name: 'Фриланс', type: 'income', parent_id: null, is_system: false },
]
const budgets = []
const goals = []

function makeTokens(userId) {
  const access = crypto.randomUUID() + crypto.randomUUID()
  const refresh = crypto.randomUUID() + crypto.randomUUID()
  tokens[access] = { userId, refresh }
  tokens[refresh] = { userId, kind: 'refresh' }
  return { access_token: access, refresh_token: refresh }
}

function parseBody(req) {
  return new Promise((resolve) => {
    let body = ''
    req.on('data', (chunk) => body += chunk)
    req.on('end', () => {
      try { resolve(JSON.parse(body)) }
      catch { resolve({}) }
    })
  })
}

function getUserId(req) {
  const auth = req.headers['authorization']
  if (!auth) return null
  const token = auth.replace('Bearer ', '')
  const t = tokens[token]
  return t ? t.userId : null
}

function writeJSON(res, code, data) {
  res.writeHead(code, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(data))
}

function writeProblem(res, code, detail) {
  res.writeHead(code, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify({ title: detail, status: code, detail }))
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://localhost')
  const path = url.pathname
  const method = req.method

  res.setHeader('Access-Control-Allow-Origin', '*')
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PATCH,DELETE,OPTIONS')
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization')

  if (method === 'OPTIONS') { res.writeHead(204); res.end(); return }

  // Health
  if (path === '/healthz' || path === '/readyz') { writeJSON(res, 200, { status: 'ok' }); return }

  try {
    // Auth: Register
    if (path === '/api/v1/auth/register' && method === 'POST') {
      const body = await parseBody(req)
      if (!body.email || !body.password) return writeProblem(res, 400, 'email and password required')
      if (users.find(u => u.email === body.email)) return writeProblem(res, 409, 'User already exists')
      const user = { id: crypto.randomUUID(), email: body.email, created_at: new Date().toISOString() }
      users.push(user)
      const t = makeTokens(user.id)
      writeJSON(res, 201, { user, ...t })
      return
    }

    // Auth: Login
    if (path === '/api/v1/auth/login' && method === 'POST') {
      const body = await parseBody(req)
      const user = users.find(u => u.email === body.email)
      if (!user) return writeProblem(res, 401, 'Invalid credentials')
      const t = makeTokens(user.id)
      writeJSON(res, 200, { user, ...t })
      return
    }

    // Auth: Refresh
    if (path === '/api/v1/auth/refresh' && method === 'POST') {
      const body = await parseBody(req)
      if (!body.refresh_token) return writeProblem(res, 400, 'refresh_token required')
      const t = tokens[body.refresh_token]
      if (!t || t.kind !== 'refresh') return writeProblem(res, 401, 'Invalid refresh token')
      delete tokens[body.refresh_token]
      const nt = makeTokens(t.userId)
      writeJSON(res, 200, nt)
      return
    }

    const userId = getUserId(req)
    if (!userId) return writeProblem(res, 401, 'Unauthorized')

    // Me
    if (path === '/api/v1/me' && method === 'GET') {
      const user = users.find(u => u.id === userId)
      if (!user) return writeProblem(res, 404, 'User not found')
      writeJSON(res, 200, user)
      return
    }

    // Dashboard
    if (path === '/api/v1/dashboard' && method === 'GET') {
      const income = operations.filter(o => o.user_id === userId && o.type === 'income').reduce((s, o) => s + parseFloat(o.amount), 0)
      const expenses = operations.filter(o => o.user_id === userId && o.type === 'expense').reduce((s, o) => s + parseFloat(o.amount), 0)
      const byCat = categories.filter(c => c.type === 'expense').map(c => {
        const amt = operations.filter(o => o.user_id === userId && o.category_id === c.id).reduce((s, o) => s + parseFloat(o.amount), 0)
        return { category: c.name, amount: String(amt), percentage: expenses > 0 ? (amt / expenses) * 100 : 0 }
      }).filter(c => parseFloat(c.amount) > 0)

      writeJSON(res, 200, {
        cashflow: { income: String(income), expenses: String(expenses), net: String(income - expenses) },
        expenses_by_category: byCat,
        net_worth: '185000.00',
        goal_progresses: goals.filter(g => g.user_id === userId).map(g => ({
          goal_id: g.id, name: g.name,
          progress: parseFloat(g.target_amount) > 0 ? (parseFloat(g.current_amount) / parseFloat(g.target_amount)) * 100 : 0,
        })),
      })
      return
    }

    // Accounts
    if (path === '/api/v1/accounts' && method === 'GET') {
      writeJSON(res, 200, accounts)
      return
    }

    // Operations list
    if (path === '/api/v1/operations' && method === 'GET') {
      const userOps = operations.filter(o => o.user_id === userId && !o.deleted_at)
      writeJSON(res, 200, { data: userOps, total: userOps.length, page: 1, per_page: 50 })
      return
    }

    // Operation create
    if (path === '/api/v1/operations' && method === 'POST') {
      const body = await parseBody(req)
      const op = {
        id: crypto.randomUUID(), user_id: userId, created_at: new Date().toISOString(), deleted_at: null,
        account_id: body.account_id, type: body.type, amount: body.amount, currency: body.currency || 'RUB',
        date: body.date, description: body.description || '', category_id: body.category_id || null,
        counterparty: body.counterparty || '', calc_id: body.calc_id,
      }
      operations.push(op)
      writeJSON(res, 201, op)
      return
    }

    // Operation delete
    const opDeleteMatch = path.match(/^\/api\/v1\/operations\/([a-f0-9-]+)$/)
    if (opDeleteMatch && method === 'DELETE') {
      const op = operations.find(o => o.id === opDeleteMatch[1] && o.user_id === userId)
      if (!op) return writeProblem(res, 404, 'Operation not found')
      op.deleted_at = new Date().toISOString()
      writeJSON(res, 204, null)
      return
    }

    // Categories
    if (path === '/api/v1/categories' && method === 'GET') {
      writeJSON(res, 200, categories)
      return
    }

    if (path === '/api/v1/categories' && method === 'POST') {
      const body = await parseBody(req)
      const cat = { id: crypto.randomUUID(), name: body.name, type: body.type, parent_id: body.parent_id || null, is_system: false }
      categories.push(cat)
      writeJSON(res, 201, cat)
      return
    }

    // Budgets
    if (path === '/api/v1/budgets' && method === 'GET') {
      writeJSON(res, 200, budgets.filter(b => b.user_id === userId))
      return
    }

    if (path === '/api/v1/budgets' && method === 'POST') {
      const body = await parseBody(req)
      const budget = {
        id: crypto.randomUUID(), user_id: userId, category_id: body.category_id,
        category_name: categories.find(c => c.id === body.category_id)?.name || body.category_id,
        amount: body.amount, period: body.period || 'monthly', rollover: body.rollover || 'none',
        created_at: new Date().toISOString(),
      }
      budgets.push(budget)
      writeJSON(res, 201, budget)
      return
    }

    // Budget status
    const budgetStatusMatch = path.match(/^\/api\/v1\/budgets\/([a-f0-9-]+)\/status$/)
    if (budgetStatusMatch && method === 'GET') {
      const budget = budgets.find(b => b.id === budgetStatusMatch[1] && b.user_id === userId)
      if (!budget) return writeProblem(res, 404, 'Budget not found')
      const spent = operations.filter(o => o.user_id === userId && o.category_id === budget.category_id && o.type === 'expense' && !o.deleted_at)
        .reduce((s, o) => s + parseFloat(o.amount || '0'), 0)
      const budgetAmt = parseFloat(budget.amount)
      writeJSON(res, 200, {
        budget_id: budget.id, category_name: budget.category_name, budget_amount: budget.amount,
        spent: String(spent), remaining: String(budgetAmt - spent),
        spent_pct: budgetAmt > 0 ? (spent / budgetAmt) * 100 : 0,
        projected_overspend: spent > budgetAmt ? String(spent - budgetAmt) : '0',
        days_remaining: 15,
      })
      return
    }

    // Budget delete
    const budgetDelMatch = path.match(/^\/api\/v1\/budgets\/([a-f0-9-]+)$/)
    if (budgetDelMatch && method === 'DELETE') {
      const idx = budgets.findIndex(b => b.id === budgetDelMatch[1] && b.user_id === userId)
      if (idx === -1) return writeProblem(res, 404, 'Budget not found')
      budgets.splice(idx, 1)
      writeJSON(res, 204, null)
      return
    }

    // Goals
    if (path === '/api/v1/goals' && method === 'GET') {
      writeJSON(res, 200, goals.filter(g => g.user_id === userId))
      return
    }

    if (path === '/api/v1/goals' && method === 'POST') {
      const body = await parseBody(req)
      const goal = {
        id: crypto.randomUUID(), user_id: userId, name: body.name,
        target_amount: body.target_amount, current_amount: '0', currency: body.currency || 'RUB',
        deadline: body.deadline, status: 'active', created_at: new Date().toISOString(),
      }
      goals.push(goal)
      writeJSON(res, 201, goal)
      return
    }

    // Goal contribution
    const contribMatch = path.match(/^\/api\/v1\/goals\/([a-f0-9-]+)\/contributions$/)
    if (contribMatch && method === 'POST') {
      const body = await parseBody(req)
      const goal = goals.find(g => g.id === contribMatch[1] && g.user_id === userId)
      if (!goal) return writeProblem(res, 404, 'Goal not found')
      const current = parseFloat(goal.current_amount)
      goal.current_amount = String(current + parseFloat(body.amount))
      const contrib = {
        id: crypto.randomUUID(), goal_id: goal.id, amount: body.amount,
        date: body.date || new Date().toISOString().split('T')[0],
        contribution_id: body.contribution_id || crypto.randomUUID(),
      }
      writeJSON(res, 201, contrib)
      return
    }

    // Goal projection
    const projMatch = path.match(/^\/api\/v1\/goals\/([a-f0-9-]+)\/projection$/)
    if (projMatch && method === 'GET') {
      const goal = goals.find(g => g.id === projMatch[1] && g.user_id === userId)
      if (!goal) return writeProblem(res, 404, 'Goal not found')
      const target = parseFloat(goal.target_amount)
      const current = parseFloat(goal.current_amount)
      const pct = target > 0 ? (current / target) * 100 : 0
      writeJSON(res, 200, {
        goal_id: goal.id, name: goal.name, target_amount: goal.target_amount,
        current_amount: goal.current_amount, monthly_contribution: '0',
        months_remaining: goal.deadline ? Math.max(0, Math.ceil((new Date(goal.deadline) - new Date()) / (1000 * 60 * 60 * 24 * 30))) : 0,
        progress_pct: pct, on_track: true,
      })
      return
    }

    // Goal simulate
    if (path === '/api/v1/goals/simulate' && method === 'POST') {
      const body = await parseBody(req)
      const target = parseFloat(body.target_amount)
      const current = parseFloat(body.current_amount || '0')
      const monthly = parseFloat(body.monthly_contribution)
      const months = body.deadline ? Math.max(0, Math.ceil((new Date(body.deadline) - new Date()) / (1000 * 60 * 60 * 24 * 30))) : 0
      const projected = current + monthly * months
      const pct = target > 0 ? (current / target) * 100 : 0
      writeJSON(res, 200, {
        name: body.name, target_amount: body.target_amount, current_amount: body.current_amount || '0',
        monthly_contribution: body.monthly_contribution, deadline: body.deadline,
        months_remaining: months, progress_pct: pct,
        on_track: projected >= target,
        projected_shortfall: projected >= target ? '0' : String(target - projected),
      })
      return
    }

    writeProblem(res, 404, 'Not found')
  } catch (err) {
    writeProblem(res, 500, err.message)
  }
})

const PORT = 8080
server.listen(PORT, () => {
  console.log(`Mock FinHelper API running on http://localhost:${PORT}`)
})
