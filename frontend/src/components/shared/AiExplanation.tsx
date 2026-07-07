import { useState } from 'react'
import { Sparkles, ChevronDown, ChevronUp } from 'lucide-react'

interface AiExplanationProps { title?: string; children: React.ReactNode }

export function AiExplanation({ title = 'Объяснение', children }: AiExplanationProps) {
  const [open, setOpen] = useState(false)
  return (
    <div className="border border-primary-200 bg-primary-50/50 rounded-xl overflow-hidden">
      <button onClick={() => setOpen(!open)} className="flex items-center justify-between w-full px-4 py-3 text-sm">
        <div className="flex items-center gap-2">
          <Sparkles size={16} className="text-primary-500" />
          <span className="font-medium text-primary-700">{title}</span>
        </div>
        {open ? <ChevronUp size={16} className="text-primary-400" /> : <ChevronDown size={16} className="text-primary-400" />}
      </button>
      {open && <div className="px-4 pb-3 text-sm text-gray-700 leading-relaxed space-y-2">{children}</div>}
    </div>
  )
}
