'use client';

import { useState, useRef, useEffect } from 'react';
import { Sun, Moon, Monitor, Palette, Check, X } from 'lucide-react';
import { useThemeStore, ACCENT_PRESETS, type ThemeMode, type AccentPreset } from '@/stores/theme';
import { cn } from '@/lib/utils';

const modeOptions: { value: ThemeMode; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'system', label: 'System', icon: Monitor },
];

export default function ThemeSwitcher({ compact = false }: { compact?: boolean }) {
  const { mode, accent, setMode, setAccent, resolvedMode } = useThemeStore();
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => { setMounted(true); }, []);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    if (open) document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [open]);

  // Always render Sun on server; switch to correct icon after mount
  const CurrentIcon = mounted && resolvedMode() === 'dark' ? Moon : Sun;
  const currentPreset = ACCENT_PRESETS[accent];

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className={cn(
          'flex items-center gap-2 rounded-lg p-2 transition-all duration-200',
          'text-text-secondary hover:bg-surface-hover hover:text-text',
        )}
        title="Theme Settings"
      >
        <CurrentIcon className="h-5 w-5" />
        {!compact && <Palette className="h-3.5 w-3.5 opacity-50" />}
      </button>

      {/* Full-screen overlay modal for theme selection */}
      {open && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
          {/* Backdrop */}
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => setOpen(false)} />

          {/* Modal */}
          <div className="relative w-full max-w-md rounded-2xl border border-border bg-surface shadow-2xl overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div className="flex items-center gap-2">
                <Palette className="h-5 w-5 text-primary" />
                <h3 className="text-base font-semibold text-text">Theme Settings</h3>
              </div>
              <button
                onClick={() => setOpen(false)}
                className="rounded-lg p-1.5 text-text-muted hover:bg-surface-hover hover:text-text transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="p-5 space-y-5">
              {/* Mode Selection */}
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">Appearance</p>
                <div className="grid grid-cols-3 gap-3">
                  {modeOptions.map((opt) => {
                    const Icon = opt.icon;
                    const isActive = mode === opt.value;
                    return (
                      <button
                        key={opt.value}
                        onClick={() => setMode(opt.value)}
                        className={cn(
                          'flex flex-col items-center gap-2 rounded-xl px-3 py-4 text-xs font-medium transition-all border-2',
                          isActive
                            ? 'border-primary bg-primary/5 text-primary'
                            : 'border-border bg-surface-secondary text-text-secondary hover:border-primary/30 hover:bg-surface-hover',
                        )}
                      >
                        <div className={cn(
                          'flex h-10 w-10 items-center justify-center rounded-xl transition-colors',
                          isActive ? 'bg-primary/10' : 'bg-surface-hover',
                        )}>
                          <Icon className="h-5 w-5" />
                        </div>
                        {opt.label}
                        {isActive && <Check className="h-3.5 w-3.5" />}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Accent Color Selection */}
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">Accent Color</p>
                <div className="grid grid-cols-3 gap-3">
                  {(Object.entries(ACCENT_PRESETS) as [AccentPreset, typeof ACCENT_PRESETS[AccentPreset]][]).map(([key, preset]) => {
                    const isActive = accent === key;
                    return (
                      <button
                        key={key}
                        onClick={() => setAccent(key)}
                        className={cn(
                          'flex items-center gap-2.5 rounded-xl px-3 py-3 text-sm font-medium transition-all border-2',
                          isActive
                            ? 'border-primary bg-primary/5'
                            : 'border-border bg-surface-secondary hover:border-primary/30 hover:bg-surface-hover',
                        )}
                      >
                        <span
                          className="h-6 w-6 rounded-full flex-shrink-0 ring-2 ring-black/5 dark:ring-white/10 flex items-center justify-center shadow-sm"
                          style={{ backgroundColor: preset.primary }}
                        >
                          {isActive && <Check className="h-3 w-3 text-white" />}
                        </span>
                        <span className="text-text truncate">{preset.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Preview — uses inline styles so it updates instantly */}
              <div className="rounded-xl border border-border bg-surface-secondary p-4">
                <p className="text-xs font-semibold uppercase tracking-wider text-text-muted mb-3">Preview</p>
                <div className="flex items-center gap-3">
                  <div className="h-10 w-10 rounded-xl flex items-center justify-center" style={{ backgroundColor: currentPreset.primary }}>
                    <span className="text-white font-bold text-sm">S</span>
                  </div>
                  <div className="flex-1">
                    <div className="h-2.5 w-24 rounded-full" style={{ backgroundColor: currentPreset.primary + '33' }} />
                    <div className="h-2 w-16 rounded-full bg-text-muted/20 mt-1.5" />
                  </div>
                  <button className="rounded-lg px-4 py-2 text-xs font-semibold text-white" style={{ backgroundColor: currentPreset.primary }}>
                    Button
                  </button>
                  <span className="rounded-full px-3 py-1 text-xs font-bold text-white" style={{ backgroundColor: currentPreset.accent }}>
                    Badge
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
