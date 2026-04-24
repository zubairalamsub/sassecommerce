'use client';

import { useState } from 'react';
import { Shield, UserPlus, Trash2 } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn, formatDate } from '@/lib/utils';
import { DEMO_USERS, type AuthUser } from '@/stores/auth';

export default function PlatformUsersPage() {
  const platformUsers = Object.values(DEMO_USERS)
    .map((d) => d.user)
    .filter((u) => u.role === 'super_admin');

  return (
    <div className="space-y-6">
      <motion.div
        className="flex items-center justify-between"
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <div>
          <h1 className="text-2xl font-bold text-text">Platform Users</h1>
          <p className="mt-1 text-sm text-text-secondary">Manage super admin accounts</p>
        </div>
        <button className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-violet-700">
          <UserPlus className="h-4 w-4" />
          Invite Admin
        </button>
      </motion.div>

      <motion.div
        className="rounded-2xl border border-border bg-surface shadow-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                <th className="px-6 py-3 font-medium">User</th>
                <th className="px-6 py-3 font-medium">Email</th>
                <th className="px-6 py-3 font-medium">Role</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Joined</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {platformUsers.map((user) => (
                <tr key={user.id} className="border-b border-border-light last:border-0 hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30 text-xs font-semibold text-violet-600 dark:text-violet-400">
                        {user.first_name[0]}{user.last_name[0]}
                      </div>
                      <span className="text-sm font-medium text-text">{user.first_name} {user.last_name}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">{user.email}</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center gap-1 rounded-full bg-violet-100 dark:bg-violet-900/30 px-2.5 py-0.5 text-xs font-medium text-violet-700 dark:text-violet-400">
                      <Shield className="h-3 w-3" />
                      Super Admin
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className="inline-flex rounded-full bg-green-100 dark:bg-green-900/30 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:text-green-400">
                      {user.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-text-muted">{formatDate(user.created_at)}</td>
                  <td className="px-6 py-4">
                    <button className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </motion.div>
    </div>
  );
}
