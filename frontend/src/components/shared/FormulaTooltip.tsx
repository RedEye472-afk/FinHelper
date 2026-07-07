import { useState } from 'react'
import { ChevronDown, ChevronUp, FunctionSquare } from 'lucide-react'
import { MathFormula } from './MathFormula'

interface FormulaTooltipProps { title: string; formula: string; explanation?: string }

export function FormulaTooltip({ title, formula, explanation }: FormulaTooltipProps) {
  const [open, setOpen] = useState(false)
  return (
    <div className="border border-gray-200 rounded-xl overflow-hidden">
      <button onClick={() => setOpen(!open)} className="flex items-center justify-between w-full px-4 py-3 text-sm bg-gray-50 hover:bg-gray-100 transition-colors">
        <div className="flex items-center gap-2">
          <FunctionSquare size={16} className="text-primary-500" />
          <span className="font-medium text-gray-700">{title}</span>
        </div>
        {open ? <ChevronUp size={16} className="text-gray-400" /> : <ChevronDown size={16} className="text-gray-400" />}
      </button>
      {open && (
        <div className="px-4 py-3 border-t border-gray-200">
          <div className="flex justify-center"><MathFormula formula={formula} /></div>
          {explanation && <p className="text-xs text-gray-500 mt-2">{explanation}</p>}
        </div>
      )}
    </div>
  )
}
