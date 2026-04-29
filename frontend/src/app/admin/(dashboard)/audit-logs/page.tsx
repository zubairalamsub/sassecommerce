'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  ScrollText, Loader2, Search, ChevronLeft, ChevronRight,
  Filter, Clock, User, Shield, AlertCircle, X, Download,
  RefreshCw, Eye, Globe, Monitor, FileText, ArrowRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { auditApi, type AuditLog } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const ACTIONS = ['CREATE', 'UPDATE', 'DELETE', 'READ', 'LOGIN', 'LOGOUT', 'EXPORT', 'IMPORT'] as const;
const RESOURCES = ['tenant', 'tenant_config', 'user', 'product', 'order'] as const;

function actionBadge(action: string) {
  switch (action) {
    case 'CREATE': return 'bg-green-100 text-green-800';
    case 'UPDATE': return 'bg-blue-100 text-blue-800';
    case 'DELETE': return 'bg-red-100 text-red-800';
    case 'READ': return 'bg-gray-100 text-gray-600';
    case 'LOGIN': return 'bg-violet-100 text-violet-800';
    case 'LOGOUT': return 'bg-orange-100 text-orange-800';
    case 'EXPORT': return 'bg-teal-100 text-teal-800';
    case 'IMPORT': return 'bg-indigo-100 text-indigo-800';
    default: return 'bg-gray-100 text-gray-600';
  }
}

function methodBadge(method: string) {
  switch (method) {
    case 'POST': return 'text-green-600';
    case 'PUT': case 'PATCH': return 'text-blue-600';
    case 'DELETE': return 'text-red-600';
    default: return 'text-gray-500';
  }
}

function statusBadge(code: number) {
  if (code >= 200 && code < 300) return 'bg-green-100 text-green-700';
  if (code >= 400 && code < 500) return 'bg-yellow-100 text-yellow-700';
  if (code >= 500) return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-600';
}

function tryParseJSON(str: string | undefined): object | null {
  if (!str) return null;
  try {
    const parsed = JSON.parse(str);
    return typeof parsed === 'object' ? parsed : null;
  } catch {
    return null;
  }
}

function JSONBlock({ label, value, accent }: { label: string; value: string | undefined; accent?: string }) {
  const parsed = tryParseJSON(value);
  if (!parsed && !value) return null;
  return (
    <div>
      <p className={cn('text-xs font-semibold mb-1.5', accent || 'text-gray-500')}>{label}</p>
      <pre className="rounded-lg bg-gray-900 text-gray-100 p-3 text-[11px] leading-relaxed overflow-x-auto max-h-64 scrollbar-thin">
        {parsed ? JSON.stringify(parsed, null, 2) : value}
      </pre>
    </div>
  );
}

// ---------- Detail Panel ----------
function DetailPanel({ log, onClose }: { log: AuditLog; onClose: () => void }) {
  const ts = new Date(log.created_at);

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/20 backdrop-blur-sm" onClick={onClose} />

      {/* Panel */}
      <div className="relative w-full max-w-xl bg-white shadow-2xl border-l border-gray-200 flex flex-col animate-in slide-in-from-right duration-200">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className={cn('inline-flex rounded-full px-2.5 py-1 text-xs font-semibold', actionBadge(log.action))}>
              {log.action}
            </span>
            <span className="text-sm font-medium text-gray-700">{log.resource}</span>
            <span className={cn('inline-flex rounded-full px-2 py-0.5 text-[10px] font-semibold', statusBadge(log.response_code))}>
              {log.response_code}
            </span>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Summary grid */}
          <div className="grid grid-cols-2 gap-4">
            <DetailField icon={<Clock className="h-3.5 w-3.5" />} label="Timestamp"
              value={ts.toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
            />
            <DetailField icon={<Monitor className="h-3.5 w-3.5" />} label="Method">
              <span className={cn('font-mono font-bold text-sm', methodBadge(log.method))}>{log.method}</span>
            </DetailField>
            <DetailField icon={<User className="h-3.5 w-3.5" />} label="User ID"
              value={log.user_id || '—'} mono
            />
            <DetailField icon={<FileText className="h-3.5 w-3.5" />} label="Resource ID"
              value={log.resource_id || '—'} mono
            />
            <DetailField icon={<Globe className="h-3.5 w-3.5" />} label="IP Address"
              value={log.ip_address || '—'} mono
            />
            <DetailField icon={<Clock className="h-3.5 w-3.5" />} label="Duration"
              value={`${log.duration_ms}ms`}
            />
          </div>

          {/* Full path */}
          <div>
            <p className="text-xs font-semibold text-gray-500 mb-1.5">Path</p>
            <p className="rounded-lg bg-gray-50 border border-gray-200 px-3 py-2 text-xs font-mono text-gray-700 break-all">
              {log.path}
            </p>
          </div>

          {/* Error */}
          {log.error_message && (
            <div>
              <p className="text-xs font-semibold text-red-500 mb-1.5">Error Message</p>
              <p className="rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">
                {log.error_message}
              </p>
            </div>
          )}

          {/* Old / New value diff */}
          {(log.old_value || log.new_value) && (
            <div>
              <p className="text-xs font-semibold text-gray-500 mb-2">Changes</p>
              <div className="grid grid-cols-1 gap-3">
                {log.old_value && (
                  <JSONBlock label="Before" value={log.old_value} accent="text-red-500" />
                )}
                {log.old_value && log.new_value && (
                  <div className="flex justify-center">
                    <ArrowRight className="h-4 w-4 text-gray-400 rotate-90" />
                  </div>
                )}
                {log.new_value && (
                  <JSONBlock label="After" value={log.new_value} accent="text-green-500" />
                )}
              </div>
            </div>
          )}

          {/* Request body */}
          <JSONBlock label="Request Body" value={log.request_body} />

          {/* Metadata */}
          <JSONBlock label="Metadata" value={log.metadata} />

          {/* User agent */}
          {log.user_agent && (
            <div>
              <p className="text-xs font-semibold text-gray-500 mb-1.5">User Agent</p>
              <p className="rounded-lg bg-gray-50 border border-gray-200 px-3 py-2 text-[10px] font-mono text-gray-500 break-all leading-relaxed">
                {log.user_agent}
              </p>
            </div>
          )}

          {/* IDs */}
          <div className="border-t border-gray-100 pt-4">
            <p className="text-xs font-semibold text-gray-500 mb-2">Identifiers</p>
            <div className="space-y-1.5">
              <IDRow label="Log ID" value={log.id} />
              <IDRow label="Tenant ID" value={log.tenant_id} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function DetailField({ icon, label, value, mono, children }: {
  icon: React.ReactNode; label: string; value?: string; mono?: boolean; children?: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-2">
      <span className="mt-0.5 text-gray-400">{icon}</span>
      <div>
        <p className="text-[10px] uppercase tracking-wider text-gray-400 mb-0.5">{label}</p>
        {children || <p className={cn('text-sm text-gray-800', mono && 'font-mono text-xs')}>{value}</p>}
      </div>
    </div>
  );
}

function IDRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between text-xs">
      <span className="text-gray-400">{label}</span>
      <span className="font-mono text-gray-600 select-all">{value || '—'}</span>
    </div>
  );
}

// ---------- CSV Export ----------
function exportCSV(logs: AuditLog[]) {
  const headers = ['Timestamp', 'Action', 'Resource', 'Resource ID', 'Method', 'Path', 'Status', 'Duration (ms)', 'User ID', 'IP Address', 'Error'];
  const rows = logs.map((l) => [
    new Date(l.created_at).toISOString(),
    l.action,
    l.resource,
    l.resource_id,
    l.method,
    l.path,
    String(l.response_code),
    String(l.duration_ms),
    l.user_id,
    l.ip_address,
    l.error_message || '',
  ]);
  const csv = [headers, ...rows].map((r) => r.map((c) => `"${c.replace(/"/g, '""')}"`).join(',')).join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

// ---------- Main Page ----------
export default function AuditLogsPage() {
  const { tenantId, token } = useAuthStore();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);

  // Filters
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [userIdFilter, setUserIdFilter] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const pageSize = 25;

  // Auto-refresh
  const [autoRefresh, setAutoRefresh] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchLogs = useCallback(async () => {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await auditApi.list(tenantId, token, {
        action: actionFilter || undefined,
        resource: resourceFilter || undefined,
        resource_id: searchQuery || undefined,
        user_id: userIdFilter || undefined,
        start_date: startDate || undefined,
        end_date: endDate ? endDate + 'T23:59:59Z' : undefined,
        page,
        page_size: pageSize,
      });
      setLogs(res.data ?? []);
      setTotal(res.total);
      setTotalPages(res.total_pages);
    } catch {
      setLogs([]);
      setTotal(0);
      setTotalPages(0);
    } finally {
      setLoading(false);
    }
  }, [tenantId, token, actionFilter, resourceFilter, searchQuery, userIdFilter, startDate, endDate, page]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  // Auto-refresh effect
  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchLogs, 30_000);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchLogs]);

  function resetFilters() {
    setActionFilter('');
    setResourceFilter('');
    setSearchQuery('');
    setUserIdFilter('');
    setStartDate('');
    setEndDate('');
    setPage(1);
  }

  const hasFilters = actionFilter || resourceFilter || searchQuery || userIdFilter || startDate || endDate;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Audit Logs</h1>
          <p className="mt-1 text-sm text-gray-500">
            Track all actions and changes made across your store.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Auto-refresh toggle */}
          <button
            onClick={() => setAutoRefresh((v) => !v)}
            className={cn(
              'flex items-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-medium transition-colors',
              autoRefresh
                ? 'border-green-200 bg-green-50 text-green-700'
                : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50',
            )}
          >
            <RefreshCw className={cn('h-3.5 w-3.5', autoRefresh && 'animate-spin')} />
            {autoRefresh ? 'Live' : 'Auto-refresh'}
          </button>

          {/* Manual refresh */}
          <button
            onClick={fetchLogs}
            disabled={loading}
            className="rounded-lg border border-gray-200 bg-white p-2 text-gray-500 hover:bg-gray-50 transition-colors disabled:opacity-40"
            title="Refresh"
          >
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
          </button>

          {/* Export CSV */}
          <button
            onClick={() => exportCSV(logs)}
            disabled={logs.length === 0}
            className="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-40"
          >
            <Download className="h-3.5 w-3.5" />
            Export CSV
          </button>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <span className="rounded-lg bg-primary-light p-2.5"><ScrollText className="h-5 w-5 text-primary" /></span>
            <div>
              <p className="text-xs text-gray-500">Total Events</p>
              <p className="text-2xl font-bold text-gray-900">{total}</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <span className="rounded-lg bg-green-50 p-2.5"><Shield className="h-5 w-5 text-green-600" /></span>
            <div>
              <p className="text-xs text-gray-500">Creates</p>
              <p className="text-2xl font-bold text-green-600">{logs.filter((l) => l.action === 'CREATE').length}</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <span className="rounded-lg bg-blue-50 p-2.5"><User className="h-5 w-5 text-blue-600" /></span>
            <div>
              <p className="text-xs text-gray-500">Updates</p>
              <p className="text-2xl font-bold text-blue-600">{logs.filter((l) => l.action === 'UPDATE').length}</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <span className="rounded-lg bg-red-50 p-2.5"><AlertCircle className="h-5 w-5 text-red-600" /></span>
            <div>
              <p className="text-xs text-gray-500">Errors</p>
              <p className="text-2xl font-bold text-red-600">{logs.filter((l) => l.response_code >= 400).length}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-2 mb-3">
          <Filter className="h-4 w-4 text-gray-400" />
          <span className="text-sm font-medium text-gray-700">Filters</span>
          {hasFilters && (
            <button onClick={resetFilters} className="ml-auto text-xs text-primary hover:text-primary-dark font-medium">
              Clear all
            </button>
          )}
        </div>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <select
            value={actionFilter}
            onChange={(e) => { setActionFilter(e.target.value); setPage(1); }}
            className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
          >
            <option value="">All Actions</option>
            {ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
          </select>
          <select
            value={resourceFilter}
            onChange={(e) => { setResourceFilter(e.target.value); setPage(1); }}
            className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
          >
            <option value="">All Resources</option>
            {RESOURCES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); setPage(1); }}
              placeholder="Resource ID..."
              className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div className="relative">
            <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              value={userIdFilter}
              onChange={(e) => { setUserIdFilter(e.target.value); setPage(1); }}
              placeholder="User ID..."
              className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <input
            type="date"
            value={startDate}
            onChange={(e) => { setStartDate(e.target.value); setPage(1); }}
            className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
          />
          <input
            type="date"
            value={endDate}
            onChange={(e) => { setEndDate(e.target.value); setPage(1); }}
            className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
          />
        </div>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <ScrollText className="h-10 w-10 text-gray-300" />
            <p className="mt-3 text-sm font-medium text-gray-700">No audit logs found</p>
            <p className="mt-1 text-xs text-gray-400">
              {hasFilters ? 'Try adjusting your filters' : 'Audit events will appear here as actions are performed'}
            </p>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-100 text-left text-xs text-gray-500">
                    <th className="px-4 py-3 font-medium">Timestamp</th>
                    <th className="px-4 py-3 font-medium">Action</th>
                    <th className="px-4 py-3 font-medium">Resource</th>
                    <th className="px-4 py-3 font-medium">Method</th>
                    <th className="px-4 py-3 font-medium">Path</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Duration</th>
                    <th className="px-4 py-3 font-medium">User</th>
                    <th className="px-4 py-3 font-medium w-10"></th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => (
                    <tr
                      key={log.id}
                      onClick={() => setSelectedLog(log)}
                      className={cn(
                        'border-b border-gray-50 transition-colors cursor-pointer',
                        log.response_code >= 400 ? 'bg-red-50/30 hover:bg-red-50/60' : 'hover:bg-gray-50',
                      )}
                    >
                      <td className="px-4 py-3 text-xs text-gray-500 whitespace-nowrap">
                        <div className="flex items-center gap-1.5">
                          <Clock className="h-3 w-3 text-gray-400" />
                          {new Date(log.created_at).toLocaleString('en-GB', {
                            day: '2-digit', month: 'short', year: 'numeric',
                            hour: '2-digit', minute: '2-digit', second: '2-digit',
                          })}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn('inline-flex rounded-full px-2 py-0.5 text-[10px] font-semibold', actionBadge(log.action))}>
                          {log.action}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-gray-700 font-medium">
                        {log.resource}
                        {log.resource_id && (
                          <span className="ml-1 text-gray-400 font-mono text-[10px]">
                            {log.resource_id.length > 12 ? log.resource_id.slice(0, 12) + '...' : log.resource_id}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn('text-xs font-mono font-semibold', methodBadge(log.method))}>
                          {log.method}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-gray-500 font-mono max-w-[200px] truncate" title={log.path}>
                        {log.path}
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn('inline-flex rounded-full px-2 py-0.5 text-[10px] font-semibold', statusBadge(log.response_code))}>
                          {log.response_code}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-gray-500 whitespace-nowrap">
                        {log.duration_ms}ms
                      </td>
                      <td className="px-4 py-3 text-xs text-gray-500 font-mono max-w-[100px] truncate" title={log.user_id}>
                        {log.user_id || '—'}
                      </td>
                      <td className="px-4 py-3">
                        <Eye className="h-3.5 w-3.5 text-gray-400" />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between border-t border-gray-100 px-4 py-3">
              <p className="text-xs text-gray-500">
                Showing {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} of {total} events
              </p>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="rounded-lg border border-gray-200 p-1.5 text-gray-500 hover:bg-gray-50 disabled:opacity-40"
                >
                  <ChevronLeft className="h-4 w-4" />
                </button>
                <span className="px-3 text-xs font-medium text-gray-700">
                  Page {page} of {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="rounded-lg border border-gray-200 p-1.5 text-gray-500 hover:bg-gray-50 disabled:opacity-40"
                >
                  <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Detail Panel */}
      {selectedLog && <DetailPanel log={selectedLog} onClose={() => setSelectedLog(null)} />}
    </div>
  );
}
