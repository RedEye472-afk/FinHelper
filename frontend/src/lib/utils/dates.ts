/**
 * date-fns обёртка для FinHelper.
 * Единый формат дат в проекте.
 */
import { format, parse, parseISO, formatDistanceToNow, isToday, isYesterday, isSameMonth, isSameYear, startOfMonth, endOfMonth, startOfYear, endOfYear, subMonths, startOfDay, differenceInDays, addDays } from 'date-fns'
import { ru } from 'date-fns/locale'

export { format, parse, parseISO, formatDistanceToNow, isToday, isYesterday, isSameMonth, isSameYear, startOfMonth, endOfMonth, startOfYear, endOfYear, subMonths, startOfDay, differenceInDays, addDays }

export const LOCALE_RU = ru

/** "2026-07-13" → "13 июля 2026" */
export function formatDateFull(dateStr: string): string {
  return format(parseISO(dateStr), 'd MMMM yyyy', { locale: ru })
}

/** "2026-07-13" → "13.07.2026" */
export function formatDateShort(dateStr: string): string {
  return format(parseISO(dateStr), 'dd.MM.yyyy')
}

/** "2026-07-13" → "13 июл" */
export function formatDateCompact(dateStr: string): string {
  return format(parseISO(dateStr), 'd MMM', { locale: ru })
}

/** "2026-07-13" → "сегодня" / "вчера" / "13 июл" */
export function formatDateRelative(dateStr: string): string {
  const d = parseISO(dateStr)
  if (isToday(d)) return 'сегодня'
  if (isYesterday(d)) return 'вчера'
  return format(d, 'd MMM', { locale: ru })
}

/** "2026-07-13T14:30" → "14:30" */
export function formatTime(isoStr: string): string {
  return format(parseISO(isoStr), 'HH:mm')
}

/** "2026-07-13" → Date */
export function parseDate(dateStr: string): Date {
  return parseISO(dateStr)
}

/** Date → "2026-07-13" */
export function toDateStr(d: Date): string {
  return format(d, 'yyyy-MM-dd')
}
