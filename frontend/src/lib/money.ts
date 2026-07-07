/**
 * money.ts — decimal.js обёртка для FinHelper
 * Правило: деньги = строки, числа только для UI-инпутов.
 */
import Decimal from 'decimal.js'

// Decimal.js настройки: 28 знаков, ROUND_HALF_UP (→ 1.005 → 1.01, как бэкенд ROUND_HALF_AWAY_FROM_ZERO)
Decimal.set({ precision: 28, rounding: Decimal.ROUND_HALF_UP })

/** Создать Decimal из строки (вход с бэка). null/undefined → Decimal(0) */
export function toDecimal(s: string | null | undefined): Decimal {
  if (!s || s === '') return new Decimal(0)
  // Бэкенд шлёт "100.50" (точка), не запятую
  return new Decimal(s)
}

/** Безопасный parse: невалидная строка → Decimal(0) */
export function safeParse(s: string): Decimal {
  try { return new Decimal(s) } catch { return new Decimal(0) }
}

/** Форматирование для отображения: "1 234,56 ₽" */
export function formatMoney(d: Decimal | string | number): string {
  const dec = d instanceof Decimal ? d : new Decimal(d)
  const rounded = dec.toDecimalPlaces(2)
  const [intPart, decPart] = rounded.toFixed(2).split('.')
  // Разбиваем целую часть пробелами
  const spaced = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
  return `${spaced},${decPart} ₽`
}

/** Форматирование без символа ₽ */
export function formatNumber(d: Decimal | string | number): string {
  const dec = d instanceof Decimal ? d : new Decimal(d)
  const rounded = dec.toDecimalPlaces(2)
  const [intPart, decPart] = rounded.toFixed(2).split('.')
  const spaced = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
  return `${spaced},${decPart}`
}

/** Короткий формат (тысячи, млн) — всегда 1 знак после запятой через запятую */
export function formatCompact(d: Decimal | string | number): string {
  const dec = d instanceof Decimal ? d : new Decimal(d)
  const abs = dec.abs()
  if (abs.gte(1_000_000_000)) return `${dec.div(1_000_000_000).toFixed(1).replace('.', ',')} млрд ₽`
  if (abs.gte(1_000_000)) return `${dec.div(1_000_000).toFixed(1).replace('.', ',')} млн ₽`
  if (abs.gte(1_000)) return `${dec.div(1_000).toFixed(1).replace('.', ',')} тыс ₽`
  return formatMoney(dec)
}

/** Конвертация Decimal → строка для отправки на бэк */
export function moneyToString(d: Decimal): string {
  return d.toDecimalPlaces(2).toFixed(2)
}

/** Сумма массива строк (с бэка) */
export function sumMoney(values: (string | Decimal)[]): Decimal {
  return values.reduce((acc: Decimal, v) => acc.plus(toDecimal(String(v))), new Decimal(0))
}

/** Арифметика */
export const M = {
  add: (a: Decimal, b: Decimal): Decimal => a.plus(b),
  sub: (a: Decimal, b: Decimal): Decimal => a.minus(b),
  mul: (a: Decimal, b: Decimal): Decimal => a.mul(b),
  div: (a: Decimal, b: Decimal): Decimal => a.div(b),
  abs: (a: Decimal): Decimal => a.abs(),
  neg: (a: Decimal): Decimal => a.negated(),
  min: (a: Decimal, b: Decimal): Decimal => Decimal.min(a, b),
  max: (a: Decimal, b: Decimal): Decimal => Decimal.max(a, b),
  zero: () => new Decimal(0),
  isPositive: (a: Decimal): boolean => a.gt(0),
  isNegative: (a: Decimal): boolean => a.lt(0),
  isZero: (a: Decimal): boolean => a.isZero(),
  /** Число в строку (для input value) */
  toInput: (d: Decimal): string => d.toFixed(2),
  /** Строка из input → Decimal */
  fromInput: (s: string): Decimal => safeParse(s),
}

export { Decimal }
