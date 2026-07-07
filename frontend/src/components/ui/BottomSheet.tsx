import { useEffect, type ReactNode } from 'react'

interface BottomSheetProps { open: boolean; onClose: () => void; title?: string; children: ReactNode; scrollBody?: boolean }

export function BottomSheet({ open, onClose, title, children, scrollBody }: BottomSheetProps) {
  useEffect(() => {
    if (open) { document.body.style.overflow = 'hidden' }
    else { document.body.style.overflow = '' }
    return () => { document.body.style.overflow = '' }
  }, [open])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    if (open) window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-fade-in" />
      <div
        className={`relative rounded-t-2xl w-full max-w-lg animate-slide-up shadow-2xl ${scrollBody ? 'max-h-[85vh] flex flex-col' : ''}`}
        style={{background: 'var(--bg-elevated)', border: '1px solid var(--border-default)'}}
        onClick={e => e.stopPropagation()}
      >
        <div className="absolute top-2 left-1/2 -translate-x-1/2 w-10 h-1 rounded-full" style={{background: 'var(--text-tertiary)'}} />
        <div className="flex items-center justify-between px-5 pt-5 pb-2" style={{borderBottom: scrollBody ? 'none' : undefined}}>
          {title && <h2 className="text-base font-semibold" style={{color: 'var(--text-primary)'}}>{title}</h2>}
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-gray-100 transition-colors ml-auto" style={{color: 'var(--text-tertiary)'}}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
          </button>
        </div>
        <div className={scrollBody ? 'overflow-y-auto p-5 pb-8' : 'p-5 pb-8'}>
          {children}
        </div>
      </div>
    </div>
  )
}
