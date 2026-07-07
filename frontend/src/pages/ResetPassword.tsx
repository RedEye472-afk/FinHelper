import { useState } from 'react'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import { ApiRequestError } from '../api/client'
import { resetPassword } from '../api/auth'
import { Lock, Eye, EyeOff, CheckCircle } from 'lucide-react'

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const token = searchParams.get('token') || ''

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (password.length < 8) { setError('Пароль должен быть минимум 8 символов'); return }
    if (password !== confirm) { setError('Пароли не совпадают'); return }
    if (!token) { setError('Ссылка для сброса пароля недействительна'); return }

    setSubmitting(true)
    try {
      // We need email — ask user or parse from query params
      const email = searchParams.get('email') || ''
      if (!email) {
        setError('Email не указан. Пожалуйста, перейдите по ссылке из письма ещё раз.')
        setSubmitting(false)
        return
      }
      await resetPassword(email, token, password)
      setSuccess(true)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : 'Неверная или просроченная ссылка для сброса')
    } finally { setSubmitting(false) }
  }

  if (success) {
    return (
      <div className="min-h-screen bg-gradient-primary flex items-center justify-center px-4 relative overflow-hidden">
        <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1, transparent)' }} />
        <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2, transparent)' }} />
        <div className="glass p-8 text-center space-y-4 max-w-md mx-auto">
          <div className="w-16 h-16 rounded-full bg-emerald-500/20 flex items-center justify-center mx-auto">
            <CheckCircle size={32} className="text-emerald-500" />
          </div>
          <h2 className="text-xl font-semibold" style={{color: 'var(--text-primary)'}}>Пароль изменён</h2>
          <p className="text-sm" style={{color: 'var(--text-secondary)'}}>Войдите с новым паролем.</p>
          <button onClick={() => navigate('/login', { replace: true })} className="btn btn-primary px-6 py-2.5 text-sm">
            Войти
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-primary flex items-center justify-center px-4 relative overflow-hidden">
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1, transparent)' }} />
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2, transparent)' }} />
      <div className="absolute top-20 left-10 w-32 h-32 rounded-full bg-white/5 blur-3xl" />
      <div className="absolute bottom-20 right-10 w-40 h-40 rounded-full bg-white/5 blur-3xl" />

      <div className="relative w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur-md flex items-center justify-center text-white text-3xl font-bold mx-auto mb-4 shadow-lg animate-float">₽</div>
          <h1 className="text-2xl font-bold text-white">FinHelper</h1>
          <p className="text-sm text-white/60 mt-1">Новый пароль</p>
        </div>

        <form onSubmit={handleSubmit} className="glass p-6 space-y-4">
          <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>Укажите новый пароль</h2>

          {error && (
            <div className="text-sm px-3 py-2.5 rounded-xl" style={{color: 'var(--color-danger-600)', background: 'var(--color-danger-50)', border: '1px solid var(--color-danger-200)'}}>
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Email</label>
            <input
              type="email"
              className="input min-h-[44px]"
              defaultValue={searchParams.get('email') || ''}
              placeholder="you@example.com"
              required
              autoComplete="email"
              onChange={e => {
                // Store email in search params for later use in submit
                const url = new URL(window.location.href)
                url.searchParams.set('email', e.target.value)
                window.history.replaceState({}, '', url.toString())
              }}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Новый пароль</label>
            <div className="relative">
              <input type={showPassword ? 'text' : 'password'} className="input pr-10 min-h-[44px]" value={password} onChange={e => setPassword(e.target.value)} placeholder="Минимум 8 символов" required minLength={8} autoComplete="new-password" />
              <button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Скрыть' : 'Показать'} className="absolute right-2 top-1/2 -translate-y-1/2 w-11 h-11 flex items-center justify-center rounded-lg" style={{color: 'var(--text-tertiary)'}}>
                {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Подтверждение</label>
            <input type="password" className="input min-h-[44px]" value={confirm} onChange={e => setConfirm(e.target.value)} placeholder="Повторите пароль" required autoComplete="new-password" />
          </div>

          <button type="submit" disabled={submitting} className="btn btn-primary w-full py-3 text-base">
            {submitting
              ? <span className="flex items-center gap-2 justify-center"><span className="animate-spin w-4 h-4 border-2 border-white/30 border-t-white rounded-full" /> Сохранение...</span>
              : <span className="flex items-center gap-2 justify-center"><Lock size={16} /> Сохранить пароль</span>
            }
          </button>

          <p className="text-sm text-center" style={{color: 'var(--text-secondary)'}}>
            <Link to="/login" className="font-medium" style={{color: 'var(--color-primary-400)'}}>Вернуться ко входу</Link>
          </p>
        </form>
      </div>
    </div>
  )
}
