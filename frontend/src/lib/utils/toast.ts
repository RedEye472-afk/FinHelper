/**
 * Toast-уведомления на sonner.
 * Использовать вместо кастомного Toast.tsx.
 *
 * @example
 *   toast.success('Операция создана')
 *   toast.error('Ошибка', { description: 'Проверьте данные' })
 */
import { toast as sonnerToast } from 'sonner'

export const toast = {
  success: (message: string, opts?: { description?: string }) =>
    sonnerToast.success(message, { description: opts?.description }),

  error: (message: string, opts?: { description?: string }) =>
    sonnerToast.error(message, { description: opts?.description }),

  info: (message: string, opts?: { description?: string }) =>
    sonnerToast.info(message, { description: opts?.description }),

  warning: (message: string, opts?: { description?: string }) =>
    sonnerToast.warning(message, { description: opts?.description }),

  /** Для массового импорта — прогресс с возможностью отмены */
  loading: (message: string) =>
    sonnerToast.loading(message),

  /** Закрыть все тосты */
  dismiss: () => sonnerToast.dismiss(),
}
