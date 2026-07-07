import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { ApiRequestError } from '../api/client'
import { LogIn, Eye, EyeOff } from 'lucide-react'

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('demo@finhelper.ru')
  const [password, setPassword] = useState('demo1234')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(''); setSubmitting(true)
    try { await login(email, password); navigate('/dashboard', { replace: true }) }
    catch (err) { setError(err instanceof ApiRequestError ? err.message : 'Ошибка подключения к серверу') }
    finally { setSubmitting(false) }
  }

  return (
    <div className="min-h-screen bg-gradient-primary flex items-center justify-center px-4 relative overflow-hidden">
      {/* Decorative circles */}
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1, transparent)' }} />
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2, transparent)' }} />
      <div className="absolute top-20 left-10 w-32 h-32 rounded-full bg-white/5 blur-3xl" />
      <div className="absolute bottom-20 right-10 w-40 h-40 rounded-full bg-white/5 blur-3xl" />

      <div className="relative w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur-md flex items-center justify-center text-white text-3xl font-bold mx-auto mb-4 shadow-lg animate-float">₽</div>
          <h1 className="text-2xl font-bold text-white">FinHelper</h1>
          <p className="text-sm text-white/60 mt-1">Финансовый навигатор</p>
        </div>
        <form onSubmit={handleSubmit} className="glass p-6 space-y-4">
          <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>Вход</h2>
          {error && (
            <div className="text-sm px-3 py-2.5 rounded-xl" style={{color: 'var(--color-danger-600)', background: 'var(--color-danger-50)', border: '1px solid var(--color-danger-200)'}}>
              {error}
            </div>
          )}
          <div className="text-sm px-3 py-2.5 rounded-xl" style={{color: 'var(--color-primary-600)', background: 'var(--color-primary-50)', border: '1px solid var(--color-primary-200)'}}>
            <strong>Демо-доступ:</strong> email <code>demo@finhelper.ru</code>, пароль <code>demo1234</code> — уже заполнены, просто нажмите «Войти».
          </div>
          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Email</label>
            <input type="email" name="email" className="input min-h-[44px]" value={email} onChange={e => setEmail(e.target.value)} placeholder="you@example.com" required autoComplete="email" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1" style={{color: 'var(--text-primary)'}}>Пароль</label>
            <div className="relative">
              <input type={showPassword ? 'text' : 'password'} name="password" className="input pr-10 min-h-[44px]" value={password} onChange={e => setPassword(e.target.value)} placeholder="••••••••" required autoComplete="current-password" />
                            <button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Скрыть пароль' : 'Показать пароль'} className="absolute right-2 top-1/2 -translate-y-1/2 w-11 h-11 flex items-center justify-center rounded-lg" style={{color: 'var(--text-tertiary)'}}>
                {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
          </div>
          <button type="submit" disabled={submitting} className="btn btn-primary w-full py-3 text-base">
            {submitting ? <span className="flex items-center gap-2"><span className="animate-spin w-4 h-4 border-2 border-white/30 border-t-white rounded-full" /> Вход...</span> : <span className="flex items-center gap-2 justify-center"><LogIn size={16} /> Войти</span>}
          </button>
          <p className="text-sm text-center" style={{color: 'var(--text-secondary)'}}>
            Нет аккаунта? <Link to="/register" className="font-medium" style={{color: 'var(--color-primary-400)'}}>Зарегистрироваться</Link>
          </p>
        </form>
      </div>
    </div>
  )
}
