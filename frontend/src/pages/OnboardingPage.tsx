import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, ArrowLeft, Check } from 'lucide-react'

const steps = [
  {
    title: 'Добро пожаловать в FinHelper',
    desc: 'Ваш персональный финансовый навигатор. Отслеживайте доходы и расходы, ставьте цели, планируйте бюджет и принимайте взвешенные финансовые решения.',
    features: ['📊 Учёт доходов и расходов', '🎯 Финансовые цели', '📋 Бюджеты по категориям'],
  },
  {
    title: 'Калькуляторы и аналитика',
    desc: 'Используйте встроенные финансовые калькуляторы для планирования кредитов, депозитов, ипотеки и оценки доступности.',
    features: ['🏦 Депозитный калькулятор', '💳 Кредитный калькулятор', '📈 Сравнение ипотеки и аренды'],
  },
  {
    title: 'Всё под контролем',
    desc: 'FinHelper помогает держать финансы под контролем с помощью понятной аналитики и точных математических расчётов.',
    features: ['🔒 Данные хранятся локально', '📱 Удобный мобильный интерфейс', '🎨 Сменные темы оформления'],
  },
]

export function OnboardingPage() {
  const [step, setStep] = useState(0)
  const navigate = useNavigate()

  const handleFinish = () => navigate('/dashboard')

  return (
    <div className="min-h-screen bg-banking-gradient flex flex-col relative overflow-hidden">
      <div className="absolute inset-0" style={{ background: 'var(--bg-hero-circle-1, transparent)' }} />
      <div className="absolute inset-0" style={{ background: 'var(--bg-hero-circle-2, transparent)' }} />
      <div className="relative flex-1 flex flex-col items-center justify-center px-6 text-center">
        <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur flex items-center justify-center text-white text-3xl font-bold mx-auto mb-6">₽</div>
        <h1 className="text-2xl font-bold text-white mb-3">{steps[step].title}</h1>
        <p className="text-sm text-white/80 mb-6 max-w-sm">{steps[step].desc}</p>
        <div className="space-y-3 text-left max-w-xs mx-auto">
          {steps[step].features.map((f, i) => (
            <div key={i} className="flex items-center gap-3 bg-white/10 rounded-xl px-4 py-3 backdrop-blur">
              <span className="text-lg">{f.split(' ')[0]}</span>
              <span className="text-sm text-white/90">{f.slice(f.indexOf(' ') + 1)}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="relative px-6 pb-10">
        <div className="flex justify-center gap-2 mb-6">
          {steps.map((_, i) => (
            <div key={i} className={`w-2 h-2 rounded-full transition-all ${i === step ? 'bg-white w-6' : 'bg-white/40'}`} />
          ))}
        </div>
        <div className="flex justify-between">
          {step > 0 ? (
            <button onClick={() => setStep(step - 1)} className="flex items-center gap-1.5 px-4 py-2.5 rounded-xl bg-white/15 text-white text-sm font-medium backdrop-blur hover:bg-white/25 transition-colors">
              <ArrowLeft size={16} /> Назад
            </button>
          ) : <div />}
          {step < steps.length - 1 ? (
            <button onClick={() => setStep(step + 1)} className="flex items-center gap-1.5 px-5 py-2.5 rounded-xl bg-white text-primary-600 text-sm font-medium hover:bg-white/90 transition-colors btn-press">
              Далее <ArrowRight size={16} />
            </button>
          ) : (
            <button onClick={handleFinish} className="flex items-center gap-1.5 px-5 py-2.5 rounded-xl bg-white text-primary-600 text-sm font-medium hover:bg-white/90 transition-colors btn-press">
              Начать <Check size={16} />
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
