'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  ScrollText, Loader2, Search, ChevronLeft, ChevronRight,
  Filter, Clock, User, Shield, AlertCircle,
} from 'lucide-react';
import { cn, formatDate } from '@/lib/utils';
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

export default function AuditLogsPage() {
  const { tenantId, token } = useAuthStore();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Filters
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const pageSize = 25;

  const fetchLogs = useCallback(async () => {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await auditApi.list(tenantId, token, {
        action: actionFilter || undefined,
        resource: resourceFilter || undefined,
        resource_id: searchQuery || undefined,
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
  }, [tenantId, token, actionFilter, resourceFilter, searchQuery, startDate, endDate, page]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  function resetFilters() {
    setActionFilter('');
    setResourceFilter('');
    setSearchQuery('');
    setStartDate('');
    setEndDate('');
    setPage(1);
  }

  const hasFilters = actionFilter || resourceFilter || searchQuery || startDate || endDate;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Audit Logs</h1>
        <p className="mt-1 text-sm text-gray-500">
          Track all actions and changes made across your store.
        </p>
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
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
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
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => (
                    <>
                      <tr
                        key={log.id}
                        onClick={() => setExpandedId(expandedId === log.id ? null : log.id)}
                        className={cn(
                          'border-b border-gray-50 transition-colors cursor-pointer',
                          log.response_code >= 400 ? 'bg-red-50/30' : 'hover:bg-gray-50',
                          expandedId === log.id && 'bg-blue-50/50',
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
                      </tr>
                      {expandedId === log.id && (
                        <tr key={`${log.id}-detail`} className="bg-gray-50/80">
                          <td colSpan={8} className="px-4 py-4">
                            <div className="grid grid-cols-2 gap-4 text-xs lg:grid-cols-4">
                              <div>
                                <p className="font-medium text-gray-500 mb-1">Full Path</p>
                                <p className="font-mono text-gray-700 break-all">{log.path}</p>
                              </div>
                              <div>
                                <p className="font-medium text-gray-500 mb-1">IP Address</p>
                                <p className="font-mono text-gray-700">{log.ip_address || '—'}</p>
                              </div>
                              <div>
                                <p className="font-medium text-gray-500 mb-1">User ID</p>
                                <p className="font-mono text-gray-700 break-all">{log.user_id || '—'}</p>
                              </div>
                              <div>
                                <p className="font-medium text-gray-500 mb-1">Resource ID</p>
                                <p className="font-mono text-gray-700 break-all">{log.resource_id || '—'}</p>
                              </div>
                              {log.error_message && (
                                <div className="col-span-2 lg:col-span-4">
                                  <p className="font-medium text-red-500 mb-1">Error</p>
                                  <p className="font-mono text-red-700 bg-red-50 rounded p-2">{log.error_message}</p>
                                </div>
                              )}
                              {log.user_agent && (
                                <div className="col-span-2 lg:col-span-4">
                                  <p className="font-medium text-gray-500 mb-1">User Agent</p>
                                  <p className="font-mono text-gray-600 text-[10px] break-all">{log.user_agent}</p>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
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
    </div>
  );
}
