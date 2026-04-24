'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  Building2,
  CreditCard,
  Users,
  Settings,
  LogOut,
  Shield,
  Activity,
} from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import ThemeSwitcher from '@/components/ui/theme-switcher';

const navItems = [
  { href: '/platform/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/platform/tenants', label: 'Tenants', icon: Building2 },
  { href: '/platform/plans', label: 'Plans', icon: CreditCard },
  { href: '/platform/users', label: 'Platform Users', icon: Users },
  { href: '/platform/activity', label: 'Activity Log', icon: Activity },
  { href: '/platform/settings', label: 'Settings', icon: Settings },
];

export default function PlatformSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  function handleSignOut() {
    logout();
    router.push('/platform/login');
  }

  return (
    <aside className="fixed left-0 top-0 z-40 flex h-screen w-64 flex-col border-r border-border bg-surface">
      <div className="border-b border-border px-6 py-4">
        <div className="flex items-center gap-2">
          <div className="h-8 w-8 rounded-lg bg-violet-600 flex items-center justify-center">
            <Shield className="h-4 w-4 text-white" />
          </div>
          <span className="text-lg font-semibold text-text">Saajan Platform</span>
        </div>
        <p className="mt-1 text-xs text-text-muted pl-10">Super Admin Console</p>
      </div>

      <nav className="flex-1 space-y-1 px-3 py-4 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive =
            pathname === item.href ||
            (item.href !== '/platform/dashboard' && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200',
                isActive
                  ? 'bg-violet-600/10 text-violet-600 dark:text-violet-400 shadow-sm'
                  : 'text-text-secondary hover:bg-surface-hover hover:text-text',
              )}
            >
              <Icon className="h-5 w-5" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="border-t border-border p-3 space-y-1">
        <div className="flex items-center justify-between px-3 py-1">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-text-muted">Theme</span>
          <ThemeSwitcher compact />
        </div>
        {user && (
          <div className="flex items-center gap-3 rounded-lg px-3 py-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-violet-600/10 text-xs font-semibold text-violet-600 dark:text-violet-400">
              {user.first_name[0]}
              {user.last_name[0]}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-text">
                {user.first_name} {user.last_name}
              </p>
              <span className="inline-block rounded-full bg-violet-100 dark:bg-violet-900/30 px-2 py-0.5 text-[10px] font-medium text-violet-800 dark:text-violet-400">
                super_admin
              </span>
            </div>
          </div>
        )}
        <button
          onClick={handleSignOut}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover hover:text-text transition-colors"
        >
          <LogOut className="h-5 w-5" />
          Sign Out
        </button>
      </div>
    </aside>
  );
}
