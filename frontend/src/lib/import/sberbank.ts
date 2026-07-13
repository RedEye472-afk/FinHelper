/**
 * Sberbank statement parser
 *
 * Два формата:
 * 1. parseSberbankText() — вертикальный (каждое поле с новой строки, для copy-paste)
 * 2. parseSberbankInline() — горизонтальный (все поля на одной строке, для pdf.js)
 *
 * pdf.js возвращает текст построчно (координатная сортировка):
 *   DD.MM.YYYY HH:MM CATEGORY AMOUNT,XX BALANCE,XX
 *   DD.MM.YYYY AUTHCODE MERCHANT DESCRIPTION
 */

export interface ParsedTransaction {
  date: string        // YYYY-MM-DD
  category: string
  description: string
  amount: number      // positive = income, negative = expense
}

/** Убрать неразрывные пробелы и перевести "1 234,56" → 1234.56 */
function parseRussianNumber(s: string): number {
  // Сначала заменяем запятую на точку, потом убираем пробелы (кроме между цифр)
  // "4 100,00" -> "4 100.00" -> "4100.00" -> 4100
  const normalized = s.replace(',', '.').replace(/[\s\xa0]/g, '')
  return parseFloat(normalized) || 0
}

/** "DD.MM.YYYY" → "YYYY-MM-DD" */
function normalizeDate(s: string): string {
  const [d, m, y] = s.split('.')
  return `${y}-${m}-${d}`
}

/** Проверить что строка похожа на дату DD.MM.YYYY */
function isDate(s: string): boolean {
  return /^\d{2}\.\d{2}\.\d{4}$/.test(s.trim())
}

/** Проверить что строка похожа на время HH:MM */
function isTime(s: string): boolean {
  return /^\d{2}:\d{2}$/.test(s.trim())
}

/** Проверить что строка похожа на сумму: "186,97" или "+4 000,00" или "100,00" */
function isAmount(s: string): boolean {
  return /^[+-]?[\d\s\xa0]+,\d{2}$/.test(s.trim())
}

/** Проверить что токен является частью суммы (цифра или число с запятой) */
function isAmountPart(s: string): boolean {
  return /^[+-]?\d+$/.test(s.trim()) || isAmount(s)
}

/** Склеить разделённые пробелом части суммы */
function joinAmountParts(parts: string[], startIdx: number): { amountStr: string; nextIdx: number } {
  let amountStr = parts[startIdx]
  let j = startIdx + 1
  // Собираем части суммы: "4" + "100,00" = "4 100,00"
  while (j < parts.length && isAmountPart(parts[j])) {
    amountStr += ' ' + parts[j]
    j++
  }
  return { amountStr, nextIdx: j }
}

/** Проверить что строка похожа на остаток: "5 613,50" */
function isBalance(s: string): boolean {
  return /^[\d\s\xa0]+,\d{2}$/.test(s.trim()) && !/[+-]/.test(s)
}

/** Код авторизации (6 цифр) */
function isAuthCode(s: string): boolean {
  return /^\d{6}$/.test(s.trim())
}

/** Категории доходов */
const INCOME_CATEGORIES = new Set([
  'внесение наличных', 'перевод с карты', 'перевод на карту',
  'перевод сбп', 'перевод', 'зачисление', 'зарплата', 'проценты',
  'кэшбэк', 'кешбэк', 'возврат', 'возврат, отмена операции',
  'возврат покупки по qr–коду сбп', 'оплата по qr–коду сбп',
])

/** Sberbank категория → нормализованное название */
function normalizeCategory(cat: string): string {
  const map: Record<string, string> = {
    'супермаркеты': 'Продукты',
    'продукты': 'Продукты',
    'рестораны и кафе': 'Рестораны',
    'рестораны': 'Рестораны',
    'кафе': 'Рестораны',
    'транспорт': 'Транспорт',
    'такси': 'Транспорт',
    'топливо': 'Транспорт',
    'аптеки': 'Здоровье',
    'здоровье и красота': 'Здоровье',
    'здоровье': 'Здоровье',
    'красота': 'Здоровье',
    'жильё': 'Жильё',
    'коммунальные платежи': 'Жильё',
    'коммунальные платежи, связь, интернет.': 'Жильё',
    'коммунальные': 'Жильё',
    'развлечения': 'Развлечения',
    'отдых и развлечения': 'Развлечения',
    'кино': 'Развлечения',
    'связь': 'Связь',
    'интернет': 'Связь',
    'одежда и аксессуары': 'Одежда',
    'одежда': 'Одежда',
    'образование': 'Образование',
    'подарки': 'Подарки',
    'спорт': 'Спорт',
    'подписки': 'Подписки',
    'перевод с карты': 'Переводы',
    'перевод на карту': 'Переводы',
    'перевод сбп': 'Переводы',
    'перевод': 'Переводы',
    'внесение наличных': 'Пополнение',
    'выдача наличных': 'Наличные',
    'снятие наличных': 'Наличные',
    'банкомат': 'Наличные',
    'кешбэк': 'Кешбэк',
    'кэшбэк': 'Кешбэк',
    'зарплата': 'Зарплата',
    'проценты': 'Проценты',
    'комиссия': 'Комиссии',
    'оплата по qr–коду сбп': 'Переводы',
    'оплата по qr-коду сбп': 'Переводы',
    'возврат покупки по qr–коду сбп': 'Возвраты',
    'возврат, отмена операции': 'Возвраты',
    'возврат': 'Возвраты',
    'прочие расходы': 'Прочее',
    'прочие операции': 'Прочее',
    'прочие': 'Прочее',
    'услуги': 'Прочее',
    'услуги и прочее': 'Прочее',
    'штрафы': 'Прочее',
    'налоги': 'Прочее',
    'все для дома': 'Дом',
    'автомобиль': 'Транспорт',
    'яndex': 'Транспорт',
    'яндекс': 'Транспорт',
    'яndex go': 'Транспорт',
    'мтс': 'Связь',
    'билайн': 'Связь',
    'мегафон': 'Связь',
    'теле2': 'Связь',
  }
  return map[cat.toLowerCase().trim()] || cat
}

/** Определить тип: income или expense */
function categorize(cat: string): 'income' | 'expense' {
  return INCOME_CATEGORIES.has(cat.toLowerCase().trim()) ? 'income' : 'expense'
}

/**
 * Парсинг текста ИЗ КООРДИНАТНО-СОРТИРОВАННОГО PDF (pdf.js).
 *
 * Формат строки (pdf.js группировка):
 *   DD.MM.YYYY HH:MM CATEGORY AMOUNT,XX BALANCE,XX
 *   DD.MM.YYYY AUTH_CODE MERCHANT_DESCRIPTION...
 *
 * Первая строка — дата, время, категория, сумма, остаток.
 * Вторая строка — дата, код авторизации, описание мерчанта.
 */
export function parseSberbankInline(text: string): ParsedTransaction[] {
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean)
  const txns: ParsedTransaction[] = []

  for (let i = 0; i < lines.length; i++) {
    const parts = lines[i].split(/\s+/)

    // Ищем строку с датой и временем
    const dateIdx = parts.findIndex(p => isDate(p))
    if (dateIdx === -1) continue

    const dateStr = parts[dateIdx]
    const timeStr = dateIdx + 1 < parts.length && isTime(parts[dateIdx + 1]) ? parts[dateIdx + 1] : ''

    // Ищем сумму (число вида 186,97) после категории
    // Сумма может быть разбита пробелом: "4" "100,00" -> нужно склеить
    let amountIdx = -1
    let foundMinus = false
    for (let j = dateIdx + (timeStr ? 2 : 1); j < parts.length; j++) {
      if (parts[j] === '-') { foundMinus = true; continue }
      if (parts[j] === '+') { foundMinus = false; continue }
      if (isAmount(parts[j])) { 
        amountIdx = j
        // Проверяем предыдущий токен — может быть частью суммы (тысячи)
        if (amountIdx > 0 && isAmountPart(parts[amountIdx - 1])) {
          const joined = joinAmountParts(parts, amountIdx - 1)
          parts.splice(amountIdx - 1, 2, joined.amountStr)
          amountIdx = amountIdx - 1
        }
        break 
      }
    }
    if (amountIdx === -1) continue

    // Категория — всё между временем и суммой
    const catStart = dateIdx + (timeStr ? 2 : 1)
    let category = parts.slice(catStart, amountIdx).join(' ')

    // Убираем лишние числовые токены из категории (коды авторизации и т.д.)
    category = category.split(' ').filter(t => !isAuthCode(t) && !/^\d+$/.test(t)).join(' ')

    // Сумма и остаток
    const amountStr = (foundMinus ? '-' : '') + parts[amountIdx]
    let balanceStr = ''
    if (amountIdx + 1 < parts.length && !isDate(parts[amountIdx + 1])) {
      let balanceIdx = amountIdx + 1
      // Баланс тоже может быть разделен пробелом: "5 613,50"
      if (balanceIdx + 1 < parts.length && isAmountPart(parts[balanceIdx + 1])) {
        const joined = joinAmountParts(parts, balanceIdx)
        balanceStr = joined.amountStr
      } else {
        balanceStr = parts[balanceIdx]
      }
    }

    // Описание — следующая строка, если она начинается с даты и имеет код
    let description = ''
    if (i + 1 < lines.length) {
      const nextParts = lines[i + 1].split(/\s+/)
      const nextDateIdx = nextParts.findIndex(p => isDate(p))
      if (nextDateIdx !== -1 && nextDateIdx + 1 < nextParts.length && isAuthCode(nextParts[nextDateIdx + 1])) {
        description = nextParts.slice(nextDateIdx + 2).join(' ').trim()
        i++ // пропускаем строку описания
      }
    }

    if (!category) continue

    const amount = parseRussianNumber(amountStr.replace(/^\+/, ''))
    const isNegative = amountStr.startsWith('-') || (!amountStr.startsWith('+') && categorize(category) === 'expense')

    txns.push({
      date: normalizeDate(dateStr),
      category: normalizeCategory(category),
      description: description || category,
      amount: isNegative ? -amount : amount,
    })
  }

  return txns
}

/**
 * Парсинг текста, извлечённого ИЗ КОПИРОВАННОГО PDF (вертикальный формат).
 * Каждое поле на отдельной строке.
 */
export function parseSberbankText(text: string): ParsedTransaction[] {
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean)
  const txns: ParsedTransaction[] = []

  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    if (isDate(line)) {
      const dateStr = line
      i++

      let time = ''
      if (i < lines.length && isTime(lines[i])) {
        time = lines[i]
        i++
      }

      let category = ''
      while (i < lines.length) {
        const next = lines[i]
        if (isAmount(next) || isDate(next)) break
        if (isBalance(next) && !category) { i++; break }
        if (!isAuthCode(next) || !category) {
          category = next
          i++
          break
        }
        i++
      }

      if (!category || i >= lines.length) continue

      while (i < lines.length && !isAmount(lines[i]) && !isDate(lines[i])) {
        i++
      }
      if (i >= lines.length || isDate(lines[i])) continue

      const amountStr = lines[i]
      i++

      while (i < lines.length && !isBalance(lines[i]) && !isDate(lines[i]) && !isAmount(lines[i])) {
        i++
      }
      if (i >= lines.length || isDate(lines[i]) || isAmount(lines[i])) continue

      i++

      let descLines: string[] = []
      while (i < lines.length && !isDate(lines[i])) {
        const l = lines[i].trim()
        if (l && !isAuthCode(l) && l !== 'Продолжение на следующей странице' && !l.startsWith('Страница')) {
          descLines.push(l)
        }
        i++
      }

      const amount = parseRussianNumber(amountStr.replace(/^\+/, ''))
      const isNegative = amountStr.startsWith('-') || (!amountStr.startsWith('+') && categorize(category) === 'expense')
      const desc = descLines.join(' ').replace(/\s+/g, ' ').trim()

      txns.push({
        date: normalizeDate(dateStr),
        category: normalizeCategory(category),
        description: desc || `${category}`,
        amount: isNegative ? -amount : amount,
      })
    } else {
      i++
    }
  }

  return txns
}

/**
 * Парсинг CSV выписки Сбербанка (; разделитель).
 */
export function parseSberbankCSV(text: string): ParsedTransaction[] {
  const lines = text.split('\n').filter(l => l.trim())
  if (lines.length < 2) return []

  // Пропускаем заголовок
  const dataStart = lines[0].includes('Дата') ? 1 : 0
  const txns: ParsedTransaction[] = []

  for (let i = dataStart; i < lines.length; i++) {
    const cols = lines[i].split(';')
    if (cols.length < 4) continue

    const [dateStr, , description, rawAmount] = cols
    if (!isDate(dateStr.trim())) continue

    const amount = parseRussianNumber(rawAmount.replace(/["\s]/g, ''))
    const isNegative = amount < 0 || rawAmount.trim().startsWith('-')
    const cat = description.trim()

    txns.push({
      date: normalizeDate(dateStr.trim()),
      category: normalizeCategory(cat),
      description: cat,
      amount: isNegative ? amount : Math.abs(amount),
    })
  }

  return txns
}
