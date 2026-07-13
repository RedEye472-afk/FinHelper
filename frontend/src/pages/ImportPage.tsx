/**
 * ImportPage — загрузка выписок (Сбербанк, Т-Банк, CSV)
 * Шаги: загрузка → разбор → подтверждение → импорт
 */
import { useState, useRef, useMemo, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Upload, FileText, AlertCircle, CheckCircle2, Loader2,
  ArrowLeft, ArrowRight, Download, X, TrendingUp, TrendingDown,
} from 'lucide-react'
import { useAccounts, useCreateOperation, useCategories } from '../api/queries'
import { parseSberbankText, parseSberbankCSV, parseSberbankInline, type ParsedTransaction } from '../lib/import/sberbank'
import { extractTextFromPDF } from '../lib/import/pdfExtractor'
import { apiRequest } from '../api/client'
import type { OperationCreate } from '../types'

type Step = 'upload' | 'review' | 'confirm'

const categoryColors: Record<string, string> = {
  'Продукты': '#f59e0b', 'Транспорт': '#3b82f6', 'Рестораны': '#f97316',
  'Жильё': '#ef4444', 'Развлечения': '#8b5cf6', 'Здоровье': '#ec4899',
  'Связь': '#06b6d4', 'Одежда': '#6366f1', 'Образование': '#14b8a6',
  'Подарки': '#f43f5e', 'Спорт': '#a855f7', 'Подписки': '#0ea5e9',
  'Зарплата': '#10b981', 'Фриланс': '#34d399', 'Переводы': '#64748b', 'Прочее': '#6b7280',
}

export function ImportPage() {
  const navigate = useNavigate()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const createOp = useCreateOperation()

  const [step, setStep] = useState<Step>('upload')
  const [rawText, setRawText] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [importing, setImporting] = useState(false)
  const [imported, setImported] = useState(0)
  const [errors, setErrors] = useState<string[]>([])
  const [accountId, setAccountId] = useState<number | null>(null)
  const [format, setFormat] = useState<'pdf' | 'csv'>('pdf')
  const [processing, setProcessing] = useState(false)

  const { data: accounts } = useAccounts()
  const { data: categories } = useCategories()
  const accList = accounts ?? []
  const catMap = useMemo(() => {
    const m = new Map<string, number>()
    if (categories) {
      for (const c of categories) {
        m.set(c.name.toLowerCase(), c.id)
      }
    }
    return m
  }, [categories])

  // Parse transactions from raw text
  const rawLineCount = useMemo(() => rawText.split('\n').filter(l => l.trim()).length, [rawText])
  const transactions = useMemo<ParsedTransaction[]>(() => {
    if (!rawText.trim()) return []
    try {
      // Для PDF (pdf.js координатный вывод) используем inline-парсер
      if (format === 'pdf') {
        return parseSberbankInline(rawText)
      }
      return format === 'csv'
        ? parseSberbankCSV(rawText)
        : parseSberbankText(rawText)
    } catch (e) {
      console.warn('Parser error:', e)
      return []
    }
  }, [rawText, format])
  const totalIncome = useMemo(
    () => transactions.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0),
    [transactions],
  )
  const totalExpense = useMemo(
    () => transactions.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0),
    [transactions],
  )

  // Select all/none
  const toggleAll = useCallback(() => {
    if (selectedIds.size === transactions.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(transactions.map((_, i) => i)))
    }
  }, [transactions, selectedIds])

  const toggleOne = useCallback((id: number) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }, [])

  const validTransactions = useMemo(
    () => transactions.filter((_, i) => selectedIds.has(i)),
    [transactions, selectedIds],
  )

  // Bulk import — батчами по 5
  const handleImport = async () => {
    if (!accountId || validTransactions.length === 0) return
    setImporting(true)
    setErrors([])
    setImported(0)

    let success = 0
    const errs: string[] = []
    const BATCH = 5

    for (let i = 0; i < validTransactions.length; i += BATCH) {
      const batch = validTransactions.slice(i, i + BATCH)
      const results = await Promise.allSettled(
        batch.map(async (tx) => {
          const payload: OperationCreate = {
            account_id: accountId,
            type: tx.amount > 0 ? 'income' : 'expense',
            amount: Math.abs(tx.amount).toFixed(2),
            operation_date: tx.date,
            description: tx.description || tx.category,
            category_id: catMap.get(tx.category.toLowerCase()),
            calc_id: crypto.randomUUID(),
          }
          await createOp.mutateAsync(payload)
          return tx
        })
      )

      for (let j = 0; j < results.length; j++) {
        const r = results[j]
        const tx = batch[j]
        if (r.status === 'fulfilled') {
          success++
          setImported(success)
        } else {
          const msg = r.reason instanceof Error ? r.reason.message : 'Ошибка'
          errs.push(`${tx.date} ${tx.description || tx.category}: ${msg}`)
        }
      }
    }

    setErrors(errs)
    setStep('confirm')
    setImporting(false)
  }

  const handleFile = async (file: File) => {
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (ext === 'csv') {
      setFormat('csv')
      const text = await file.text()
      setRawText(text)
      setStep('review')
    } else {
      setFormat('pdf')
      setProcessing(true)
      try {
        // Try backend PDF parser first
        const formData = new FormData()
        formData.append('file', file)
        const res = await apiRequest<{ text: string; line_count: number; fallback?: boolean }>(
          'POST', '/api/v1/import/parse-pdf', formData, { skipAuth: true }
        )
        // If backend returned empty text or fallback flag, use client-side pdf.js
        if (!res.text || res.fallback) {
          throw new Error('Backend PDF parser unavailable, using client-side fallback')
        }
        setRawText(res.text)
        setStep('review')
      } catch {
        // Fallback: client-side pdf.js
        try {
          const buffer = await file.arrayBuffer()
          const text = await extractTextFromPDF(buffer)
          setRawText(text)
          setStep('review')
        } catch (e2) {
          // Last fallback: read as plain text
          const text = await file.text()
          setRawText(text)
          setStep('review')
        }
      } finally {
        setProcessing(false)
      }
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-2">
        <button onClick={() => navigate(-1)} className="p-1 rounded-lg" style={{ color: '#94A3B8' }}>
          <ArrowLeft size={20} />
        </button>
        <h1 className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>Импорт выписки</h1>
      </div>

      {step === 'upload' && (
        <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="space-y-4">
          {/* Drop zone */}
          <div
            onDragOver={e => { e.preventDefault(); setDragOver(true) }}
            onDragLeave={() => setDragOver(false)}
            onDrop={e => { e.preventDefault(); setDragOver(false); const f = e.dataTransfer.files[0]; if (f) handleFile(f) }}
            className="rounded-[20px] border-2 border-dashed p-8 text-center cursor-pointer transition-all"
            style={{
              borderColor: dragOver ? '#6E56CF' : 'rgba(255,255,255,0.1)',
              background: dragOver ? 'rgba(110,86,207,0.05)' : 'var(--bg-card)',
            }}
            onClick={() => fileInputRef.current?.click()}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.pdf,.txt"
              className="hidden"
              onChange={e => { const f = e.target.files?.[0]; if (f) handleFile(f) }}
            />
            <div className="w-14 h-14 rounded-2xl flex items-center justify-center mx-auto mb-3"
              style={{ background: 'rgba(110,86,207,0.12)' }}>
              <Upload size={24} style={{ color: '#A78BFA' }} />
            </div>
            <p className="text-base font-semibold mb-1">Загрузите выписку</p>
            <p className="text-sm mb-4" style={{ color: '#64748B' }}>
              PDF или CSV из Сбербанка, Т-Банка
            </p>
            <div className="flex gap-3 justify-center text-xs" style={{ color: '#64748B' }}>
              <span className="px-3 py-1.5 rounded-full" style={{ background: 'rgba(255,255,255,0.04)' }}>PDF</span>
              <span className="px-3 py-1.5 rounded-full" style={{ background: 'rgba(255,255,255,0.04)' }}>CSV</span>
              <span className="px-3 py-1.5 rounded-full" style={{ background: 'rgba(255,255,255,0.04)' }}>TXT</span>
            </div>
          </div>

          {/* Processing indicator */}
          {processing && (
            <div className="rounded-[16px] border p-5 text-center" style={{ borderColor: 'rgba(110,86,207,0.2)', background: 'rgba(110,86,207,0.04)' }}>
              <Loader2 size={24} className="animate-spin mx-auto mb-2" style={{ color: '#A78BFA' }} />
              <p className="text-sm font-medium" style={{ color: '#94A3B8' }}>Обработка PDF...</p>
              <p className="text-xs mt-1" style={{ color: '#64748B' }}>Извлекаем текст и разбираем операции</p>
            </div>
          )}

          {/* Or paste */}
          <details className="rounded-[16px] border" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
            <summary className="p-3 text-sm font-medium cursor-pointer" style={{ color: '#94A3B8' }}>
              Или вставьте текст
            </summary>
            <div className="p-3 pt-0">
              <textarea
                rows={6}
                className="w-full rounded-xl p-3 text-sm font-mono mt-2"
                style={{
                  background: 'var(--bg-surface)',
                  borderColor: 'var(--border-default)',
                  color: 'var(--text-primary)',
                }}
                placeholder="Вставьте текст выписки..."
                value={rawText}
                onChange={e => setRawText(e.target.value)}
              />
              <button
                onClick={() => { if (rawText.trim()) setStep('review') }}
                disabled={!rawText.trim()}
                className="w-full py-2.5 rounded-xl font-medium text-sm mt-2 disabled:opacity-50"
                style={{
                  background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
                  color: '#fff',
                }}
              >
                Разобрать
              </button>
            </div>
          </details>

          {/* Format reference */}
          <div className="rounded-[16px] border p-4" style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
            <p className="text-sm font-semibold mb-2">Поддерживаемые форматы</p>
            <div className="space-y-2 text-xs" style={{ color: '#64748B' }}>
              <p><span className="font-medium" style={{ color: '#94A3B8' }}>Сбербанк PDF</span> — выписка по счёту</p>
              <p><span className="font-medium" style={{ color: '#94A3B8' }}>Сбербанк CSV</span> — экспорт операций (; разделитель)</p>
              <p><span className="font-medium" style={{ color: '#94A3B8' }}>Текст</span> — скопируйте таблицу из PDF</p>
            </div>
          </div>
        </motion.div>
      )}

      {step === 'review' && (
        <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="space-y-4">
          {/* Summary */}
          <div className="rounded-[16px] border p-4" style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <FileText size={18} style={{ color: '#A78BFA' }} />
                <p className="text-sm font-semibold">
                  Найдено {transactions.length} операций
                </p>
                <p className="text-[10px] ml-2" style={{ color: '#64748B' }}>
                  из {rawLineCount} строк
                </p>
              </div>
              <div className="flex gap-2">
                <button onClick={() => setStep('upload')} className="p-1.5 rounded-lg" style={{ color: '#64748B' }}>
                  <X size={16} />
                </button>
              </div>
            </div>
            {transactions.length === 0 && rawText.trim() && (
              <details className="mt-2">
                <summary className="text-xs cursor-pointer" style={{ color: '#F43F5E' }}>
                  🔍 Сырой текст ({rawLineCount} строк, показаны первые 30)
                </summary>
                <pre className="text-[10px] mt-2 p-2 rounded-lg max-h-40 overflow-y-auto font-mono leading-tight"
                  style={{ background: 'rgba(0,0,0,0.2)', color: '#94A3B8' }}>
                  {rawText.split('\n').slice(0, 30).join('\n')}
                </pre>
              </details>
            )}
            <div className="grid grid-cols-2 gap-3 mt-3">
              <div className="rounded-xl p-3" style={{ background: 'rgba(34,197,94,0.08)' }}>
                <p className="text-lg font-bold font-mono-money" style={{ color: '#22C55E' }}>
                  +{totalIncome.toLocaleString('ru-RU', { minimumFractionDigits: 2 })} ₽
                </p>
                <p className="text-[10px]" style={{ color: '#64748B' }}>Доходы</p>
              </div>
              <div className="rounded-xl p-3" style={{ background: 'rgba(244,63,94,0.08)' }}>
                <p className="text-lg font-bold font-mono-money" style={{ color: '#F43F5E' }}>
                  −{totalExpense.toLocaleString('ru-RU', { minimumFractionDigits: 2 })} ₽
                </p>
                <p className="text-[10px]" style={{ color: '#64748B' }}>Расходы</p>
              </div>
            </div>
          </div>

          {/* Account select */}
          <div className="rounded-[16px] border p-3" style={{ borderColor: 'rgba(255,255,255,0.06)', background: 'var(--bg-card)' }}>
            <div className="flex items-center justify-between mb-2">
              <p className="text-xs font-medium" style={{ color: '#64748B' }}>Счёт для импорта</p>
              {accountId && (
                <button
                  onClick={async () => {
                    if (!confirm('Удалить ВСЕ операции этого счёта? Это действие необратимо.')) return
                    try {
                      await apiRequest('DELETE', `/api/v1/operations/bulk?account_id=${accountId}`)
                      alert('Операции удалены')
                      // Reload page to refresh data
                      window.location.reload()
                    } catch (e) {
                      console.error(e)
                      alert('Ошибка удаления')
                    }
                  }}
                  className="text-xs px-2 py-1 rounded border"
                  style={{ borderColor: 'rgba(244,63,94,0.3)', color: '#F43F5E' }}
                >
                  🗑️ Очистить счёт
                </button>
              )}
            </div>
            <div className="flex gap-2 flex-wrap">
              {accList.map(a => (
                <button key={a.id} onClick={() => setAccountId(a.id)}
                  className="px-3 py-1.5 rounded-full text-xs font-medium transition-all"
                  style={accountId === a.id
                    ? { background: '#6E56CF', color: '#fff' }
                    : { background: 'rgba(255,255,255,0.04)', color: '#94A3B8' }
                  }>
                  {a.name}
                </button>
              ))}
            </div>
          </div>

          {/* Select all */}
          <div className="flex items-center justify-between">
            <button onClick={toggleAll} className="text-xs font-medium" style={{ color: '#A78BFA' }}>
              {selectedIds.size === transactions.length ? 'Снять все' : `Выбрать все (${transactions.length})`}
            </button>
            <p className="text-xs" style={{ color: '#64748B' }}>
              Выбрано {selectedIds.size}
            </p>
          </div>

          {/* Transaction list */}
          <div className="space-y-1 max-h-96 overflow-y-auto rounded-[16px] border"
            style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
            {transactions.map((tx, i) => {
              const selected = selectedIds.has(i)
              return (
                <button key={i} onClick={() => toggleOne(i)}
                  className="w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors"
                  style={{
                    background: selected ? 'rgba(110,86,207,0.04)' : 'transparent',
                    borderBottom: i < transactions.length - 1 ? '1px solid rgba(255,255,255,0.03)' : 'none',
                  }}>
                  <div className={`w-5 h-5 rounded-md border flex items-center justify-center shrink-0 transition-colors ${selected ? 'border-none' : ''}`}
                    style={selected
                      ? { background: '#6E56CF' }
                      : { borderColor: 'rgba(255,255,255,0.15)' }
                    }>
                    {selected && <CheckCircle2 size={14} style={{ color: '#fff' }} />}
                  </div>
                  <div className="w-1.5 h-1.5 rounded-full shrink-0"
                    style={{ background: categoryColors[tx.category] || '#6b7280' }} />
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium truncate">{tx.description}</p>
                    <p className="text-[10px]" style={{ color: '#64748B' }}>
                      {tx.category} · {tx.date}
                    </p>
                  </div>
                  <span className={`text-xs font-semibold font-mono-money shrink-0 ${tx.amount > 0 ? '' : ''}`}
                    style={{ color: tx.amount > 0 ? '#22C55E' : '#F43F5E' }}>
                    {tx.amount > 0 ? '+' : ''}{tx.amount.toFixed(2)} ₽
                  </span>
                </button>
              )
            })}
          </div>

          {/* Import button */}
          <button
            onClick={handleImport}
            disabled={importing || !accountId || selectedIds.size === 0}
            className="w-full py-3 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 disabled:opacity-50"
            style={{
              background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
              color: '#fff',
            }}
          >
            {importing ? (
              <><Loader2 size={18} className="animate-spin" /> Импортируется {imported}/{validTransactions.length}...</>
            ) : (
              <><Download size={18} /> Импортировать {selectedIds.size} операций</>
            )}
          </button>
        </motion.div>
      )}

      {step === 'confirm' && (
        <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} className="space-y-4">
          <div className="rounded-[20px] border p-8 text-center"
            style={{
              borderColor: imported > 0 ? 'rgba(34,197,94,0.2)' : 'rgba(244,63,94,0.2)',
              background: imported > 0 ? 'rgba(34,197,94,0.04)' : 'rgba(244,63,94,0.04)',
            }}>
            <div className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
              style={{ background: imported > 0 ? 'rgba(34,197,94,0.12)' : 'rgba(244,63,94,0.12)' }}>
              {imported > 0
                ? <CheckCircle2 size={32} style={{ color: '#22C55E' }} />
                : <X size={32} style={{ color: '#F43F5E' }} />}
            </div>
            <p className="text-xl font-bold mb-1">
              {imported > 0 ? 'Импорт завершён' : 'Ошибка импорта'}
            </p>
            <p className="text-sm mb-2" style={{ color: '#94A3B8' }}>
              {imported > 0
                ? `Успешно: <span class="font-semibold" style="color:#22C55E">${imported}</span>`
                : 'Не удалось импортировать ни одной операции'}
            </p>
            {errors.length > 0 && (
              <div className="rounded-xl p-3 mt-3 text-left text-xs" style={{ background: 'rgba(244,63,94,0.08)' }}>
                <p className="font-medium mb-1" style={{ color: '#F43F5E' }}>Ошибки ({errors.length}):</p>
                {errors.slice(0, 5).map((e, i) => (
                  <p key={i} className="mb-0.5" style={{ color: '#94A3B8' }}>{e}</p>
                ))}
              </div>
            )}
          </div>

          <div className="flex gap-3">
            <button onClick={() => { setRawText(''); setSelectedIds(new Set()); setStep('upload') }}
              className="flex-1 py-2.5 rounded-xl font-medium text-sm"
              style={{ background: 'rgba(255,255,255,0.06)', color: '#94A3B8' }}>
              Ещё импорт
            </button>
            <button onClick={() => navigate('/operations')}
              className="flex-1 py-2.5 rounded-xl font-medium text-sm"
              style={{
                background: 'linear-gradient(135deg, #6E56CF, #A78BFA)',
                color: '#fff',
              }}>
              К операциям
            </button>
          </div>
        </motion.div>
      )}
    </div>
  )
}
