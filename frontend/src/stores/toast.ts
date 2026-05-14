'use client';

import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  duration?: number;
}

export interface ToastOptions {
  title?: string;
  /** Display duration in ms. Default 4000. Set to 0 to require manual dismiss. */
  duration?: number;
}

interface ToastState {
  toasts: Toast[];
  addToast: (type: ToastType, message: string, duration?: number) => void;
  show: (type: ToastType, message: string, opts?: ToastOptions) => void;
  removeToast: (id: string) => void;
}

let counter = 0;

export const useToastStore = create<ToastState>()((set, get) => ({
  toasts: [],
  addToast: (type, message, duration = 3000) => get().show(type, message, { duration }),
  show: (type, message, opts = {}) => {
    const id = `toast-${++counter}`;
    const duration = opts.duration ?? 4000;
    set((s) => ({ toasts: [...s.toasts, { id, type, message, title: opts.title, duration }] }));
    if (duration > 0) {
      setTimeout(() => {
        set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
      }, duration);
    }
  },
  removeToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

/**
 * `toast` — convenience helpers usable from anywhere (including outside React).
 * Mirrors the API of common toast libraries: `toast.success("Saved")`.
 */
export const toast = {
  success: (message: string, opts?: ToastOptions) =>
    useToastStore.getState().show('success', message, opts),
  error: (message: string, opts?: ToastOptions) =>
    useToastStore.getState().show('error', message, opts),
  info: (message: string, opts?: ToastOptions) =>
    useToastStore.getState().show('info', message, opts),
  warning: (message: string, opts?: ToastOptions) =>
    useToastStore.getState().show('warning', message, opts),
};
