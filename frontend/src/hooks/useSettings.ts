import { useState, useCallback, useEffect } from 'react'
import type { Settings, ThemeMode, Currency } from '../types'

const STORAGE_KEY = 'finhelper-settings'

const defaults: Settings = {
  theme: 'dark' as ThemeMode,
  currency: 'RUB' as Currency,
  hideBalance: false,
  name: 'Алексей',
  initials: 'АИ',
  email: 'alexey@mail.ru',
}

const currencySymbols: Record<Currency, string> = { RUB: '₽', USD: '$', EUR: '€' }

function loadSettings(): Settings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) { const s = JSON.parse(raw); return { ...defaults, ...s } }
  } catch {}
  return defaults
}

export function useSettings() {
  const [settings, setSettingsState] = useState<Settings>(loadSettings)

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
    document.documentElement.setAttribute('data-theme', settings.theme)
  }, [settings])

  const setSettings = useCallback((patch: Partial<Settings>) => {
    setSettingsState(prev => ({ ...prev, ...patch }))
  }, [])

  return {
    ...settings,
    setSettings,
    symbol: currencySymbols[settings.currency] || '₽',
  }
}
