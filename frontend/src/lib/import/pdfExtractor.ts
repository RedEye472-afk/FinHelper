/**
 * Извлечение текста из PDF с помощью pdf.js
 */
import * as pdfjsLib from 'pdfjs-dist'

// Устанавливаем worker
pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString()

/**
 * Извлечь текст из PDF-файла (ArrayBuffer)
 * Возвращает сырой текст, готовый для parseSberbankText()
 */
export async function extractTextFromPDF(data: ArrayBuffer): Promise<string> {
  const pdf = await pdfjsLib.getDocument({ data }).promise
  const pages: string[] = []

  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i)
    const content = await page.getTextContent()
    const text = content.items.map(item => {
      if ('str' in item) return item.str
      return ''
    }).join('\n')
    pages.push(text)
  }

  return pages.join('\n')
}

/**
 * Проверить, похож ли PDF на выписку Сбербанка
 */
export function looksLikeSberbankPDF(text: string): boolean {
  return text.includes('СберБанк') || text.includes('Sberbank') || text.includes('Выписка по платёжному счёту')
}
