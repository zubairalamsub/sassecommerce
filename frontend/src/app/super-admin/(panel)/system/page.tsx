'use client';

import { useState } from 'react';
import {
  Database,
  Download,
  Trash2,
  RotateCcw,
  Play,
  Save,
  AlertTriangle,
  AlertCircle,
  Info,
  X,
} from 'lucide-react';
import { cn, formatDate, formatDateTime } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Tab = 'health' | 'backups' | 'security' | 'disputes';

interface ServiceInfo {
  name: string;
  status: 'online' | 'offline';
  responseTime: number;
  uptime: number;
}

interface InfraItem {
  name: string;
  status: string;
  details: { label: string; value: string }[];
}

interface BackupEntry {
  id: string;
  type: 'full' | 'database' | 'config';
  size: string;
  status: 'completed' | 'in_progress' | 'failed';
  createdAt: string;
  duration: string;
}

interface AuditEntry {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  ip: string;
  details: string;
}

interface SecurityAlert {
  id: string;
  message: string;
  severity: 'critical' | 'warning' | 'info';
  timestamp: string;
}

interface Dispute {
  id: string;
  type: 'refund' | 'product_quality' | 'delivery' | 'fraud';
  tenant: string;
  customer: string;
  customerEmail: string;
  orderRef: string;
  amount: number;
  status: 'open' | 'in_review' | 'resolved' | 'escalated';
  createdAt: string;
  priority: 'high' | 'medium' | 'low';
  description: string;
  timeline: { date: string; event: string }[];
}

// ---------------------------------------------------------------------------
// Demo data
// ---------------------------------------------------------------------------

const services: ServiceInfo[] = [
  { name: 'Order', status: 'online', responseTime: 125, uptime: 99.98 },
  { name: 'Tenant', status: 'online', responseTime: 89, uptime: 99.99 },
  { name: 'User', status: 'online', responseTime: 95, uptime: 99.97 },
  { name: 'Product', status: 'online', responseTime: 112, uptime: 99.99 },
  { name: 'Inventory', status: 'online', responseTime: 148, uptime: 99.95 },
  { name: 'Payment', status: 'online', responseTime: 210, uptime: 99.99 },
  { name: 'Shipping', status: 'online', responseTime: 185, uptime: 99.96 },
  { name: 'Notification', status: 'online', responseTime: 98, uptime: 99.98 },
  { name: 'Review', status: 'online', responseTime: 82, uptime: 99.99 },
  { name: 'Cart', status: 'online', responseTime: 105, uptime: 99.97 },
  { name: 'Search', status: 'online', responseTime: 145, uptime: 99.94 },
  { name: 'Promotion', status: 'online', responseTime: 132, uptime: 99.98 },
  { name: 'Vendor', status: 'online', responseTime: 118, uptime: 99.96 },
  { name: 'Analytics', status: 'online', responseTime: 245, uptime: 99.93 },
  { name: 'Recommendation', status: 'online', responseTime: 198, uptime: 99.91 },
  { name: 'Config', status: 'online', responseTime: 80, uptime: 99.99 },
];

const infrastructure: InfraItem[] = [
  {
    name: 'PostgreSQL',
    status: 'Connected',
    details: [
      { label: 'Databases', value: '5' },
      { label: 'Connections', value: '12' },
    ],
  },
  {
    name: 'MongoDB',
    status: 'Connected',
    details: [
      { label: 'Databases', value: '3' },
      { label: 'Connections', value: '8' },
    ],
  },
  {
    name: 'Redis',
    status: 'Connected',
    details: [
      { label: 'Memory Used', value: '45 MB' },
      { label: 'Keys', value: '156' },
    ],
  },
  {
    name: 'Kafka',
    status: 'Running',
    details: [
      { label: 'Topics', value: '12' },
      { label: 'Consumer Lag', value: '0' },
    ],
  },
  {
    name: 'Elasticsearch',
    status: 'Green',
    details: [
      { label: 'Nodes', value: '1' },
      { label: 'Indices', value: '2' },
    ],
  },
];

const systemMetrics = [
  { label: 'CPU Usage', value: 34, color: 'bg-indigo-500' },
  { label: 'Memory Usage', value: 62, color: 'bg-purple-500' },
  { label: 'Disk Usage', value: 41, color: 'bg-blue-500' },
  { label: 'Network I/O', value: 25, color: 'bg-green-500', display: '12.5 MB/s' },
];

const backupHistory: BackupEntry[] = [
  { id: 'BKP-001', type: 'full', size: '2.4 GB', status: 'completed', createdAt: '2026-04-25T02:00:00Z', duration: '12m 34s' },
  { id: 'BKP-002', type: 'database', size: '1.8 GB', status: 'completed', createdAt: '2026-04-24T02:00:00Z', duration: '8m 12s' },
  { id: 'BKP-003', type: 'full', size: '2.5 GB', status: 'completed', createdAt: '2026-04-23T02:00:00Z', duration: '13m 05s' },
  { id: 'BKP-004', type: 'config', size: '156 MB', status: 'completed', createdAt: '2026-04-22T02:00:00Z', duration: '1m 22s' },
  { id: 'BKP-005', type: 'database', size: '1.7 GB', status: 'failed', createdAt: '2026-04-21T02:00:00Z', duration: '—' },
  { id: 'BKP-006', type: 'full', size: '2.3 GB', status: 'completed', createdAt: '2026-04-20T02:00:00Z', duration: '11m 48s' },
  { id: 'BKP-007', type: 'full', size: '—', status: 'in_progress', createdAt: '2026-04-25T14:30:00Z', duration: '—' },
  { id: 'BKP-008', type: 'database', size: '1.6 GB', status: 'completed', createdAt: '2026-04-19T02:00:00Z', duration: '7m 55s' },
  { id: 'BKP-009', type: 'config', size: '150 MB', status: 'completed', createdAt: '2026-04-18T02:00:00Z', duration: '1m 18s' },
  { id: 'BKP-010', type: 'full', size: '2.2 GB', status: 'completed', createdAt: '2026-04-17T02:00:00Z', duration: '11m 30s' },
];

const auditLog: AuditEntry[] = [
  { id: '1', timestamp: '2026-04-25T14:22:00Z', user: 'Rahim Uddin', action: 'login', ip: '103.48.16.22', details: 'Successful login from Dhaka' },
  { id: '2', timestamp: '2026-04-25T13:45:00Z', user: 'Fatima Akter', action: 'settings_changed', ip: '103.109.2.45', details: 'Updated rate limiting configuration' },
  { id: '3', timestamp: '2026-04-25T12:30:00Z', user: 'Kamal Hossain', action: 'tenant_created', ip: '103.230.108.15', details: 'Created tenant "Dhaka Electronics"' },
  { id: '4', timestamp: '2026-04-25T11:10:00Z', user: 'Nasreen Begum', action: 'backup_created', ip: '103.48.16.22', details: 'Full system backup initiated' },
  { id: '5', timestamp: '2026-04-25T10:05:00Z', user: 'Rahim Uddin', action: 'user_suspended', ip: '103.48.16.22', details: 'Suspended user for policy violation' },
  { id: '6', timestamp: '2026-04-24T22:15:00Z', user: 'Jamal Ahmed', action: 'login', ip: '103.109.2.88', details: 'Successful login from Chittagong' },
  { id: '7', timestamp: '2026-04-24T20:30:00Z', user: 'Fatima Akter', action: 'settings_changed', ip: '103.109.2.45', details: 'Updated CORS allowed origins' },
  { id: '8', timestamp: '2026-04-24T18:00:00Z', user: 'Mizanur Rahman', action: 'tenant_created', ip: '103.230.108.20', details: 'Created tenant "BD Fashion House"' },
  { id: '9', timestamp: '2026-04-24T15:45:00Z', user: 'Kamal Hossain', action: 'backup_created', ip: '103.230.108.15', details: 'Database-only backup initiated' },
  { id: '10', timestamp: '2026-04-24T14:20:00Z', user: 'Nasreen Begum', action: 'login', ip: '103.48.16.30', details: 'Successful login from Dhaka' },
];

const securityAlerts: SecurityAlert[] = [
  { id: '1', message: 'Multiple failed login attempts from IP 103.52.48.91', severity: 'critical', timestamp: '2026-04-25T13:55:00Z' },
  { id: '2', message: 'SSL certificate expires in 15 days', severity: 'warning', timestamp: '2026-04-25T06:00:00Z' },
  { id: '3', message: 'Unusual API usage spike detected from tenant "QuickMart"', severity: 'info', timestamp: '2026-04-24T22:30:00Z' },
];

const disputes: Dispute[] = [
  {
    id: '#DSP-001', type: 'refund', tenant: 'Dhaka Electronics', customer: 'Aminul Islam',
    customerEmail: 'aminul@example.com', orderRef: 'ORD-10234', amount: 4500,
    status: 'open', createdAt: '2026-04-24T10:30:00Z', priority: 'high',
    description: 'Customer claims product was defective upon arrival. Requesting full refund for a wireless mouse that stopped working after 2 hours.',
    timeline: [
      { date: '2026-04-24T10:30:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-24T11:00:00Z', event: 'Notification sent to tenant' },
      { date: '2026-04-24T14:15:00Z', event: 'Tenant responded: "Product was tested before shipping"' },
    ],
  },
  {
    id: '#DSP-002', type: 'delivery', tenant: 'BD Fashion House', customer: 'Rashida Khatun',
    customerEmail: 'rashida.k@example.com', orderRef: 'ORD-10198', amount: 2800,
    status: 'in_review', createdAt: '2026-04-23T09:00:00Z', priority: 'medium',
    description: 'Order marked as delivered but customer says they never received the package. Delivery photo shows wrong address.',
    timeline: [
      { date: '2026-04-23T09:00:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-23T09:30:00Z', event: 'Assigned to review team' },
      { date: '2026-04-23T14:00:00Z', event: 'Delivery proof requested from shipping partner' },
      { date: '2026-04-24T10:00:00Z', event: 'Shipping partner confirmed wrong delivery address' },
    ],
  },
  {
    id: '#DSP-003', type: 'product_quality', tenant: 'Chittagong Bazaar', customer: 'Shahin Alam',
    customerEmail: 'shahin.a@example.com', orderRef: 'ORD-10301', amount: 7200,
    status: 'escalated', createdAt: '2026-04-22T16:00:00Z', priority: 'high',
    description: 'Received a laptop bag that is a different color and material than what was shown in the product listing. Customer requesting replacement or refund.',
    timeline: [
      { date: '2026-04-22T16:00:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-22T16:30:00Z', event: 'Photo evidence submitted by customer' },
      { date: '2026-04-23T10:00:00Z', event: 'Tenant declined responsibility' },
      { date: '2026-04-23T15:00:00Z', event: 'Escalated to platform admin' },
    ],
  },
  {
    id: '#DSP-004', type: 'fraud', tenant: 'QuickMart BD', customer: 'Tarek Mahmud',
    customerEmail: 'tarek.m@example.com', orderRef: 'ORD-10288', amount: 15000,
    status: 'open', createdAt: '2026-04-23T11:20:00Z', priority: 'high',
    description: 'Customer reports unauthorized transaction on their account. Two orders placed from unfamiliar IP address with different shipping address.',
    timeline: [
      { date: '2026-04-23T11:20:00Z', event: 'Fraud report submitted by customer' },
      { date: '2026-04-23T11:45:00Z', event: 'Account temporarily frozen' },
      { date: '2026-04-23T14:00:00Z', event: 'Investigation initiated by security team' },
    ],
  },
  {
    id: '#DSP-005', type: 'refund', tenant: 'Sylhet Traders', customer: 'Nusrat Jahan',
    customerEmail: 'nusrat.j@example.com', orderRef: 'ORD-10315', amount: 3200,
    status: 'resolved', createdAt: '2026-04-20T08:00:00Z', priority: 'low',
    description: 'Customer returned item within return window. Refund not processed after 5 business days.',
    timeline: [
      { date: '2026-04-20T08:00:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-20T10:00:00Z', event: 'Refund processing confirmed delayed' },
      { date: '2026-04-21T09:00:00Z', event: 'Refund of BDT 3,200 processed' },
      { date: '2026-04-21T09:30:00Z', event: 'Dispute resolved - favor customer' },
    ],
  },
  {
    id: '#DSP-006', type: 'delivery', tenant: 'Dhaka Electronics', customer: 'Habibur Rahman',
    customerEmail: 'habib.r@example.com', orderRef: 'ORD-10278', amount: 1800,
    status: 'in_review', createdAt: '2026-04-22T13:00:00Z', priority: 'medium',
    description: 'Package arrived damaged. Outer box was crushed and product inside (phone case set) was broken.',
    timeline: [
      { date: '2026-04-22T13:00:00Z', event: 'Dispute opened with photos of damaged package' },
      { date: '2026-04-22T14:00:00Z', event: 'Forwarded to shipping partner for investigation' },
      { date: '2026-04-23T11:00:00Z', event: 'Shipping partner acknowledged handling issue' },
    ],
  },
  {
    id: '#DSP-007', type: 'product_quality', tenant: 'BD Fashion House', customer: 'Sumaiya Akhter',
    customerEmail: 'sumaiya.a@example.com', orderRef: 'ORD-10330', amount: 5500,
    status: 'open', createdAt: '2026-04-24T15:30:00Z', priority: 'medium',
    description: 'Saree received has significant color difference from product images. Customer requesting exchange for the correct shade.',
    timeline: [
      { date: '2026-04-24T15:30:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-24T16:00:00Z', event: 'Notification sent to tenant' },
    ],
  },
  {
    id: '#DSP-008', type: 'refund', tenant: 'Chittagong Bazaar', customer: 'Faruk Hossain',
    customerEmail: 'faruk.h@example.com', orderRef: 'ORD-10295', amount: 9800,
    status: 'resolved', createdAt: '2026-04-19T07:45:00Z', priority: 'low',
    description: 'Ordered laptop stand, received wrong model. Customer wants refund instead of replacement.',
    timeline: [
      { date: '2026-04-19T07:45:00Z', event: 'Dispute opened by customer' },
      { date: '2026-04-19T10:00:00Z', event: 'Tenant confirmed wrong item shipped' },
      { date: '2026-04-20T09:00:00Z', event: 'Return label generated' },
      { date: '2026-04-22T14:00:00Z', event: 'Item returned and refund processed' },
    ],
  },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const tabs: { key: Tab; label: string }[] = [
  { key: 'health', label: 'System Health' },
  { key: 'backups', label: 'Backups' },
  { key: 'security', label: 'Security' },
  { key: 'disputes', label: 'Disputes' },
];

const backupStatusBadge: Record<string, string> = {
  completed: 'bg-green-100 text-green-800',
  in_progress: 'bg-yellow-100 text-yellow-800',
  failed: 'bg-red-100 text-red-800',
};

const backupTypeBadge: Record<string, string> = {
  full: 'bg-indigo-100 text-indigo-800',
  database: 'bg-blue-100 text-blue-800',
  config: 'bg-gray-100 text-gray-800',
};

const priorityBadge: Record<string, string> = {
  high: 'bg-red-100 text-red-800',
  medium: 'bg-yellow-100 text-yellow-800',
  low: 'bg-green-100 text-green-800',
};

const disputeStatusBadge: Record<string, string> = {
  open: 'bg-yellow-100 text-yellow-800',
  in_review: 'bg-blue-100 text-blue-800',
  resolved: 'bg-green-100 text-green-800',
  escalated: 'bg-red-100 text-red-800',
};

const disputeTypeLabel: Record<string, string> = {
  refund: 'Refund',
  product_quality: 'Product Quality',
  delivery: 'Delivery',
  fraud: 'Fraud',
};

const severityBadge: Record<string, string> = {
  critical: 'bg-red-100 text-red-800',
  warning: 'bg-yellow-100 text-yellow-800',
  info: 'bg-blue-100 text-blue-800',
};

const severityIcon: Record<string, typeof AlertCircle> = {
  critical: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

const actionLabel: Record<string, string> = {
  login: 'Login',
  settings_changed: 'Settings Changed',
  tenant_created: 'Tenant Created',
  backup_created: 'Backup Created',
  user_suspended: 'User Suspended',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function SystemAdminPage() {
  const [activeTab, setActiveTab] = useState<Tab>('health');

  // Backup settings state
  const [autoBackup, setAutoBackup] = useState(true);
  const [backupFrequency, setBackupFrequency] = useState('daily');
  const [backupRetention, setBackupRetention] = useState('30');
  const [backupTime, setBackupTime] = useState('02:00');
  const [manualBackupType, setManualBackupType] = useState('full');

  // Security settings state
  const [jwtExpiry, setJwtExpiry] = useState('24');
  const [refreshExpiry, setRefreshExpiry] = useState('30');
  const [maxLoginAttempts, setMaxLoginAttempts] = useState('5');
  const [lockoutDuration, setLockoutDuration] = useState('30');
  const [require2FA, setRequire2FA] = useState(true);
  const [passwordMinLength, setPasswordMinLength] = useState('8');
  const [apiRateLimit, setApiRateLimit] = useState('100');
  const [loginRateLimit, setLoginRateLimit] = useState('10');
  const [registrationRateLimit, setRegistrationRateLimit] = useState('20');
  const [allowedOrigins, setAllowedOrigins] = useState('https://saajan.com\nhttps://admin.saajan.com\nhttps://api.saajan.com');
  const [forceHttps, setForceHttps] = useState(true);
  const [ipWhitelist, setIpWhitelist] = useState('103.48.16.0/24\n103.109.2.0/24');

  // Disputes state
  const [selectedDispute, setSelectedDispute] = useState<Dispute | null>(null);
  const [disputeStatusChange, setDisputeStatusChange] = useState('');
  const [disputeNote, setDisputeNote] = useState('');
  const [disputeResolution, setDisputeResolution] = useState('');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">System Administration</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage system health, backups, security settings, and customer disputes
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors',
                activeTab === tab.key
                  ? 'border-indigo-600 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
              )}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'health' && <HealthTab />}
      {activeTab === 'backups' && (
        <BackupsTab
          autoBackup={autoBackup}
          setAutoBackup={setAutoBackup}
          backupFrequency={backupFrequency}
          setBackupFrequency={setBackupFrequency}
          backupRetention={backupRetention}
          setBackupRetention={setBackupRetention}
          backupTime={backupTime}
          setBackupTime={setBackupTime}
          manualBackupType={manualBackupType}
          setManualBackupType={setManualBackupType}
        />
      )}
      {activeTab === 'security' && (
        <SecurityTab
          jwtExpiry={jwtExpiry} setJwtExpiry={setJwtExpiry}
          refreshExpiry={refreshExpiry} setRefreshExpiry={setRefreshExpiry}
          maxLoginAttempts={maxLoginAttempts} setMaxLoginAttempts={setMaxLoginAttempts}
          lockoutDuration={lockoutDuration} setLockoutDuration={setLockoutDuration}
          require2FA={require2FA} setRequire2FA={setRequire2FA}
          passwordMinLength={passwordMinLength} setPasswordMinLength={setPasswordMinLength}
          apiRateLimit={apiRateLimit} setApiRateLimit={setApiRateLimit}
          loginRateLimit={loginRateLimit} setLoginRateLimit={setLoginRateLimit}
          registrationRateLimit={registrationRateLimit} setRegistrationRateLimit={setRegistrationRateLimit}
          allowedOrigins={allowedOrigins} setAllowedOrigins={setAllowedOrigins}
          forceHttps={forceHttps} setForceHttps={setForceHttps}
          ipWhitelist={ipWhitelist} setIpWhitelist={setIpWhitelist}
        />
      )}
      {activeTab === 'disputes' && (
        <DisputesTab
          selectedDispute={selectedDispute}
          setSelectedDispute={setSelectedDispute}
          disputeStatusChange={disputeStatusChange}
          setDisputeStatusChange={setDisputeStatusChange}
          disputeNote={disputeNote}
          setDisputeNote={setDisputeNote}
          disputeResolution={disputeResolution}
          setDisputeResolution={setDisputeResolution}
        />
      )}
    </div>
  );
}

// ===========================================================================
// SYSTEM HEALTH TAB
// ===========================================================================

function HealthTab() {
  return (
    <div className="space-y-8">
      {/* Service Status Grid */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-gray-900">Service Status</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {services.map((svc) => (
            <div
              key={svc.name}
              className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-900">{svc.name}</span>
                <span className="flex items-center gap-1.5">
                  <span
                    className={cn(
                      'inline-block h-2 w-2 rounded-full',
                      svc.status === 'online' ? 'bg-green-500' : 'bg-red-500',
                    )}
                  />
                  <span
                    className={cn(
                      'text-xs font-medium capitalize',
                      svc.status === 'online' ? 'text-green-700' : 'text-red-700',
                    )}
                  >
                    {svc.status}
                  </span>
                </span>
              </div>
              <div className="mt-3 flex items-center justify-between text-xs text-gray-500">
                <span>{svc.responseTime}ms</span>
                <span>{svc.uptime}% uptime</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Infrastructure Status */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-gray-900">Infrastructure Status</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {infrastructure.map((item) => (
            <div
              key={item.name}
              className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
            >
              <div className="flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50">
                  <Database className="h-5 w-5 text-indigo-600" />
                </span>
                <div>
                  <h3 className="text-sm font-semibold text-gray-900">{item.name}</h3>
                  <span className="text-xs font-medium text-green-600">{item.status}</span>
                </div>
              </div>
              <div className="mt-4 space-y-2">
                {item.details.map((d) => (
                  <div key={d.label} className="flex items-center justify-between text-sm">
                    <span className="text-gray-500">{d.label}</span>
                    <span className="font-medium text-gray-900">{d.value}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* System Metrics */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-gray-900">System Metrics</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {systemMetrics.map((metric) => (
            <div
              key={metric.label}
              className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-500">{metric.label}</span>
                <span className="text-lg font-bold text-gray-900">
                  {metric.display ?? `${metric.value}%`}
                </span>
              </div>
              <div className="mt-3 h-2.5 w-full overflow-hidden rounded-full bg-gray-100">
                <div
                  className={cn('h-full rounded-full transition-all', metric.color)}
                  style={{ width: `${metric.value}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ===========================================================================
// BACKUPS TAB
// ===========================================================================

interface BackupsTabProps {
  autoBackup: boolean;
  setAutoBackup: (v: boolean) => void;
  backupFrequency: string;
  setBackupFrequency: (v: string) => void;
  backupRetention: string;
  setBackupRetention: (v: string) => void;
  backupTime: string;
  setBackupTime: (v: string) => void;
  manualBackupType: string;
  setManualBackupType: (v: string) => void;
}

function BackupsTab({
  autoBackup, setAutoBackup,
  backupFrequency, setBackupFrequency,
  backupRetention, setBackupRetention,
  backupTime, setBackupTime,
  manualBackupType, setManualBackupType,
}: BackupsTabProps) {
  return (
    <div className="space-y-8">
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Backup Schedule */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-5 text-lg font-semibold text-gray-900">Backup Schedule</h2>
          <div className="space-y-5">
            {/* Auto toggle */}
            <div className="flex items-center justify-between">
              <div>
                <span className="text-sm font-medium text-gray-900">Auto Backup</span>
                <p className="text-xs text-gray-500">Automatically create backups on schedule</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={autoBackup}
                onClick={() => setAutoBackup(!autoBackup)}
                className={cn(
                  'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                  autoBackup ? 'bg-indigo-600' : 'bg-gray-200',
                )}
              >
                <span
                  className={cn(
                    'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                    autoBackup ? 'translate-x-5' : 'translate-x-0',
                  )}
                />
              </button>
            </div>

            {/* Frequency */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Frequency</label>
              <select
                value={backupFrequency}
                onChange={(e) => setBackupFrequency(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              >
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
                <option value="monthly">Monthly</option>
              </select>
            </div>

            {/* Retention */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Retention Period</label>
              <select
                value={backupRetention}
                onChange={(e) => setBackupRetention(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              >
                <option value="7">7 days</option>
                <option value="14">14 days</option>
                <option value="30">30 days</option>
                <option value="90">90 days</option>
              </select>
            </div>

            {/* Time */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Backup Time</label>
              <input
                type="time"
                value={backupTime}
                onChange={(e) => setBackupTime(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>

            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Save className="h-4 w-4" />
              Save Schedule
            </button>
          </div>
        </div>

        {/* Manual Backup */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-5 text-lg font-semibold text-gray-900">Manual Backup</h2>
          <p className="mb-4 text-sm text-gray-500">
            Create an on-demand backup of the platform. This may take several minutes depending on the backup type.
          </p>
          <div className="space-y-5">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Backup Type</label>
              <select
                value={manualBackupType}
                onChange={(e) => setManualBackupType(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              >
                <option value="full">Full Backup</option>
                <option value="database">Database Only</option>
                <option value="config">Config Only</option>
              </select>
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Play className="h-4 w-4" />
              Create Backup Now
            </button>
          </div>
        </div>
      </div>

      {/* Backup History Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Backup History</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">Backup ID</th>
                <th className="px-6 py-3 font-medium">Type</th>
                <th className="px-6 py-3 font-medium">Size</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Created At</th>
                <th className="px-6 py-3 font-medium">Duration</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {backupHistory.map((backup) => (
                <tr
                  key={backup.id}
                  className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                >
                  <td className="px-6 py-4 text-sm font-medium text-gray-900">{backup.id}</td>
                  <td className="px-6 py-4">
                    <span
                      className={cn(
                        'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        backupTypeBadge[backup.type],
                      )}
                    >
                      {backup.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">{backup.size}</td>
                  <td className="px-6 py-4">
                    <span
                      className={cn(
                        'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        backupStatusBadge[backup.status],
                      )}
                    >
                      {backup.status.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {formatDateTime(backup.createdAt)}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">{backup.duration}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-1">
                      <button
                        className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                        title="Download"
                      >
                        <Download className="h-4 w-4" />
                      </button>
                      <button
                        className="rounded-lg p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600 transition-colors"
                        title="Restore"
                      >
                        <RotateCcw className="h-4 w-4" />
                      </button>
                      <button
                        className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors"
                        title="Delete"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ===========================================================================
// SECURITY TAB
// ===========================================================================

interface SecurityTabProps {
  jwtExpiry: string; setJwtExpiry: (v: string) => void;
  refreshExpiry: string; setRefreshExpiry: (v: string) => void;
  maxLoginAttempts: string; setMaxLoginAttempts: (v: string) => void;
  lockoutDuration: string; setLockoutDuration: (v: string) => void;
  require2FA: boolean; setRequire2FA: (v: boolean) => void;
  passwordMinLength: string; setPasswordMinLength: (v: string) => void;
  apiRateLimit: string; setApiRateLimit: (v: string) => void;
  loginRateLimit: string; setLoginRateLimit: (v: string) => void;
  registrationRateLimit: string; setRegistrationRateLimit: (v: string) => void;
  allowedOrigins: string; setAllowedOrigins: (v: string) => void;
  forceHttps: boolean; setForceHttps: (v: boolean) => void;
  ipWhitelist: string; setIpWhitelist: (v: string) => void;
}

function SecurityTab({
  jwtExpiry, setJwtExpiry,
  refreshExpiry, setRefreshExpiry,
  maxLoginAttempts, setMaxLoginAttempts,
  lockoutDuration, setLockoutDuration,
  require2FA, setRequire2FA,
  passwordMinLength, setPasswordMinLength,
  apiRateLimit, setApiRateLimit,
  loginRateLimit, setLoginRateLimit,
  registrationRateLimit, setRegistrationRateLimit,
  allowedOrigins, setAllowedOrigins,
  forceHttps, setForceHttps,
  ipWhitelist, setIpWhitelist,
}: SecurityTabProps) {
  return (
    <div className="space-y-8">
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Authentication Settings */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-5 text-lg font-semibold text-gray-900">Authentication Settings</h2>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">JWT Expiry (hours)</label>
                <input
                  type="number"
                  value={jwtExpiry}
                  onChange={(e) => setJwtExpiry(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Refresh Token Expiry (days)</label>
                <input
                  type="number"
                  value={refreshExpiry}
                  onChange={(e) => setRefreshExpiry(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Max Login Attempts</label>
                <input
                  type="number"
                  value={maxLoginAttempts}
                  onChange={(e) => setMaxLoginAttempts(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Lockout Duration (min)</label>
                <input
                  type="number"
                  value={lockoutDuration}
                  onChange={(e) => setLockoutDuration(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Password Minimum Length</label>
              <input
                type="number"
                value={passwordMinLength}
                onChange={(e) => setPasswordMinLength(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div className="flex items-center justify-between pt-1">
              <div>
                <span className="text-sm font-medium text-gray-900">Require 2FA for Admins</span>
                <p className="text-xs text-gray-500">Force two-factor authentication for all admin accounts</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={require2FA}
                onClick={() => setRequire2FA(!require2FA)}
                className={cn(
                  'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                  require2FA ? 'bg-indigo-600' : 'bg-gray-200',
                )}
              >
                <span
                  className={cn(
                    'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                    require2FA ? 'translate-x-5' : 'translate-x-0',
                  )}
                />
              </button>
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Save className="h-4 w-4" />
              Save Authentication Settings
            </button>
          </div>
        </div>

        {/* Rate Limiting */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-5 text-lg font-semibold text-gray-900">Rate Limiting</h2>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">API Rate Limit (requests/min)</label>
              <input
                type="number"
                value={apiRateLimit}
                onChange={(e) => setApiRateLimit(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Login Rate Limit (attempts/min)</label>
              <input
                type="number"
                value={loginRateLimit}
                onChange={(e) => setLoginRateLimit(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Registration Rate Limit (per hour)</label>
              <input
                type="number"
                value={registrationRateLimit}
                onChange={(e) => setRegistrationRateLimit(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Save className="h-4 w-4" />
              Save Rate Limits
            </button>
          </div>
        </div>
      </div>

      {/* CORS & Network */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-5 text-lg font-semibold text-gray-900">CORS &amp; Network</h2>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Allowed Origins</label>
              <textarea
                rows={4}
                value={allowedOrigins}
                onChange={(e) => setAllowedOrigins(e.target.value)}
                placeholder="One origin per line"
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
              <p className="mt-1 text-xs text-gray-400">One origin per line</p>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <span className="text-sm font-medium text-gray-900">Force HTTPS</span>
                <p className="text-xs text-gray-500">Redirect all HTTP traffic to HTTPS</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={forceHttps}
                onClick={() => setForceHttps(!forceHttps)}
                className={cn(
                  'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                  forceHttps ? 'bg-indigo-600' : 'bg-gray-200',
                )}
              >
                <span
                  className={cn(
                    'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                    forceHttps ? 'translate-x-5' : 'translate-x-0',
                  )}
                />
              </button>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">IP Whitelist</label>
            <textarea
              rows={4}
              value={ipWhitelist}
              onChange={(e) => setIpWhitelist(e.target.value)}
              placeholder="One IP/CIDR per line"
              className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
            <p className="mt-1 text-xs text-gray-400">One IP or CIDR range per line</p>
          </div>
        </div>
        <div className="mt-5">
          <button
            type="button"
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
          >
            <Save className="h-4 w-4" />
            Save Network Settings
          </button>
        </div>
      </div>

      {/* Security Alerts */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Security Alerts</h2>
        </div>
        <div className="divide-y divide-gray-100">
          {securityAlerts.map((alert) => {
            const SevIcon = severityIcon[alert.severity];
            return (
              <div key={alert.id} className="flex items-start gap-4 px-6 py-4">
                <span
                  className={cn(
                    'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
                    alert.severity === 'critical'
                      ? 'bg-red-50'
                      : alert.severity === 'warning'
                        ? 'bg-yellow-50'
                        : 'bg-blue-50',
                  )}
                >
                  <SevIcon
                    className={cn(
                      'h-4 w-4',
                      alert.severity === 'critical'
                        ? 'text-red-600'
                        : alert.severity === 'warning'
                          ? 'text-yellow-600'
                          : 'text-blue-600',
                    )}
                  />
                </span>
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-900">{alert.message}</p>
                  <p className="mt-0.5 text-xs text-gray-500">
                    {formatDateTime(alert.timestamp)}
                  </p>
                </div>
                <span
                  className={cn(
                    'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                    severityBadge[alert.severity],
                  )}
                >
                  {alert.severity}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {/* Audit Log */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Audit Log</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">Timestamp</th>
                <th className="px-6 py-3 font-medium">User</th>
                <th className="px-6 py-3 font-medium">Action</th>
                <th className="px-6 py-3 font-medium">IP Address</th>
                <th className="px-6 py-3 font-medium">Details</th>
              </tr>
            </thead>
            <tbody>
              {auditLog.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                >
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                    {formatDateTime(entry.timestamp)}
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-900">{entry.user}</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">
                      {actionLabel[entry.action] ?? entry.action}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm font-mono text-gray-500">{entry.ip}</td>
                  <td className="px-6 py-4 text-sm text-gray-500">{entry.details}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ===========================================================================
// DISPUTES TAB
// ===========================================================================

interface DisputesTabProps {
  selectedDispute: Dispute | null;
  setSelectedDispute: (v: Dispute | null) => void;
  disputeStatusChange: string;
  setDisputeStatusChange: (v: string) => void;
  disputeNote: string;
  setDisputeNote: (v: string) => void;
  disputeResolution: string;
  setDisputeResolution: (v: string) => void;
}

function DisputesTab({
  selectedDispute, setSelectedDispute,
  disputeStatusChange, setDisputeStatusChange,
  disputeNote, setDisputeNote,
  disputeResolution, setDisputeResolution,
}: DisputesTabProps) {
  const openCount = disputes.filter((d) => d.status === 'open').length;
  const inReviewCount = disputes.filter((d) => d.status === 'in_review').length;
  const resolvedThisMonth = disputes.filter((d) => d.status === 'resolved').length;

  const stats = [
    { label: 'Open Disputes', value: openCount, color: 'text-yellow-600', bg: 'bg-yellow-50' },
    { label: 'In Review', value: inReviewCount, color: 'text-blue-600', bg: 'bg-blue-50' },
    { label: 'Resolved This Month', value: resolvedThisMonth, color: 'text-green-600', bg: 'bg-green-50' },
    { label: 'Avg Resolution Time', value: '2.4 days', color: 'text-indigo-600', bg: 'bg-indigo-50' },
  ];

  function openDetail(d: Dispute) {
    setSelectedDispute(d);
    setDisputeStatusChange(d.status);
    setDisputeNote('');
    setDisputeResolution('');
  }

  return (
    <div className="space-y-6">
      {/* Stats Row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((s) => (
          <div
            key={s.label}
            className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
          >
            <span className="text-sm font-medium text-gray-500">{s.label}</span>
            <div className="mt-2">
              <span className={cn('text-2xl font-bold', s.color)}>{s.value}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Disputes Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">All Disputes</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">Dispute ID</th>
                <th className="px-6 py-3 font-medium">Type</th>
                <th className="px-6 py-3 font-medium">Tenant</th>
                <th className="px-6 py-3 font-medium">Customer</th>
                <th className="px-6 py-3 font-medium">Amount</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Created</th>
                <th className="px-6 py-3 font-medium">Priority</th>
              </tr>
            </thead>
            <tbody>
              {disputes.map((d) => (
                <tr
                  key={d.id}
                  onClick={() => openDetail(d)}
                  className="cursor-pointer border-b border-gray-50 transition-colors hover:bg-gray-50"
                >
                  <td className="px-6 py-4 text-sm font-medium text-indigo-600">{d.id}</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">
                      {disputeTypeLabel[d.type]}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-900">{d.tenant}</td>
                  <td className="px-6 py-4 text-sm text-gray-900">{d.customer}</td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-900">
                    BDT {d.amount.toLocaleString()}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={cn(
                        'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        disputeStatusBadge[d.status],
                      )}
                    >
                      {d.status.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">{formatDate(d.createdAt)}</td>
                  <td className="px-6 py-4">
                    <span
                      className={cn(
                        'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        priorityBadge[d.priority],
                      )}
                    >
                      {d.priority}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Dispute Detail Modal */}
      {selectedDispute && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-gray-200 bg-white shadow-xl">
            {/* Modal Header */}
            <div className="sticky top-0 flex items-center justify-between border-b border-gray-200 bg-white px-6 py-4">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">
                  Dispute {selectedDispute.id}
                </h3>
                <div className="mt-1 flex items-center gap-2">
                  <span
                    className={cn(
                      'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                      disputeStatusBadge[selectedDispute.status],
                    )}
                  >
                    {selectedDispute.status.replace('_', ' ')}
                  </span>
                  <span
                    className={cn(
                      'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                      priorityBadge[selectedDispute.priority],
                    )}
                  >
                    {selectedDispute.priority} priority
                  </span>
                </div>
              </div>
              <button
                onClick={() => setSelectedDispute(null)}
                className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-6 p-6">
              {/* Info Grid */}
              <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                {/* Customer Info */}
                <div>
                  <h4 className="mb-3 text-sm font-semibold text-gray-900">Customer Info</h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-500">Name</span>
                      <span className="font-medium text-gray-900">{selectedDispute.customer}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-500">Email</span>
                      <span className="font-medium text-gray-900">{selectedDispute.customerEmail}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-500">Order Ref</span>
                      <span className="font-medium text-indigo-600">{selectedDispute.orderRef}</span>
                    </div>
                  </div>
                </div>

                {/* Dispute Info */}
                <div>
                  <h4 className="mb-3 text-sm font-semibold text-gray-900">Dispute Info</h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-500">Type</span>
                      <span className="font-medium text-gray-900">{disputeTypeLabel[selectedDispute.type]}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-500">Store</span>
                      <span className="font-medium text-gray-900">{selectedDispute.tenant}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-500">Amount</span>
                      <span className="font-medium text-gray-900">BDT {selectedDispute.amount.toLocaleString()}</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Description */}
              <div>
                <h4 className="mb-2 text-sm font-semibold text-gray-900">Description</h4>
                <p className="rounded-lg bg-gray-50 p-4 text-sm text-gray-700">
                  {selectedDispute.description}
                </p>
              </div>

              {/* Timeline */}
              <div>
                <h4 className="mb-3 text-sm font-semibold text-gray-900">Timeline</h4>
                <div className="space-y-0">
                  {selectedDispute.timeline.map((entry, idx) => (
                    <div key={idx} className="relative flex gap-4 pb-4">
                      {/* Connector line */}
                      {idx < selectedDispute.timeline.length - 1 && (
                        <div className="absolute left-[7px] top-4 h-full w-px bg-gray-200" />
                      )}
                      <div className="relative z-10 mt-1 h-3.5 w-3.5 shrink-0 rounded-full border-2 border-indigo-600 bg-white" />
                      <div>
                        <p className="text-sm text-gray-900">{entry.event}</p>
                        <p className="text-xs text-gray-500">
                          {formatDateTime(entry.date)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Actions */}
              <div className="border-t border-gray-200 pt-6">
                <h4 className="mb-4 text-sm font-semibold text-gray-900">Actions</h4>
                <div className="space-y-4">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Change Status</label>
                    <select
                      value={disputeStatusChange}
                      onChange={(e) => setDisputeStatusChange(e.target.value)}
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    >
                      <option value="open">Open</option>
                      <option value="in_review">In Review</option>
                      <option value="escalated">Escalated</option>
                      <option value="resolved">Resolved</option>
                    </select>
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Add Note</label>
                    <textarea
                      rows={3}
                      value={disputeNote}
                      onChange={(e) => setDisputeNote(e.target.value)}
                      placeholder="Add an internal note about this dispute..."
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Resolve With</label>
                    <select
                      value={disputeResolution}
                      onChange={(e) => setDisputeResolution(e.target.value)}
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    >
                      <option value="">Select resolution...</option>
                      <option value="favor_customer">Favor Customer</option>
                      <option value="favor_tenant">Favor Tenant</option>
                      <option value="partial_refund">Partial Refund</option>
                    </select>
                  </div>
                  <div className="flex gap-3">
                    <button
                      type="button"
                      className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                    >
                      <Save className="h-4 w-4" />
                      Submit
                    </button>
                    <button
                      type="button"
                      onClick={() => setSelectedDispute(null)}
                      className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
