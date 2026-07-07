import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiRequestError } from '../api/client'
import { forgotPassword } from '../api/auth'
import { Mail, ArrowLeft, CheckCircle } from 'lucide-react'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email) { setError('Введите email'); return }
    setSubmitting(true); setError('')
    try {
      await forgotPassword(email)
      setSuccess(true)
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : 'Ошибка подключения к серверу')
    } finally { setSubmitting(false) }
  }

  if (success) {
    return (
      <div className="min-h-screen bg-gradient-primary flex items-center justify-center px-4 relative overflow-hidden">
        <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1, transparent)' }} />
        <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2, transparent)' }} />
        <div className="absolute top-20 left-10 w-32 h-32 rounded-full bg-white/5 blur-3xl" />
        <div className="absolute bottom-20 right-10 w-40 h-40 rounded-full bg-white/5 blur-3xl" />
        <div className="relative w-full max-w-md">
          <div className="glass p-8 text-center space-y-4">
            <div className="w-16 h-16 rounded-full bg-emerald-500/20 flex items-center justify-center mx-auto">
              <CheckCircle size={32} className="text-emerald-500" />
            </div>
            <h2 className="text-xl font-semibold" style={{color: 'var(--text-primary)'}}>Письмо отправлено</h2>
            <p className="text-sm" style={{color: 'var(--text-secondary)'}}>
              Если аккаунт с email <strong>{email}</strong> существует, мы отправили ссылку для сброса пароля.
            </p>
            <Link to="/login" className="btn btn-primary inline-block px-6 py-2.5 text-sm">
              Вернуться ко входу
            </Link>
          </div>
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
          <p className="text-sm text-white/60 mt-1">Восстановление пароля</p>
        </div>

        <form onSubmit={handleSubmit} className="glass p-6 space-y-4">
          <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>Забыли пароль?</h2>
          <p className="text-sm" style={{color: 'var(--text-secondary)'}}>
            Введите email, привязанный к аккаунту. Мы отправим ссылку для сброса пароля.
          </p>

          {error && (
            <div className="text-sm px-3 py-2.5 rounded-xl" style={{color: 'var(--color-danger-600)', background: 'var(--color-danger-50)', border: '1px solid var(--color-danger-200)'}}>
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Email</label>
            <input type="email" className="input min-h-[44px]" value={email} onChange={e => setEmail(e.target.value)} placeholder="you@example.com" required autoComplete="email" />
          </div>

          <button type="submit" disabled={submitting} className="btn btn-primary w-full py-3 text-base">
            {submitting
              ? <span className="flex items-center gap-2 justify-center"><span className="animate-spin w-4 h-4 border-2 border-white/30 border-t-white rounded-full" /> Отправка...</span>
              : <span className="flex items-center gap-2 justify-center"><Mail size={16} /> Отправить ссылку</span>
            }
          </button>

          <p className="text-sm text-center" style={{color: 'var(--text-secondary)'}}>
            <Link to="/login" className="flex items-center gap-1 justify-center font-medium" style={{color: 'var(--color-primary-400)'}}>
              <ArrowLeft size={14} /> Вернуться ко входу
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}
