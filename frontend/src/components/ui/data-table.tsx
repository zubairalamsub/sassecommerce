'use client';

import { useState, useMemo, type ReactNode } from 'react';
import {
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
  Search,
  Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Column<T> {
  /** Unique key — also used as default accessor if `accessor` is not provided */
  key: string;
  /** Column header label */
  header: string;
  /** Extract the cell value from the row. Defaults to `row[key]`. */
  accessor?: (row: T) => unknown;
  /** Custom cell renderer */
  cell?: (row: T) => ReactNode;
  /** Enable sorting for this column */
  sortable?: boolean;
  /** Custom sort comparator. Receives the two raw values from `accessor`. */
  sortFn?: (a: unknown, b: unknown) => number;
  /** Header alignment */
  headerClassName?: string;
  /** Cell className */
  cellClassName?: string;
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  /** Unique key extractor for each row */
  rowKey: (row: T) => string;
  /** Show loading spinner */
  loading?: boolean;
  /** Custom loading text */
  loadingText?: string;
  /** Shown when data is empty and not loading */
  emptyIcon?: ReactNode;
  emptyTitle?: string;
  emptyDescription?: string;
  emptyAction?: ReactNode;
  /** Enable built-in search across all columns */
  searchable?: boolean;
  searchPlaceholder?: string;
  /** Items per page. Set to 0 to disable pagination. */
  pageSize?: number;
  /** Extra content in the toolbar (right side) */
  toolbar?: ReactNode;
  /** Row click handler */
  onRowClick?: (row: T) => void;
  /** Additional className for the container */
  className?: string;
}

type SortDir = 'asc' | 'desc';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function DataTable<T>({
  columns,
  data,
  rowKey,
  loading = false,
  loadingText = 'Loading...',
  emptyIcon,
  emptyTitle = 'No data found',
  emptyDescription,
  emptyAction,
  searchable = false,
  searchPlaceholder = 'Search...',
  pageSize = 10,
  toolbar,
  onRowClick,
  className,
}: DataTableProps<T>) {
  const [search, setSearch] = useState('');
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [page, setPage] = useState(1);

  // Reset page when search changes
  function handleSearch(value: string) {
    setSearch(value);
    setPage(1);
  }

  // --- accessor helper ---
  function getValue(row: T, col: Column<T>): unknown {
    if (col.accessor) return col.accessor(row);
    return (row as Record<string, unknown>)[col.key];
  }

  // --- filtered ---
  const filtered = useMemo(() => {
    if (!search.trim()) return data;
    const q = search.toLowerCase();
    return data.filter((row) =>
      columns.some((col) => {
        const val = getValue(row, col);
        if (val == null) return false;
        return String(val).toLowerCase().includes(q);
      }),
    );
  }, [data, search, columns]);

  // --- sorted ---
  const sorted = useMemo(() => {
    if (!sortKey) return filtered;
    const col = columns.find((c) => c.key === sortKey);
    if (!col) return filtered;

    return [...filtered].sort((a, b) => {
      const va = getValue(a, col);
      const vb = getValue(b, col);

      if (col.sortFn) {
        return sortDir === 'asc' ? col.sortFn(va, vb) : col.sortFn(vb, va);
      }

      // Default: string comparison, numbers compared numerically
      if (typeof va === 'number' && typeof vb === 'number') {
        return sortDir === 'asc' ? va - vb : vb - va;
      }
      const sa = String(va ?? '');
      const sb = String(vb ?? '');
      return sortDir === 'asc' ? sa.localeCompare(sb) : sb.localeCompare(sa);
    });
  }, [filtered, sortKey, sortDir, columns]);

  // --- paginated ---
  const paginated = pageSize > 0 ? sorted.slice((page - 1) * pageSize, page * pageSize) : sorted;
  const totalPages = pageSize > 0 ? Math.max(1, Math.ceil(sorted.length / pageSize)) : 1;

  function toggleSort(key: string) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
    setPage(1);
  }

  // --- render ---
  const colSpan = columns.length;

  return (
    <div className={cn('space-y-4', className)}>
      {/* Toolbar */}
      {(searchable || toolbar) && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          {searchable && (
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
              <input
                type="text"
                value={search}
                onChange={(e) => handleSearch(e.target.value)}
                placeholder={searchPlaceholder}
                className="w-full rounded-lg border border-border bg-surface py-2 pl-9 pr-3 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          )}
          {toolbar && <div className="flex items-center gap-2">{toolbar}</div>}
        </div>
      )}

      {/* Table */}
      <div className="rounded-2xl border border-border bg-surface shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                {columns.map((col) => (
                  <th
                    key={col.key}
                    className={cn(
                      'px-6 py-3 font-medium',
                      col.sortable && 'cursor-pointer select-none',
                      col.headerClassName,
                    )}
                    onClick={col.sortable ? () => toggleSort(col.key) : undefined}
                  >
                    <span className="inline-flex items-center gap-1">
                      {col.header}
                      {col.sortable && (
                        <span className="text-text-muted">
                          {sortKey === col.key ? (
                            sortDir === 'asc' ? (
                              <ChevronUp className="h-3.5 w-3.5" />
                            ) : (
                              <ChevronDown className="h-3.5 w-3.5" />
                            )
                          ) : (
                            <ChevronsUpDown className="h-3.5 w-3.5" />
                          )}
                        </span>
                      )}
                    </span>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={colSpan} className="px-6 py-16 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
                    <p className="mt-2 text-sm text-text-secondary">{loadingText}</p>
                  </td>
                </tr>
              ) : paginated.length === 0 ? (
                <tr>
                  <td colSpan={colSpan} className="px-6 py-16 text-center">
                    {emptyIcon && <div className="mb-3 flex justify-center">{emptyIcon}</div>}
                    <p className="text-sm font-medium text-text">{emptyTitle}</p>
                    {emptyDescription && (
                      <p className="mt-1 text-sm text-text-muted">{emptyDescription}</p>
                    )}
                    {emptyAction && <div className="mt-4">{emptyAction}</div>}
                  </td>
                </tr>
              ) : (
                paginated.map((row) => (
                  <tr
                    key={rowKey(row)}
                    className={cn(
                      'border-b border-border transition-colors last:border-b-0 hover:bg-surface-hover',
                      onRowClick && 'cursor-pointer',
                    )}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                  >
                    {columns.map((col) => (
                      <td
                        key={col.key}
                        className={cn('px-6 py-4 text-sm text-text', col.cellClassName)}
                      >
                        {col.cell ? col.cell(row) : String(getValue(row, col) ?? '')}
                      </td>
                    ))}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {pageSize > 0 && !loading && sorted.length > 0 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-3">
            <p className="text-xs text-text-muted">
              Showing {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, sorted.length)} of{' '}
              {sorted.length}
              {search && ` (filtered from ${data.length})`}
            </p>
            <div className="flex items-center gap-1">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="rounded-lg p-1.5 text-text-secondary transition-colors hover:bg-surface-hover disabled:opacity-30"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1)
                .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                .reduce<(number | 'gap')[]>((acc, p, idx, arr) => {
                  if (idx > 0 && p - (arr[idx - 1] as number) > 1) acc.push('gap');
                  acc.push(p);
                  return acc;
                }, [])
                .map((item, idx) =>
                  item === 'gap' ? (
                    <span key={`gap-${idx}`} className="px-1 text-xs text-text-muted">
                      ...
                    </span>
                  ) : (
                    <button
                      key={item}
                      onClick={() => setPage(item)}
                      className={cn(
                        'min-w-[32px] rounded-lg px-2 py-1 text-xs font-medium transition-colors',
                        page === item
                          ? 'bg-primary text-white'
                          : 'text-text-secondary hover:bg-surface-hover',
                      )}
                    >
                      {item}
                    </button>
                  ),
                )}
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="rounded-lg p-1.5 text-text-secondary transition-colors hover:bg-surface-hover disabled:opacity-30"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
