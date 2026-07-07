import { Download } from 'lucide-react'
import { useToast } from '../ui/Toast'
import { Card } from '../ui/Card'

export function DataExportImport() {
  const { toast } = useToast()

  const handleExport = () => {
    toast('info', 'Экспорт данных пока недоступен в серверной версии')
  }

  return (
    <Card>
      <h3 className="text-sm font-semibold text-gray-900 mb-3">Управление данными</h3>
      <button onClick={handleExport} className="flex items-center gap-1.5 w-full py-2.5 rounded-xl bg-primary-50 text-primary-600 font-medium text-xs hover:bg-primary-100 transition-colors">
        <Download size={14} /> Экспорт JSON
      </button>
    </Card>
  )
}