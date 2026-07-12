/**
 * React Hook Form обёртка для FinHelper.
 */
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import type { UseFormReturn, FieldValues, Path, UseFormProps } from 'react-hook-form'

export type { UseFormReturn, FieldValues, Path }

/**
 * Создаёт форму с Zod-валидацией.
 */
export function useFinForm<T extends FieldValues>(
  schema: z.ZodType<T>,
  options?: UseFormProps<T>,
) {
  return useForm<T>({
    ...options,
    resolver: zodResolver(schema as any) as any,
  })
}

export interface FormOption {
  value: string | number
  label: string
}

/** Схема для денежного поля (строка) */
export const moneyField = (label = 'Сумма') =>
  z.string().min(1, `Введите ${label.toLowerCase()}`).regex(/^\d+([.,]\d{1,2})?$/, 'Неверный формат')

/** Схема для обязательного текстового поля */
export const requiredField = (label = 'Поле') =>
  z.string().min(1, `${label} обязательно`)

/** Схема для даты (YYYY-MM-DD) */
export const dateField = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, 'Неверный формат даты (ГГГГ-ММ-ДД)')
