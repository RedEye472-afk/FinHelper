/**
 * Извлечение текста из PDF с помощью pdf.js
 * 
 * Проблема: getTextContent() возвращает элементы в PDF-структурном порядке,
 * а не в порядке чтения. Решение: группируем элементы по строкам (Y-координата)
 * и сортируем строки сверху вниз.
 */
import * as pdfjsLib from 'pdfjs-dist'

if (typeof window !== 'undefined') {
  pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
    'pdfjs-dist/build/pdf.worker.min.mjs',
    import.meta.url,
  ).href
}

interface TextItem {
  str: string
  x: number
  y: number
}

/**
 * Группирует элементы по строкам на основе Y-координаты.
 * Допуск: элементы с разницей по Y < 10px считаются одной строкой.
 */
function groupIntoLines(items: TextItem[]): TextItem[][] {
  if (items.length === 0) return []

  const sorted = [...items].sort((a, b) => b.y - a.y) // сверху вниз

  const lines: TextItem[][] = [[sorted[0]]]
  for (let i = 1; i < sorted.length; i++) {
    const prevY = sorted[i - 1].y
    const currY = sorted[i].y
    if (Math.abs(prevY - currY) < 10) {
      lines[lines.length - 1].push(sorted[i]) // та же строка
    } else {
      lines.push([sorted[i]]) // новая строка
    }
  }

  // Сортируем элементы в каждой строке слева направо
  for (const line of lines) {
    line.sort((a, b) => a.x - b.x)
  }

  return lines
}

/**
 * Извлечь текст из PDF с правильным порядком чтения.
 */
export async function extractTextFromPDF(data: ArrayBuffer): Promise<string> {
  const loadingTask = pdfjsLib.getDocument({ data })
  const pdf = await loadingTask.promise
  const pages: string[] = []

  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i)
    const content = await page.getTextContent()

    const items: TextItem[] = []
    for (const raw of content.items) {
      const item = raw as Record<string, unknown>
      if (typeof item.str === 'string' && item.str.trim()) {
        const t = (item.transform as number[]) || []
        items.push({
          str: item.str,
          x: t[4] || 0,
          y: t[5] || 0,
        })
      }
    }

    const lines = groupIntoLines(items)
    pages.push(lines.map(line => line.map(w => w.str).join(' ')).join('\n'))
  }

  const result = pages.join('\n\n')
  return result
}

/**
 * Для дебага — возвращает сырые элементы с координатами
 */
export async function debugExtractPDF(data: ArrayBuffer): Promise<TextItem[]> {
  const loadingTask = pdfjsLib.getDocument({ data })
  const pdf = await loadingTask.promise
  const all: TextItem[] = []

  for (let i = 1; i <= Math.min(pdf.numPages, 3); i++) {
    const page = await pdf.getPage(i)
    const content = await page.getTextContent()
    for (const raw of content.items) {
      const item = raw as Record<string, unknown>
      if (typeof item.str === 'string' && item.str.trim()) {
        const t = (item.transform as number[]) || []
        all.push({
          str: item.str,
          x: t[4] || 0,
          y: t[5] || 0,
        })
      }
    }
  }

  all.sort((a, b) => b.y - a.y)
  return all
}
