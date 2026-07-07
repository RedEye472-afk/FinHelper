import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'

type ToastType = 'success' | 'error' | 'info' | 'warning'

interface Toast { id: number; type: ToastType; message: string }

interface ToastContextType { toast: (type: ToastType, message: string) => void }

const ToastContext = createContext<ToastContextType | null>(null)

const icons = { success: CheckCircle, error: AlertCircle, info: Info, warning: AlertTriangle }
const bgColors = { success: '#10b981', error: '#ef4444', info: '#3b82f6', warning: '#f59e0b' }

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((type: ToastType, message: string) => {
    const id = Date.now()
    setToasts(prev => [...prev, { id, type, message }])
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 3500)
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed top-4 right-4 z-[100] flex flex-col gap-2 max-w-sm w-full pointer-events-none">
        {toasts.map(t => {
          const Icon = icons[t.type]
          return (
            <div key={t.id}
              className="text-white px-4 py-3 rounded-xl flex items-start gap-3 animate-slide-up pointer-events-auto shadow-2xl"
              style={{background: bgColors[t.type]}}>
              <Icon size={18} className="mt-0.5 flex-shrink-0" />
              <p className="text-sm flex-1">{t.message}</p>
              <button onClick={() => setToasts(prev => prev.filter(x => x.id !== t.id))} className="text-white/70 hover:text-white">
                <X size={14} />
              </button>
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}
