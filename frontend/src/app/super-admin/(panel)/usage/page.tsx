'use client';

import { useEffect, useMemo, useState, useCallback } from 'react';
import {
  BarChart3,
  Loader2,
  RefreshCw,
  AlertTriangle,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Building2,
} from 'lucide-react';
import { cn } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface UsageRow {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  tier: string;
  status: string;
  created_at?: string;
  audit_log_count: number;
  audit_log_bytes_estimate: number;
}

interface UsageApiResponse {
  data?: UsageRow[];
}

interface PromVectorResult {
  metric: Record<string, string>;
  value: [number, string];
}

interface PromResponse {
  status: string;
  data?: {
    resultType: string;
    result: PromVectorResult[];
  };
}

interface MergedRow {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  tier: string;
  status: string;
  requests: number;
  bytes_in: number;
  bytes_out: number;
  audit_log_count: number;
  audit_log_bytes_estimate: number;
}

type SortKey =
  | 'tenant_name'
  | 'tier'
  | 'status'
  | 'requests'
  | 'bytes_in'
  | 'bytes_out'
  | 'audit_log_count'
  | 'audit_log_bytes_estimate';

type SortDir = 'asc' | 'desc';

// ---------------------------------------------------------------------------
// Styling helpers
// ---------------------------------------------------------------------------

const tierColor: Record<string, string> = {
  free: 'bg-gray-100 text-gray-800',
  starter: 'bg-blue-100 text-blue-800',
  professional: 'bg-indigo-100 text-indigo-800',
  enterprise: 'bg-purple-100 text-purple-800',
};

const statusColorMap: Record<string, string> = {
  active: 'bg-green-100 text-green-800',
  pending: 'bg-yellow-100 text-yellow-800',
  suspended: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-100 text-gray-700',
  provisioning: 'bg-blue-100 text-blue-800',
};

function formatBytes(bytes: number): string {
  if (!bytes || bytes < 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB', 'PB'];
  let val = bytes / 1024;
  let i = 0;
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024;
    i++;
  }
  return `${val.toFixed(val >= 100 ? 0 : val >= 10 ? 1 : 2)} ${units[i]}`;
}

function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

// ---------------------------------------------------------------------------
// Fetch helpers
// ---------------------------------------------------------------------------

async function fetchPromQuery(query: string, signal?: AbortSignal): Promise<PromVectorResult[]> {
  const url = `/proxy/prometheus/api/v1/query?query=${encodeURIComponent(query)}`;
  const res = await fetch(url, { signal });
  if (!res.ok) {
    throw new Error(`Prometheus query failed (${res.status})`);
  }
  const body = (await res.json()) as PromResponse;
  if (body.status !== 'success' || !body.data) {
    throw new Error(`Prometheus query unsuccessful: ${body.status}`);
  }
  return body.data.result ?? [];
}

async function fetchUsage(signal?: AbortSignal): Promise<UsageRow[]> {
  const res = await fetch('/proxy/tenant/api/v1/admin/usage', { signal });
  if (!res.ok) {
    throw new Error(`Tenant usage fetch failed (${res.status})`);
  }
  const body = (await res.json()) as UsageApiResponse;
  return body.data ?? [];
}

// Build a lookup from tenant identifier -> numeric value.
// Prometheus labels use tenant slug or id depending on the emitter; key by
// whatever the metric exposes and we match on either side when merging.
function indexByTenant(results: PromVectorResult[], extraLabel?: string): Record<string, number> {
  const out: Record<string, number> = {};
  for (const r of results) {
    const tenant = r.metric.tenant;
    if (!tenant) continue;
    const key = extraLabel ? `${tenant}::${r.metric[extraLabel] ?? ''}` : tenant;
    const n = Number(r.value?.[1]);
    if (Number.isFinite(n)) out[key] = (out[key] ?? 0) + n;
  }
  return out;
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function SuperAdminUsagePage() {
  const [rows, setRows] = useState<MergedRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>('requests');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  const load = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError(null);
    try {
      const [usage, requests, bytes] = await Promise.all([
        fetchUsage(signal),
        fetchPromQuery('sum by (tenant) (tenant_http_requests_total)', signal),
        fetchPromQuery('sum by (tenant, direction) (tenant_http_bytes_total)', signal),
      ]);

      const requestsByTenant = indexByTenant(requests);
      // For bytes, the key is `${tenant}::${direction}`.
      const bytesByTenantDir = indexByTenant(bytes, 'direction');

      const merged: MergedRow[] = usage.map((u) => {
        // Try matching by id first, then slug — different services may label
        // metrics either way.
        const promKey =
          requestsByTenant[u.tenant_id] !== undefined
            ? u.tenant_id
            : requestsByTenant[u.tenant_slug] !== undefined
              ? u.tenant_slug
              : u.tenant_id;

        const reqs = requestsByTenant[promKey] ?? requestsByTenant[u.tenant_slug] ?? 0;
        const bIn =
          bytesByTenantDir[`${promKey}::in`] ?? bytesByTenantDir[`${u.tenant_slug}::in`] ?? 0;
        const bOut =
          bytesByTenantDir[`${promKey}::out`] ?? bytesByTenantDir[`${u.tenant_slug}::out`] ?? 0;

        return {
          tenant_id: u.tenant_id,
          tenant_name: u.tenant_name,
          tenant_slug: u.tenant_slug,
          tier: u.tier,
          status: u.status,
          requests: reqs,
          bytes_in: bIn,
          bytes_out: bOut,
          audit_log_count: u.audit_log_count ?? 0,
          audit_log_bytes_estimate: u.audit_log_bytes_estimate ?? 0,
        };
      });

      setRows(merged);
    } catch (e) {
      if ((e as Error).name === 'AbortError') return;
      setError((e as Error).message ?? 'Failed to load usage data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    load(ctrl.signal);
    return () => ctrl.abort();
  }, [load]);

  const sorted = useMemo(() => {
    const copy = [...rows];
    copy.sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      let cmp = 0;
      if (typeof av === 'number' && typeof bv === 'number') {
        cmp = av - bv;
      } else {
        cmp = String(av).localeCompare(String(bv));
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return copy;
  }, [rows, sortKey, sortDir]);

  const totals = useMemo(() => {
    return rows.reduce(
      (acc, r) => {
        acc.requests += r.requests;
        acc.bytes_in += r.bytes_in;
        acc.bytes_out += r.bytes_out;
        acc.audit_log_count += r.audit_log_count;
        acc.audit_log_bytes_estimate += r.audit_log_bytes_estimate;
        return acc;
      },
      {
        requests: 0,
        bytes_in: 0,
        bytes_out: 0,
        audit_log_count: 0,
        audit_log_bytes_estimate: 0,
      },
    );
  }, [rows]);

  function toggleSort(k: SortKey) {
    if (sortKey === k) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(k);
      setSortDir(k === 'tenant_name' || k === 'tier' || k === 'status' ? 'asc' : 'desc');
    }
  }

  function SortIcon({ k }: { k: SortKey }) {
    if (sortKey !== k) return <ArrowUpDown className="h-3 w-3 opacity-30" />;
    return sortDir === 'asc' ? (
      <ArrowUp className="h-3 w-3" />
    ) : (
      <ArrowDown className="h-3 w-3" />
    );
  }

  function HeaderCell({
    label,
    k,
    align = 'left',
  }: {
    label: string;
    k: SortKey;
    align?: 'left' | 'right';
  }) {
    return (
      <th
        className={cn(
          'cursor-pointer select-none px-6 py-3 font-medium hover:text-gray-700',
          align === 'right' ? 'text-right' : 'text-left',
        )}
        onClick={() => toggleSort(k)}
      >
        <span
          className={cn(
            'inline-flex items-center gap-1.5',
            align === 'right' ? 'justify-end' : '',
          )}
        >
          {label}
          <SortIcon k={k} />
        </span>
      </th>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenant Usage</h1>
          <p className="mt-1 text-sm text-gray-500">
            HTTP traffic and audit log footprint per tenant. Metrics are sampled from Prometheus
            and the tenant service.
          </p>
        </div>
        <button
          onClick={() => load()}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-50"
        >
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </button>
      </div>

      {/* Totals */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
        {[
          { label: 'Tenants', value: formatNumber(rows.length), icon: Building2 },
          { label: 'Total Requests', value: formatNumber(totals.requests), icon: BarChart3 },
          { label: 'Bytes In', value: formatBytes(totals.bytes_in), icon: ArrowDown },
          { label: 'Bytes Out', value: formatBytes(totals.bytes_out), icon: ArrowUp },
          {
            label: 'Audit Storage',
            value: formatBytes(totals.audit_log_bytes_estimate),
            icon: BarChart3,
          },
        ].map((s) => {
          const Icon = s.icon;
          return (
            <div
              key={s.label}
              className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-gray-500">{s.label}</span>
                <span className="rounded-lg bg-indigo-50 p-1.5 text-indigo-600">
                  <Icon className="h-4 w-4" />
                </span>
              </div>
              <p className="mt-2 text-xl font-bold text-gray-900">{s.value}</p>
            </div>
          );
        })}
      </div>

      {/* Error */}
      {error && (
        <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-4">
          <AlertTriangle className="mt-0.5 h-4 w-4 text-red-600" />
          <div className="flex-1">
            <p className="text-sm font-medium text-red-800">Couldn&apos;t load usage data</p>
            <p className="mt-0.5 text-sm text-red-700">{error}</p>
          </div>
          <button
            onClick={() => load()}
            className="rounded-lg border border-red-200 bg-white px-3 py-1 text-xs font-medium text-red-700 hover:bg-red-50"
          >
            Retry
          </button>
        </div>
      )}

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        {loading && rows.length === 0 ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
          </div>
        ) : !loading && rows.length === 0 ? (
          <div className="py-16 text-center">
            <BarChart3 className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-2 text-sm text-gray-400">No usage data available.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-sm text-gray-500">
                  <HeaderCell label="Tenant" k="tenant_name" />
                  <HeaderCell label="Tier" k="tier" />
                  <HeaderCell label="Status" k="status" />
                  <HeaderCell label="Requests" k="requests" align="right" />
                  <HeaderCell label="Bytes In" k="bytes_in" align="right" />
                  <HeaderCell label="Bytes Out" k="bytes_out" align="right" />
                  <HeaderCell label="Audit Logs" k="audit_log_count" align="right" />
                  <HeaderCell label="Audit Bytes" k="audit_log_bytes_estimate" align="right" />
                </tr>
              </thead>
              <tbody>
                {/* Totals row */}
                <tr className="border-b border-gray-100 bg-gray-50 text-sm font-semibold text-gray-700">
                  <td className="px-6 py-3" colSpan={3}>
                    Totals across {rows.length} tenant{rows.length === 1 ? '' : 's'}
                  </td>
                  <td className="px-6 py-3 text-right">{formatNumber(totals.requests)}</td>
                  <td className="px-6 py-3 text-right">{formatBytes(totals.bytes_in)}</td>
                  <td className="px-6 py-3 text-right">{formatBytes(totals.bytes_out)}</td>
                  <td className="px-6 py-3 text-right">{formatNumber(totals.audit_log_count)}</td>
                  <td className="px-6 py-3 text-right">
                    {formatBytes(totals.audit_log_bytes_estimate)}
                  </td>
                </tr>

                {sorted.map((r) => (
                  <tr
                    key={r.tenant_id}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 text-sm font-bold text-indigo-600">
                          {r.tenant_name.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <div className="text-sm font-medium text-gray-900">{r.tenant_name}</div>
                          <div className="text-xs text-gray-500">{r.tenant_slug}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          tierColor[r.tier] ?? 'bg-gray-100 text-gray-800',
                        )}
                      >
                        {r.tier}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusColorMap[r.status] ?? 'bg-gray-100 text-gray-700',
                        )}
                      >
                        {r.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right text-sm tabular-nums text-gray-900">
                      {formatNumber(r.requests)}
                    </td>
                    <td className="px-6 py-4 text-right text-sm tabular-nums text-gray-700">
                      {formatBytes(r.bytes_in)}
                    </td>
                    <td className="px-6 py-4 text-right text-sm tabular-nums text-gray-700">
                      {formatBytes(r.bytes_out)}
                    </td>
                    <td className="px-6 py-4 text-right text-sm tabular-nums text-gray-700">
                      {formatNumber(r.audit_log_count)}
                    </td>
                    <td className="px-6 py-4 text-right text-sm tabular-nums text-gray-700">
                      {formatBytes(r.audit_log_bytes_estimate)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
