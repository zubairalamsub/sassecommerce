'use client';

import { Building2, UserPlus, CreditCard, Shield, Settings, AlertTriangle } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';

interface ActivityEntry {
  id: string;
  type: 'tenant_created' | 'tenant_suspended' | 'plan_changed' | 'user_added' | 'settings_changed' | 'alert';
  message: string;
  detail: string;
  timestamp: string;
}

const iconMap = {
  tenant_created: { icon: Building2, color: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400' },
  tenant_suspended: { icon: AlertTriangle, color: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400' },
  plan_changed: { icon: CreditCard, color: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400' },
  user_added: { icon: UserPlus, color: 'bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400' },
  settings_changed: { icon: Settings, color: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400' },
  alert: { icon: AlertTriangle, color: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400' },
};

// Activity log will be populated from persisted data once backend is connected
const activities: ActivityEntry[] = [];

export default function ActivityPage() {
  return (
    <div className="space-y-6">
      <motion.div initial={{ opacity: 0, y: -12 }} animate={{ opacity: 1, y: 0 }}>
        <h1 className="text-2xl font-bold text-text">Activity Log</h1>
        <p className="mt-1 text-sm text-text-secondary">Platform-wide audit trail</p>
      </motion.div>

      <motion.div
        className="rounded-2xl border border-border bg-surface shadow-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        {activities.length === 0 ? (
          <div className="py-16 text-center">
            <Shield className="mx-auto h-10 w-10 text-text-muted" />
            <p className="mt-3 text-sm font-medium text-text">No activity yet</p>
            <p className="mt-1 text-sm text-text-muted">
              Platform events will appear here as they happen.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {activities.map((entry, i) => {
              const { icon: Icon, color } = iconMap[entry.type];
              return (
                <motion.div
                  key={entry.id}
                  className="flex items-start gap-4 px-6 py-4"
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: i * 0.05 }}
                >
                  <div className={cn('mt-0.5 rounded-lg p-2', color)}>
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-text">{entry.message}</p>
                    <p className="mt-0.5 text-xs text-text-muted">{entry.detail}</p>
                  </div>
                  <span className="flex-shrink-0 text-xs text-text-muted">{entry.timestamp}</span>
                </motion.div>
              );
            })}
          </div>
        )}
      </motion.div>
    </div>
  );
}
