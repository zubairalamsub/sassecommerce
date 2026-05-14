'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { AlertTriangle, HelpCircle, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useConfirmStore, type ConfirmVariant } from '@/stores/confirm';

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: ConfirmVariant;
  /** Called when the user clicks Confirm. May be async — the dialog will
   *  show a loading spinner while the promise is pending. If the promise
   *  rejects, the dialog stays open so the caller can show an error toast. */
  onConfirm: () => void | Promise<void>;
}

/**
 * Polished confirmation dialog.
 *
 * Visual style:
 * - Modal: dark backdrop (40% black), centered, ~440px wide, rounded-2xl, p-6.
 * - Variant "danger": red trash icon at top, red Confirm button.
 * - Variant "default": blue/primary icon, primary-colored Confirm button.
 * - Spring-in animation (matching the toast style).
 * - Focus is trapped while open. Backdrop click or Escape closes it.
 */
export default function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  onConfirm,
}: ConfirmDialogProps) {
  const [pending, setPending] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const confirmBtnRef = useRef<HTMLButtonElement>(null);
  const cancelBtnRef = useRef<HTMLButtonElement>(null);
  const previousActive = useRef<HTMLElement | null>(null);

  // Reset pending whenever the dialog opens — protects against stale state
  // after a previous failed attempt.
  useEffect(() => {
    if (open) setPending(false);
  }, [open]);

  // Escape to dismiss; focus trap (Tab/Shift-Tab loops between the two buttons).
  useEffect(() => {
    if (!open) return;

    previousActive.current = document.activeElement as HTMLElement | null;
    // Default focus on Cancel for safety with destructive variants.
    requestAnimationFrame(() => {
      (cancelBtnRef.current ?? confirmBtnRef.current)?.focus();
    });

    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !pending) {
        e.preventDefault();
        onOpenChange(false);
        return;
      }
      if (e.key === 'Tab') {
        // Two-button focus trap.
        const focusables = [cancelBtnRef.current, confirmBtnRef.current].filter(
          Boolean,
        ) as HTMLElement[];
        if (focusables.length === 0) return;
        const active = document.activeElement;
        const idx = focusables.findIndex((el) => el === active);
        e.preventDefault();
        const next = e.shiftKey
          ? focusables[(idx - 1 + focusables.length) % focusables.length]
          : focusables[(idx + 1) % focusables.length];
        next.focus();
      }
    }

    document.addEventListener('keydown', handleKey);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handleKey);
      document.body.style.overflow = '';
      // Restore focus to whatever opened the dialog.
      previousActive.current?.focus?.();
    };
  }, [open, pending, onOpenChange]);

  const handleConfirm = useCallback(async () => {
    if (pending) return;
    setPending(true);
    try {
      await onConfirm();
      // Success — close. Reset pending lazily; if the dialog re-opens it'll
      // be reset by the open-effect anyway.
      onOpenChange(false);
    } catch {
      // Stay open so caller can surface an error toast. Re-enable buttons.
      setPending(false);
    }
  }, [pending, onConfirm, onOpenChange]);

  function handleBackdropClick() {
    if (pending) return;
    onOpenChange(false);
  }

  const isDanger = variant === 'danger';
  const Icon = isDanger ? AlertTriangle : HelpCircle;

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="fixed inset-0 z-[200] flex items-center justify-center p-4"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.18 }}
          aria-hidden={!open}
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/40 backdrop-blur-sm"
            onClick={handleBackdropClick}
            aria-hidden="true"
          />

          {/* Dialog */}
          <motion.div
            ref={dialogRef}
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="confirm-dialog-title"
            aria-describedby={description ? 'confirm-dialog-description' : undefined}
            initial={{ opacity: 0, scale: 0.96, y: 12 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 8, transition: { duration: 0.16 } }}
            transition={{ type: 'spring', stiffness: 380, damping: 32 }}
            className={cn(
              'relative w-full max-w-[440px] rounded-2xl border border-border bg-surface p-6',
              'shadow-[0_24px_60px_-12px_rgb(0_0_0_/_0.35),0_8px_18px_-6px_rgb(0_0_0_/_0.18)]',
            )}
          >
            {/* Icon */}
            <div className="flex items-start gap-4">
              <div
                className={cn(
                  'flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full',
                  isDanger ? 'bg-red-100 text-red-600' : 'bg-primary-light text-primary',
                )}
                aria-hidden="true"
              >
                <Icon className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h2
                  id="confirm-dialog-title"
                  className="text-base font-semibold text-text"
                >
                  {title}
                </h2>
                {description && (
                  <p
                    id="confirm-dialog-description"
                    className="mt-1 text-sm text-text-secondary"
                  >
                    {description}
                  </p>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="mt-6 flex items-center justify-end gap-2.5">
              <button
                ref={cancelBtnRef}
                type="button"
                onClick={() => onOpenChange(false)}
                disabled={pending}
                className={cn(
                  'inline-flex h-10 items-center justify-center rounded-lg border border-border bg-surface px-4 text-sm font-medium text-text-secondary transition-colors',
                  'hover:bg-surface-hover focus:outline-none focus:ring-2 focus:ring-primary/30',
                  'disabled:cursor-not-allowed disabled:opacity-50',
                )}
              >
                {cancelLabel}
              </button>
              <button
                ref={confirmBtnRef}
                type="button"
                onClick={handleConfirm}
                disabled={pending}
                className={cn(
                  'inline-flex h-10 min-w-[88px] items-center justify-center gap-2 rounded-lg px-4 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-surface',
                  isDanger
                    ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500/50'
                    : 'bg-primary hover:bg-primary-dark focus:ring-primary/50',
                  'disabled:cursor-not-allowed disabled:opacity-70',
                )}
              >
                {pending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
                {pending ? 'Working...' : confirmLabel}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/**
 * Root component — mount once at the app layout. It listens to the
 * confirm store and renders the dialog imperatively, so callers can use
 * `useConfirm()` without rendering anything themselves.
 */
export function ConfirmDialogRoot() {
  const current = useConfirmStore((s) => s.current);
  const resolve = useConfirmStore((s) => s.resolve);

  const open = current !== null;

  return (
    <ConfirmDialog
      // Keying on the dialog id makes sure each new confirmation resets
      // internal state (pending flag, focus position).
      key={current?.id ?? 'idle'}
      open={open}
      onOpenChange={(o) => {
        if (!o) resolve(false);
      }}
      title={current?.title ?? ''}
      description={current?.description}
      confirmLabel={current?.confirmLabel}
      cancelLabel={current?.cancelLabel}
      variant={current?.variant ?? 'default'}
      onConfirm={() => resolve(true)}
    />
  );
}
