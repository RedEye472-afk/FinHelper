import { useEffect, useRef } from 'react'
import katex from 'katex'

interface MathFormulaProps { formula: string; displayMode?: boolean }

export function MathFormula({ formula, displayMode = true }: MathFormulaProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) {
      try {
        katex.render(formula, ref.current, { displayMode, throwOnError: false })
      } catch {
        ref.current.textContent = formula
      }
    }
  }, [formula, displayMode])

  return <div ref={ref} className="my-2 overflow-x-auto" />
}
