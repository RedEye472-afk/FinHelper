import { useEffect, type ReactNode } from 'react'

interface ModalProps { open: boolean; onClose: () => void; title?: string; children: ReactNode }

export function Modal({ open, onClose, title, children }: ModalProps) {
  useEffect(() => {
    if (open) document.body.style.overflow = 'hidden'
    else document.body.style.overflow = ''
    return () => { document.body.style.overflow = '' }
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in" />
      <div
        className="relative rounded-2xl w-full max-w-md p-6 animate-scale-in shadow-2xl"
        style={{background: 'var(--bg-elevated)', border: '1px solid var(--border-default)'}}
        onClick={e => e.stopPropagation()}
      >
        {title && (
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold" style={{color: 'var(--text-primary)'}}>{title}</h2>
            <button onClick={onClose} className="p-1 rounded-lg hover:bg-gray-100 transition-colors" style={{color: 'var(--text-tertiary)'}}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
            </button>
          </div>
        )}
        {children}
      </div>
    </div>
  )
}
