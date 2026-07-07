import { useSettings } from '../hooks/useSettings'
import { useAuth } from '../context/AuthContext'
import { Eye, EyeOff, LogOut, Keyboard, Check } from 'lucide-react'
import { DataExportImport } from '../components/shared/DataExportImport'
import type { ThemeMode, Currency } from '../types'

const themes: { id: ThemeMode; label: string; gradient: string; colors: string[] }[] = [
  { id: 'emerald', label: 'Изумрудная', gradient: 'linear-gradient(135deg, #059669, #10b981, #34d399)', colors: ['#059669', '#10b981', '#34d399'] },
  { id: 'blue', label: 'Океан', gradient: 'linear-gradient(135deg, #1e3a8a, #2563eb, #60a5fa)', colors: ['#1e3a8a', '#2563eb', '#60a5fa'] },
  { id: 'purple', label: 'Фиолетовая', gradient: 'linear-gradient(135deg, #4c1d95, #7c3aed, #a78bfa)', colors: ['#4c1d95', '#7c3aed', '#a78bfa'] },
  { id: 'dark', label: 'Ночь', gradient: 'linear-gradient(135deg, #0f172a, #1e293b, #334155)', colors: ['#0f172a', '#1e293b', '#334155'] },
]

const currencies: { id: Currency; label: string }[] = [
  { id: 'RUB', label: 'Российский рубль (₽)' },
  { id: 'USD', label: 'Доллар США ($)' },
  { id: 'EUR', label: 'Евро (€)' },
]

export function SettingsPage() {
  const { theme, currency, hideBalance, setSettings } = useSettings()
  const { user, logout } = useAuth()

  return (
    <div className="space-y-6">
      {/* Theme */}
      <div>
        <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Тема оформления</h3>
        <div className="grid grid-cols-2 gap-3">
          {themes.map(t => (
            <button key={t.id} onClick={() => setSettings({ theme: t.id })}
              className={`p-4 rounded-2xl border-2 transition-all text-left ${theme === t.id ? 'shadow-md' : 'hover:shadow-sm'}`}
              style={theme === t.id
                ? { borderColor: 'var(--color-primary-500)', background: 'var(--bg-surface)' }
                : { borderColor: 'var(--border-default)', background: 'var(--bg-surface)' }
              }>
              <div className="h-10 rounded-lg mb-3" style={{ background: t.gradient }} />
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t.label}</p>
                {theme === t.id && <Check size={16} style={{ color: 'var(--color-primary-600)' }} />}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Currency */}
      <div>
        <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Валюта</h3>
        <div className="space-y-2">
          {currencies.map(c => (
            <button key={c.id} onClick={() => setSettings({ currency: c.id })}
              className="w-full flex items-center justify-between p-3 rounded-xl border transition-all"
              style={currency === c.id
                ? { borderColor: 'var(--color-primary-500)', background: 'rgba(16,185,129,0.08)' }
                : { borderColor: 'var(--border-default)', background: 'var(--bg-surface)' }
              }>
              <span className="text-sm" style={{ color: 'var(--text-primary)' }}>{c.label}</span>
              {currency === c.id && <div className="w-5 h-5 rounded-full flex items-center justify-center text-white text-xs" style={{ background: 'var(--color-primary-500)' }}>✓</div>}
            </button>
          ))}
        </div>
      </div>

      {/* Balance visibility */}
      <div>
        <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Отображение баланса</h3>
        <button onClick={() => setSettings({ hideBalance: !hideBalance })}
          className="w-full flex items-center justify-between p-3 rounded-xl border transition-all"
          style={{ borderColor: 'var(--border-default)', background: 'var(--bg-surface)' }}>
          <div className="flex items-center gap-3">
            {hideBalance
              ? <EyeOff size={20} style={{ color: 'var(--text-tertiary)' }} />
              : <Eye size={20} style={{ color: 'var(--color-primary-500)' }} />}
            <span className="text-sm" style={{ color: 'var(--text-primary)' }}>{hideBalance ? 'Скрыт' : 'Виден'}</span>
          </div>
          <div className="relative" style={{ width: 40, height: 20 }}>
            <div className="rounded-full transition-colors absolute inset-0"
              style={{ background: hideBalance ? 'var(--border-default)' : 'var(--color-primary-500)' }} />
            <div className="bg-white rounded-full absolute top-0.5 transition-transform"
              style={{ width: 16, height: 16, transform: hideBalance ? 'translateX(2px)' : 'translateX(22px)' }} />
          </div>
        </button>
      </div>

      {/* Data Export/Import */}
      <DataExportImport />

      {/* Keyboard shortcuts */}
      <div>
        <h3 className="text-sm font-semibold mb-3 flex items-center gap-1.5" style={{ color: 'var(--text-primary)' }}><Keyboard size={16} /> Шорткаты</h3>
        <div className="p-3 rounded-xl text-xs space-y-1.5" style={{ background: 'var(--bg-surface)' }}>
          {['g — Цели', 'o — Операции', 'b — Бюджеты', 'd — Дашборд', 'n — Новая операция', 's — Настройки'].map(s => (
            <div key={s} className="flex items-center gap-2">
              <span className="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold"
                style={{ background: 'var(--border-default)', color: 'var(--text-secondary)' }}>
                {s.split(' — ')[0]}
              </span>
              <span style={{ color: 'var(--text-secondary)' }}>{s.split(' — ')[1]}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Account */}
      <div>
        <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Аккаунт</h3>
        <div className="p-3 rounded-xl" style={{ background: 'var(--bg-surface)' }}>
          <p className="text-sm" style={{ color: 'var(--text-primary)' }}>{user?.email || 'Пользователь'}</p>
          <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Зарегистрирован: {user?.created_at?.slice(0, 10) || '—'}</p>
        </div>
        <button onClick={logout} className="flex items-center gap-2 w-full mt-2 p-3 rounded-xl transition-colors text-sm font-medium"
          style={{ color: 'var(--color-danger-500)', background: 'transparent' }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'var(--color-danger-50)' }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}>
          <LogOut size={18} /> Выйти
        </button>
      </div>

      <p className="text-xs text-center" style={{ color: 'var(--text-tertiary)' }}>FinHelper v1.0.0 • Данные хранятся локально</p>
    </div>
  )
}