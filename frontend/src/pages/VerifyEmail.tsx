import { useState, useRef, type KeyboardEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { ApiRequestError } from '../api/client'
import { Mail, ArrowLeft } from 'lucide-react'

export function VerifyEmailPage() {
  const { verifyEmail } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [digits, setDigits] = useState<string[]>(Array(6).fill(''))
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const inputsRef = useRef<(HTMLInputElement | null)[]>([])
  const email = searchParams.get('email') || ''

  const code = digits.join('')

  const handleChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return
    const newDigits = [...digits]
    newDigits[index] = value.slice(-1)
    setDigits(newDigits)
    if (value && index < 5) {
      inputsRef.current[index + 1]?.focus()
    }
  }

  const handleKeyDown = (index: number, e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !digits[index] && index > 0) {
      inputsRef.current[index - 1]?.focus()
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (code.length !== 6) { setError('Введите полный код'); return }
    setSubmitting(true); setError('')
    try {
      await verifyEmail(code)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : 'Неверный или просроченный код')
    } finally { setSubmitting(false) }
  }

  return (
    <div className="min-h-screen bg-gradient-primary flex items-center justify-center px-4 relative overflow-hidden">
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-1, transparent)' }} />
      <div className="absolute inset-0" style={{ background: 'var(--hero-circle-2, transparent)' }} />
      <div className="absolute top-20 right-10 w-36 h-36 rounded-full bg-white/5 blur-3xl" />
      <div className="absolute bottom-20 left-10 w-28 h-28 rounded-full bg-white/5 blur-3xl" />

      <div className="relative w-full max-w-md">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur-md flex items-center justify-center text-white text-3xl font-bold mx-auto mb-4 shadow-lg animate-float">
            <Mail size={28} />
          </div>
          <h1 className="text-2xl font-bold text-white">Подтверждение email</h1>
          <p className="text-sm text-white/60 mt-1">
            {email ? `Код отправлен на ${email}` : 'Введите 6-значный код из письма'}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="glass p-6 space-y-5">
          {error && (
            <div className="text-sm px-3 py-2.5 rounded-xl" style={{color: 'var(--color-danger-600)', background: 'var(--color-danger-50)', border: '1px solid var(--color-danger-200)'}}>
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium mb-3 text-center" style={{color: 'var(--text-primary)'}}>
              Код подтверждения
            </label>
            <div className="flex gap-2 justify-center">
              {digits.map((d, i) => (
                <input
                  key={i}
                  ref={el => { inputsRef.current[i] = el }}
                  type="text"
                  inputMode="numeric"
                  maxLength={1}
                  value={d}
                  onChange={e => handleChange(i, e.target.value)}
                  onKeyDown={e => handleKeyDown(i, e)}
                  className="w-12 h-14 text-center text-xl font-bold rounded-xl border-2 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-all"
                  style={{
                    background: 'var(--bg-secondary)',
                    color: 'var(--text-primary)',
                    borderColor: d ? 'var(--color-primary-400)' : 'var(--border-color)',
                  }}
                  autoComplete="one-time-code"
                />
              ))}
            </div>
          </div>

          <button type="submit" disabled={submitting || code.length !== 6} className="btn btn-primary w-full py-3 text-base">
            {submitting
              ? <span className="flex items-center gap-2 justify-center"><span className="animate-spin w-4 h-4 border-2 border-white/30 border-t-white rounded-full" /> Проверка...</span>
              : 'Подтвердить'
            }
          </button>

          <div className="flex justify-between text-sm">
            <button type="button" onClick={() => navigate(-1)} className="flex items-center gap-1" style={{color: 'var(--text-secondary)'}}>
              <ArrowLeft size={14} /> Назад
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
