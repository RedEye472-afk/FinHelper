/**
 * Sberbank statement parser — для PDF выписок Сбербанка
 * Формат: каждая строка = отдельное поле, данные идут блоками
 *
 * Структура транзакции:
 *   DD.MM.YYYY        ← дата
 *   HH:MM             ← время (опционально, иногда пропускается)
 *   Категория         ← например "Супермаркеты"
 *   Сумма             ← например "186,97" или "+4 000,00"
 *   Остаток           ← например "5 613,50"
 *   [код авторизации] ← 6 цифр, опционально
 *   [описание...]     ← merchant, может быть на 1-2 строки
 */

export interface ParsedTransaction {
  date: string        // YYYY-MM-DD
  category: string
  description: string
  amount: number      // positive = income, negative = expense
}

/** Убрать неразрывные пробелы и перевести "1 234,56" → 1234.56 */
function parseRussianNumber(s: string): number {
  const cleaned = s.replace(/[\s\xa0]/g, '').replace(',', '.')
  return parseFloat(cleaned) || 0
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

/** Проверить что строка похожа на сумму: "186,97" или "+4 000,00" */
function isAmount(s: string): boolean {
  return /^[+-]?[\d\s\xa0]+,\d{2}$/.test(s.trim())
}

/** Проверить что строка похожа на остаток: "5 613,50" */
function isBalance(s: string): boolean {
  return /^[\d\s\xa0]+,\d{2}$/.test(s.trim()) && !s.includes('+') && !s.includes('-')
}

/** Код авторизации (6 цифр) */
function isAuthCode(s: string): boolean {
  return /^\d{6}$/.test(s.trim())
}

/** Sberbank категория → FinHelper тип */
function categorize(cat: string): 'income' | 'expense' {
  const incomeKeywords = ['внесение наличных', 'перевод', 'зачисление', 'возврат', 'кэшбэк', 'проценты', 'зарплата']
  return incomeKeywords.some(k => cat.toLowerCase().includes(k)) ? 'income' : 'expense'
}

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
    'коммунальные': 'Жильё',
    'развлечения': 'Развлечения',
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
    'внесение наличных': 'Переводы',
    'кешбэк': 'Переводы',
    'зарплата': 'Зарплата',
    'проценты': 'Переводы',
    'комиссия': 'Переводы',
    'прочие операции': 'Прочее',
    'прочие': 'Прочее',
    'услуги': 'Прочее',
    'услуги и прочее': 'Прочее',
    'штрафы': 'Прочее',
    'налоги': 'Прочее',
    'снятие наличных': 'Наличные',
    'банкомат': 'Наличные',
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

/**
 * Парсинг текста, извлечённого из PDF выписки Сбербанка.
 * Поддерживает многостраничные выписки с вертикальным форматом.
 */
export function parseSberbankText(text: string): ParsedTransaction[] {
  const lines = text.split('\n').map(l => l.trim()).filter(Boolean)
  const txns: ParsedTransaction[] = []

  let i = 0
  while (i < lines.length) {
    const line = lines[i]

    // Ищем строку, похожую на дату
    if (isDate(line)) {
      const dateStr = line
      i++

      // Следующая строка может быть временем или сразу категорией
      let category = ''
      let time = ''

      if (i < lines.length && isTime(lines[i])) {
        time = lines[i]
        i++
      }

      // Дальше должна быть категория
      if (i < lines.length && !isAmount(lines[i]) && !isBalance(lines[i]) && !isDate(lines[i]) && !isAuthCode(lines[i])) {
        category = lines[i]
        i++
      }

      // Сумма
      if (i < lines.length && isAmount(lines[i])) {
        const amountStr = lines[i]
        i++

        // Остаток
        if (i < lines.length && isBalance(lines[i])) {
          // const balance = lines[i] // не используем
          i++

          // Собираем описание: код авторизации + merchant (до следующей даты)
          let descLines: string[] = []
          while (i < lines.length && !isDate(lines[i])) {
            const l = lines[i].trim()
            if (l && !isAuthCode(l) && l !== 'Продолжение на следующей странице' && !l.startsWith('Страница')) {
              descLines.push(l)
            }
            i++
          }

          if (category) {
            const amount = parseRussianNumber(amountStr.replace(/^\+/, ''))
            const isNegative = amountStr.startsWith('-') || (!amountStr.startsWith('+') && categorize(category) === 'expense')
            const desc = descLines.join(' ').replace(/\s+/g, ' ').trim()

            txns.push({
              date: normalizeDate(dateStr),
              category: normalizeCategory(category),
              description: desc || `${category}${time ? ' ' + time : ''}`,
              amount: isNegative ? -amount : amount,
            })
          }
        }
      }
    } else {
      i++
    }
  }

  return txns
}

/**
 * Парсинг CSV от Сбербанка (разделитель ";")
 * Формат: Дата;Категория;Описание;Сумма;Валюта;Остаток
 */
export function parseSberbankCSV(csv: string): ParsedTransaction[] {
  const lines = csv.split('\n').map(l => l.trim()).filter(Boolean)
  const txns: ParsedTransaction[] = []

  for (let i = 1; i < lines.length; i++) {
    const cols = lines[i].split(';')
    if (cols.length < 4) continue

    const [dateRaw, category, description, amountRaw] = cols
    if (!dateRaw) continue

    const amount = parseRussianNumber((amountRaw || '0').replace(/^\+/, ''))
    const isNegative = (amountRaw || '').startsWith('-')

    txns.push({
      date: normalizeDate(dateRaw.trim()),
      category: normalizeCategory((category || 'Прочее').trim()),
      description: (description || '').trim(),
      amount: isNegative ? -amount : amount,
    })
  }

  return txns
}
