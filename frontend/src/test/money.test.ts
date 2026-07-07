import { describe, it, expect } from 'vitest'
import { toDecimal, safeParse, formatMoney, formatCompact, moneyToString, sumMoney, M } from '../lib/money'

describe('money.ts — decimal.js обёртка', () => {
  describe('toDecimal', () => {
    it('конвертирует строку в Decimal', () => {
      expect(toDecimal('123.45').toString()).toBe('123.45')
    })
    it('null/undefined → 0', () => {
      expect(toDecimal(null).isZero()).toBe(true)
      expect(toDecimal(undefined).isZero()).toBe(true)
    })
    it('пустая строка → 0', () => {
      expect(toDecimal('').isZero()).toBe(true)
    })
  })

  describe('safeParse', () => {
    it('невалидная строка → 0', () => {
      expect(safeParse('abc').isZero()).toBe(true)
    })
    it('валидная строка → Decimal', () => {
      expect(safeParse('99.99').toString()).toBe('99.99')
    })
  })

  describe('0.1 + 0.2 = 0.30 (не 0.30000000004)', () => {
    it('decimal.js сохраняет точность', () => {
      const sum = toDecimal('0.1').plus(toDecimal('0.2'))
      expect(sum.toString()).toBe('0.3')
    })
  })

  describe('ROUND_HALF_AWAY_FROM_ZERO', () => {
    it('1.005 → 1.01, не 1.00 (bankers rounding)', () => {
      const d = toDecimal('1.005')
      expect(d.toFixed(2)).toBe('1.01')
    })
  })

  describe('formatMoney', () => {
    it('форматирует с пробелами и ₽', () => {
      expect(formatMoney('1234567.89')).toBe('1 234 567,89 ₽')
    })
    it('форматирует ноль', () => {
      expect(formatMoney('0')).toBe('0,00 ₽')
    })
  })

  describe('formatCompact', () => {
    it('миллиарды', () => {
      expect(formatCompact('1500000000')).toBe('1,5 млрд ₽')
    })
    it('миллионы', () => {
      expect(formatCompact('2500000')).toBe('2,5 млн ₽')
    })
    it('тысячи', () => {
      expect(formatCompact('15000')).toBe('15,0 тыс ₽')
    })
  })

  describe('moneyToString', () => {
    it('возвращает строку с 2 знаками', () => {
      expect(moneyToString(toDecimal('100'))).toBe('100.00')
    })
  })

  describe('sumMoney', () => {
    it('суммирует массив строк', () => {
      expect(sumMoney(['100.50', '200.25', '0.25']).toString()).toBe('301')
    })
  })

  describe('M — арифметика', () => {
    it('add', () => { expect(M.add(toDecimal('1'), toDecimal('2')).toString()).toBe('3') })
    it('sub', () => { expect(M.sub(toDecimal('5'), toDecimal('3')).toString()).toBe('2') })
    it('mul', () => { expect(M.mul(toDecimal('2'), toDecimal('3')).toString()).toBe('6') })
    it('div', () => { expect(M.div(toDecimal('6'), toDecimal('2')).toString()).toBe('3') })
    it('zero', () => { expect(M.zero().isZero()).toBe(true) })
  })
})