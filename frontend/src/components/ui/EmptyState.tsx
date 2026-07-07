import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps { icon: LucideIcon; title: string; description?: string; action?: { label: string; onClick: () => void } }

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center animate-fade-in">
      <div className="w-20 h-20 rounded-3xl flex items-center justify-center mb-5"
        style={{background: 'var(--color-primary-50)'}}>
        <Icon size={36} style={{color: 'var(--color-primary-400)'}} />
      </div>
      <p className="text-lg font-semibold mb-1" style={{color: 'var(--text-primary)'}}>{title}</p>
      {description && <p className="text-sm mb-5 max-w-xs" style={{color: 'var(--text-secondary)'}}>{description}</p>}
      {action && (
        <button onClick={action.onClick}
          className="btn btn-primary px-5 py-2 text-sm">
          {action.label}
        </button>
      )}
    </div>
  )
}
