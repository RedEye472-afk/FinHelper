import { AlertTriangle } from 'lucide-react'

export function Disclaimer({ text }: { text?: string }) {
  return (
    <div className="flex items-start gap-2 p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-700">
      <AlertTriangle size={14} className="mt-0.5 flex-shrink-0" />
      <p>{text || 'Результаты расчётов носят информационный характер и не являются финансовой рекомендацией. Для принятия решений обратитесь к специалисту.'}</p>
    </div>
  )
}
