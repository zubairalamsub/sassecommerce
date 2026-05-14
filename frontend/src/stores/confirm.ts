'use client';

import { create } from 'zustand';

export type ConfirmVariant = 'danger' | 'default';

export interface ConfirmOptions {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: ConfirmVariant;
}

interface PendingConfirm extends ConfirmOptions {
  id: number;
  resolve: (ok: boolean) => void;
}

interface ConfirmState {
  current: PendingConfirm | null;
  /** Imperatively request a confirmation. Resolves with the user's choice. */
  request: (opts: ConfirmOptions) => Promise<boolean>;
  /** Resolve the current dialog with the given result and clear it. */
  resolve: (ok: boolean) => void;
}

let counter = 0;

export const useConfirmStore = create<ConfirmState>()((set, get) => ({
  current: null,

  request: (opts) =>
    new Promise<boolean>((resolve) => {
      // If a dialog is already open, cancel it first so we don't drop the
      // outgoing promise — resolve it as false then queue the new one.
      const existing = get().current;
      if (existing) {
        existing.resolve(false);
      }
      set({ current: { ...opts, id: ++counter, resolve } });
    }),

  resolve: (ok) => {
    const cur = get().current;
    if (!cur) return;
    cur.resolve(ok);
    set({ current: null });
  },
}));

/**
 * React hook for imperative confirmations.
 *
 * Usage:
 *   const confirm = useConfirm();
 *   const ok = await confirm({ title: 'Delete?', variant: 'danger' });
 *   if (ok) doIt();
 */
export function useConfirm() {
  return useConfirmStore.getState().request;
}

/** Module-level helper for non-React callers. */
export const confirmDialog = (opts: ConfirmOptions) => useConfirmStore.getState().request(opts);
