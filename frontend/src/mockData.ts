import type { LocalOperation, LocalBudget, LocalGoal, LocalAccount } from './types'

/** Deterministic PRNG */
function createRng(seed: number) {
  let s = seed | 0
  return () => {
    s = (s * 1664525 + 1013904223) | 0
    return (s >>> 0) / 4294967296
  }
}

/** Generate realistic mock operations for the last 12 months */
function generateMockData(seed: number): { operations: LocalOperation[] } {
  const rng = createRng(seed)
  const pick = <T>(arr: T[]): T => arr[Math.floor(rng() * arr.length)]
  const rand = (min: number, max: number) => Math.floor(rng() * (max - min + 1)) + min
  const ops: LocalOperation[] = []
  let id = 0
  const now = new Date()

  const makeDate = (year: number, month: number, day: number) =>
    new Date(year, month, day).toISOString().slice(0, 10)

  for (let mo = 11; mo >= 0; mo--) {
    const m = now.getMonth() - mo
    const y = now.getFullYear() + (m < 0 ? -1 : 0)
    const mon = ((m % 12) + 12) % 12
    const dim = new Date(y, mon + 1, 0).getDate()
    const day = (d: number) => makeDate(y, mon, Math.min(d, dim))
    const nextId = () => `o${++id}`

    // Income
    ops.push({ id: nextId(), date: day(10), category: 'Зарплата', description: 'Зарплата', amount: rand(140000, 160000), type: 'income', account: 'Тинькофф Black' })
    const fl = rand(1, 2)
    for (let f = 0; f < fl; f++)
      ops.push({ id: nextId(), date: day(rand(12, 26)), category: 'Фриланс', description: pick(['Вёрстка сайта', 'Дизайн макета', 'Консультация', 'Разработка API']), amount: rand(25000, 70000), type: 'income', account: 'Тинькофф Black' })
    if (rng() > 0.65)
      ops.push({ id: nextId(), date: day(rand(1, 5)), category: 'Прочее', description: 'Кешбэк', amount: rand(300, 1200), type: 'income', account: 'Тинькофф Black' })

    // Fixed
    ops.push({ id: nextId(), date: day(1), category: 'Жильё', description: 'Аренда квартиры', amount: 35000, type: 'expense', account: 'Тинькофф Black' })
    ops.push({ id: nextId(), date: day(rand(12, 18)), category: 'Жильё', description: 'Коммунальные платежи', amount: rand(3500, 5500), type: 'expense', account: 'Тинькофф Black' })
    ops.push({ id: nextId(), date: day(rand(3, 7)), category: 'Связь', description: 'Тинькофф Мобайл', amount: 599, type: 'expense', account: 'Тинькофф Black' })
    ops.push({ id: nextId(), date: day(rand(8, 12)), category: 'Связь', description: 'Интернет', amount: rand(450, 650), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.3)
      ops.push({ id: nextId(), date: day(rand(1, 5)), category: 'Подписки', description: pick(['YouTube Premium', 'Яндекс Плюс', 'Netflix']), amount: pick([199, 299, 499]), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.4)
      ops.push({ id: nextId(), date: day(rand(1, 10)), category: 'Спорт', description: pick(['Фитнес-клуб', 'Бассейн', 'Йога']), amount: pick([2500, 3000]), type: 'expense', account: 'Тинькофф Black' })

    // Variable
    for (let g = 0, gc = rand(8, 12); g < gc; g++)
      ops.push({ id: nextId(), date: day(rand(1, dim)), category: 'Продукты', description: pick(['Пятёрочка', 'Магнит', 'Ашан', 'ВкусВилл']), amount: g % 3 === 0 ? rand(2000, 5000) : rand(400, 1500), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })
    for (let r = 0, rc = rand(2, 4); r < rc; r++)
      ops.push({ id: nextId(), date: day(rand(5, 26)), category: 'Рестораны', description: pick(['Суши Wok', 'KFC', 'Burger King', 'Додо Пицца']), amount: rand(800, 3500), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })
    for (let t = 0, tc = rand(6, 10); t < tc; t++)
      ops.push({ id: nextId(), date: day(rand(1, dim)), category: 'Транспорт', description: pick(['Метро', 'Яндекс Go', 'Автобус', 'Такси']), amount: pick([45, 65, 300, 450, 550, 800]), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })

    ;[6, 13, 20, 27].filter(ed => ed <= dim).forEach(ed => {
      if (rng() > 0.35)
        ops.push({ id: nextId(), date: day(ed), category: 'Развлечения', description: pick(['Кино', 'Боулинг', 'Квест', 'Концерт', 'Бар']), amount: rand(500, 4000), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })
    })

    if (rng() > 0.4)
      ops.push({ id: nextId(), date: day(rand(5, 25)), category: 'Здоровье', description: pick(['Аптека', 'Стоматолог', 'Чек-ап', 'Витамины']), amount: rand(800, 5000), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })
    if (rng() > 0.5)
      ops.push({ id: nextId(), date: day(rand(5, 25)), category: 'Одежда', description: pick(['Zara', 'H&M', 'Uniqlo', 'Lamoda']), amount: rand(3000, 15000), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.7)
      ops.push({ id: nextId(), date: day(rand(10, 20)), category: 'Образование', description: pick(['Stepik курс', 'Книги', 'Курсы англ.']), amount: rand(1000, 8000), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.7)
      ops.push({ id: nextId(), date: day(rand(20, 28)), category: 'Подарки', description: pick(['Подарок другу', 'Цветы']), amount: rand(1000, 5000), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.5)
      ops.push({ id: nextId(), date: day(rand(5, 20)), category: 'Прочее', description: pick(['Бытовая химия', 'Косметика', 'Хозтовары']), amount: rand(300, 3000), type: 'expense', account: pick(['Тинькофф Black', 'Наличные']) })

    if (rng() > 0.85 && mo > 0)
      ops.push({ id: nextId(), date: day(rand(10, 25)), category: 'Прочее', description: pick(['Техника', 'Мебель', 'Авиабилеты']), amount: rand(15000, 60000), type: 'expense', account: 'Тинькофф Black' })
    if (rng() > 0.8)
      ops.push({ id: nextId(), date: day(rand(10, 20)), category: 'Здоровье', description: 'Страховка', amount: rand(3000, 8000), type: 'expense', account: 'Тинькофф Black' })

    if (rng() > 0.3) {
      const amt = rand(15000, 35000)
      ops.push({ id: nextId(), date: day(rand(12, 15)), category: 'Переводы', description: 'Пополнение вклада', amount: amt, type: 'expense', account: 'Тинькофф Black' })
      ops.push({ id: nextId(), date: day(rand(12, 15)), category: 'Переводы', description: 'Пополнение вклада', amount: amt, type: 'income', account: 'Сбер Вклад' })
    }
  }

  ops.sort((a, b) => a.date.localeCompare(b.date) || a.id.localeCompare(b.id))
  return { operations: ops }
}

// Lazy singleton - generated once, cached forever
let _generated: LocalOperation[] | null = null

/** Returns mock operations, generating them lazily on first call */
export function getMockOperations(): LocalOperation[] {
  if (!_generated) {
    try {
      _generated = generateMockData(42).operations
    } catch (e) {
      console.warn('Mock data generation failed:', e)
      _generated = []
    }
  }
  return _generated
}

/** Pre-generated mock operations for direct import (initializes on first module access) */
export const mockOperations: LocalOperation[] = getMockOperations()

export const categoryColors: Record<string, string> = {
  Зарплата: '#10b981', Фриланс: '#34d399', Продукты: '#f59e0b', Транспорт: '#3b82f6',
  Рестораны: '#f97316', Жильё: '#ef4444', Развлечения: '#8b5cf6', Здоровье: '#ec4899',
  Связь: '#06b6d4', Одежда: '#6366f1', Образование: '#14b8a6', Подарки: '#f43f5e',
  Спорт: '#a855f7', Подписки: '#0ea5e9', Переводы: '#64748b', Прочее: '#6b7280',
}

export const mockBudgets: LocalBudget[] = [
  { id: 'b1', category: 'Продукты', limit: 35000, spent: 0, period: 'monthly' },
  { id: 'b2', category: 'Транспорт', limit: 7000, spent: 0, period: 'monthly' },
  { id: 'b3', category: 'Рестораны', limit: 12000, spent: 0, period: 'monthly' },
  { id: 'b4', category: 'Развлечения', limit: 10000, spent: 0, period: 'monthly' },
  { id: 'b5', category: 'Одежда', limit: 15000, spent: 0, period: 'monthly' },
  { id: 'b6', category: 'Жильё', limit: 45000, spent: 0, period: 'monthly' },
  { id: 'b7', category: 'Здоровье', limit: 8000, spent: 0, period: 'monthly' },
  { id: 'b8', category: 'Связь', limit: 2500, spent: 0, period: 'monthly' },
  { id: 'b9', category: 'Спорт', limit: 5000, spent: 0, period: 'monthly' },
  { id: 'b10', category: 'Подписки', limit: 2000, spent: 0, period: 'monthly' },
  { id: 'b11', category: 'Образование', limit: 10000, spent: 0, period: 'monthly' },
]

export const mockGoals: LocalGoal[] = [
  { id: 'g1', name: 'Подушка безопасности', target: 500000, current: 250000, deadline: '2026-12-31', icon: '🛡' },
  { id: 'g2', name: 'Новый ноутбук', target: 150000, current: 80000, deadline: '2026-09-30', icon: '💻' },
  { id: 'g3', name: 'Отпуск в Таиланде', target: 200000, current: 60000, deadline: '2026-11-30', icon: '✈' },
  { id: 'g4', name: 'Инвестиционный портфель', target: 1000000, current: 150000, deadline: '2027-06-30', icon: '📈' },
]

export const mockAccounts: LocalAccount[] = [
  { id: 'a1', name: 'Наличные', balance: 45000, type: 'cash' },
  { id: 'a2', name: 'Тинькофф Black', balance: 234500, type: 'checking' },
  { id: 'a3', name: 'Сбер Вклад', balance: 500000, type: 'savings' },
]
