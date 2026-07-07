import { useEffect } from 'react'

type KeyHandler = Record<string, (e: KeyboardEvent) => void>

export function useKeyboard(handlers: KeyHandler, enabled = true) {
  useEffect(() => {
    if (!enabled) return
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') return
      const key = e.key.toLowerCase()
      if (handlers[key]) { e.preventDefault(); handlers[key](e) }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [handlers, enabled])
}
