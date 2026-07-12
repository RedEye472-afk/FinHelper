/**
 * Виртуальная таблица с сортировкой.
 * Использует @tanstack/react-table + @tanstack/react-virtual.
 *
 * Для списка операций (4000+ строк) — без тормозов.
 */
import { useRef, useMemo, useState } from 'react'
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import { useVirtualizer } from '@tanstack/react-virtual'

interface VirtualTableProps<T> {
  data: T[]
  columns: ColumnDef<T>[]
  /** Высота контейнера в px (по умолчанию 600) */
  height?: number
  /** Высота строки в px (по умолчанию 48) */
  rowHeight?: number
  /** Начальная сортировка */
  initialSort?: SortingState
}

export function VirtualTable<T extends { id?: number | string }>({
  data,
  columns,
  height = 600,
  rowHeight = 48,
  initialSort = [],
}: VirtualTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>(initialSort)
  const tableContainerRef = useRef<HTMLDivElement>(null)

  const table = useReactTable({
    data,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  const { rows } = table.getRowModel()

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => tableContainerRef.current,
    estimateSize: () => rowHeight,
    overscan: 10,
  })

  return (
    <div ref={tableContainerRef} className="overflow-auto rounded-xl" style={{ height, border: '1px solid rgba(255,255,255,0.06)' }}>
      <table className="w-full">
        <thead>
          {table.getHeaderGroups().map(headerGroup => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map(header => (
                <th
                  key={header.id}
                  onClick={header.column.getToggleSortingHandler()}
                  className="text-left text-xs font-semibold px-4 py-3 cursor-pointer select-none sticky top-0 z-10"
                  style={{ background: '#141A2D', borderBottom: '1px solid rgba(255,255,255,0.06)', color: '#64748B' }}
                >
                  {flexRender(header.column.columnDef.header, header.getContext())}
                  {{ asc: ' ▲', desc: ' ▼' }[header.column.getIsSorted() as string] ?? ''}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            position: 'relative',
          }}
        >
          {virtualizer.getVirtualItems().map(virtualRow => {
            const row = rows[virtualRow.index]
            return (
              <tr
                key={row.id}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  height: `${rowHeight}px`,
                  transform: `translateY(${virtualRow.start}px)`,
                }}
                className="hover:opacity-80 transition-opacity"
              >
                {row.getVisibleCells().map(cell => (
                  <td key={cell.id} className="px-4 py-3 text-sm" style={{ borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

