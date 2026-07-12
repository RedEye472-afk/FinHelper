import { useSettings } from '../hooks/useSettings'
import { useAuth } from '../context/AuthContext'
import { Eye, EyeOff, LogOut, Keyboard, Moon, Sun, Palette } from 'lucide-react'
import { DataExportImport } from '../components/shared/DataExportImport'
import type { ThemeMode, Currency } from '../types'

const themes: { id: ThemeMode; label: string; color: string }[] = [
  { id: 'emerald', label: 'Изумрудная', color: '#10b981' },
  { id: 'blue', label: 'Океан', color: '#3b82f6' },
  { id: 'purple', label: 'Фиолетовая', color: '#8b5cf6' },
  { id: 'dark', label: 'Ночь', color: '#6E56CF' },
]

const currencies: { id: Currency; label: string; symbol: string }[] = [
  { id: 'RUB', label: 'Российский рубль', symbol: '₽' },
  { id: 'USD', label: 'Доллар США', symbol: '$' },
  { id: 'EUR', label: 'Евро', symbol: '€' },
]

export function SettingsPage() {
  const { theme, currency, hideBalance, setSettings, initials, name, email } = useSettings()
  const { user, logout } = useAuth()

  return (
    <div className="space-y-4">

      {/* User card */}
      <div className="rounded-[20px] border p-5 text-center relative overflow-hidden"
        style={{
          background: 'linear-gradient(135deg, #141A2D 0%, #1E293B 100%)',
          borderColor: 'rgba(255,255,255,0.06)',
        }}
      >
        <div className="w-16 h-16 rounded-full mx-auto mb-3 flex items-center justify-center text-white text-xl font-bold"
          style={{
            background: 'linear-gradient(135deg, #6E56CF 0%, #A78BFA 100%)',
            boxShadow: '0 0 24px rgba(110,86,207,0.3)',
          }}>
          {initials}
        </div>
        <p className="text-base font-semibold" style={{ color: 'var(--text-primary)' }}>{name}</p>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>{user?.email || email}</p>
      </div>

      {/* Theme */}
      <div className="rounded-[20px] border p-4" style={{ background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Palette size={16} style={{ color: 'var(--text-tertiary)' }} />
          <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Тема оформления</p>
        </div>
        <div className="flex gap-3">
          {themes.map(t => (
            <button key={t.id} onClick={() => setSettings({ theme: t.id })}
              className="flex-1 flex flex-col items-center gap-2 py-3 rounded-xl transition-all btn-press"
              style={{
                background: theme === t.id ? `${t.color}18` : 'transparent',
                border: theme === t.id ? `2px solid ${t.color}` : '2px solid transparent',
              }}>
              <div className="w-8 h-8 rounded-full" style={{ background: t.color }} />
              <span className="text-[11px] font-medium" style={{ color: theme === t.id ? t.color : 'var(--text-secondary)' }}>
                {t.label}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Currency */}
      <div className="rounded-[20px] border p-4" style={{ background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}>
        <p className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Валюта</p>
        <div className="flex gap-2">
          {currencies.map(c => (
            <button key={c.id} onClick={() => setSettings({ currency: c.id })}
              className="flex-1 py-3 rounded-xl text-sm font-medium transition-all btn-press"
              style={{
                background: currency === c.id ? 'rgba(110,86,207,0.12)' : 'transparent',
                border: currency === c.id ? '1px solid rgba(110,86,207,0.3)' : '1px solid rgba(255,255,255,0.06)',
                color: currency === c.id ? 'var(--color-primary-400)' : 'var(--text-secondary)',
              }}>
              <span className="text-lg">{c.symbol}</span>
              <p className="text-[10px] mt-0.5">{c.label.split(' ').pop()}</p>
            </button>
          ))}
        </div>
      </div>

      {/* Balance visibility */}
      <div className="rounded-[20px] border p-4" style={{ background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}>
        <p className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Отображение баланса</p>
        <button onClick={() => setSettings({ hideBalance: !hideBalance })}
          className="w-full flex items-center justify-between py-2 rounded-xl transition-all">
          <div className="flex items-center gap-3">
            {hideBalance
              ? <EyeOff size={20} style={{ color: 'var(--text-tertiary)' }} />
              : <Eye size={20} style={{ color: 'var(--color-primary-400)' }} />}
            <span className="text-sm" style={{ color: 'var(--text-primary)' }}>
              {hideBalance ? 'Скрыт' : 'Виден'}
            </span>
          </div>
          <div className="relative rounded-full transition-colors" style={{
            width: 44, height: 24,
            background: hideBalance ? 'rgba(255,255,255,0.1)' : 'var(--color-primary-600)',
          }}>
            <div className="bg-white rounded-full absolute top-0.5 transition-all duration-200" style={{
              width: 20, height: 20,
              transform: hideBalance ? 'translateX(2px)' : 'translateX(22px)',
              boxShadow: '0 1px 4px rgba(0,0,0,0.3)',
            }} />
          </div>
        </button>
      </div>

      {/* Shortcuts */}
      <div className="rounded-[20px] border p-4" style={{ background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Keyboard size={16} style={{ color: 'var(--text-tertiary)' }} />
          <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Шорткаты</p>
        </div>
        <div className="grid grid-cols-2 gap-1.5">
          {[
            { key: 'g', label: 'Цели' }, { key: 'o', label: 'Операции' },
            { key: 'b', label: 'Бюджеты' }, { key: 'd', label: 'Дашборд' },
            { key: 'n', label: 'Новая операция' }, { key: 's', label: 'Настройки' },
          ].map(s => (
            <div key={s.key} className="flex items-center gap-2 py-1.5">
              <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold"
                style={{ background: 'rgba(255,255,255,0.06)', color: 'var(--text-secondary)' }}>
                {s.key}
              </span>
              <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{s.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Data */}
      <DataExportImport />

      {/* Account */}
      <div className="rounded-[20px] border p-4" style={{ background: 'var(--bg-surface)', borderColor: 'var(--border-default)' }}>
        <p className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Аккаунт</p>
        <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>{user?.email || email}</p>
        <button onClick={logout}
          className="flex items-center gap-2 w-full mt-3 py-2.5 rounded-xl text-sm font-medium transition-all btn-press"
          style={{ color: '#F43F5E', background: 'rgba(244,63,94,0.08)' }}
        >
          <LogOut size={16} /> Выйти
        </button>
      </div>

      <p className="text-xs text-center py-4" style={{ color: 'var(--text-tertiary)' }}>
        FinHelper v1.0.0 • Приватный финансовый трекер
      </p>
    </div>
  )
}
