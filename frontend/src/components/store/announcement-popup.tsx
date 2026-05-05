'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { X, ShoppingBag } from 'lucide-react';
import { useStoreConfigStore, type AnnouncementPopup as PopupConfig } from '@/stores/store-config';

const STORAGE_KEY = 'announcement_popup_dismissed';

function isWithinSchedule(popup: PopupConfig): boolean {
  const now = new Date();
  if (popup.start_date) {
    // Parse as local midnight (not UTC) so the date matches the user's timezone
    const start = new Date(popup.start_date + 'T00:00:00');
    if (now < start) return false;
  }
  if (popup.end_date) {
    const end = new Date(popup.end_date + 'T23:59:59');
    if (now > end) return false;
  }
  return true;
}

function wasDismissed(popup: PopupConfig): boolean {
  if (!popup.show_once) return false;
  try {
    const dismissed = localStorage.getItem(STORAGE_KEY);
    if (!dismissed) return false;
    const data = JSON.parse(dismissed);
    // If the popup content changed, show it again
    return data.title === popup.title && data.message === popup.message;
  } catch {
    return false;
  }
}

function markDismissed(popup: PopupConfig) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      title: popup.title,
      message: popup.message,
      at: new Date().toISOString(),
    }));
  } catch {
    // localStorage unavailable
  }
}

export default function AnnouncementPopup() {
  const popup = useStoreConfigStore((s) => s.config.announcement_popup);
  const loading = useStoreConfigStore((s) => s.loading);
  const [visible, setVisible] = useState(false);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    // Wait until config has finished loading from the backend
    if (loading) return;
    if (!popup?.enabled) return;
    if (!isWithinSchedule(popup)) return;
    if (wasDismissed(popup)) return;

    // Small delay so the page renders first
    const timer = setTimeout(() => setVisible(true), 800);
    return () => clearTimeout(timer);
  }, [popup, loading]);

  function handleClose() {
    setClosing(true);
    markDismissed(popup);
    setTimeout(() => {
      setVisible(false);
      setClosing(false);
    }, 200);
  }

  if (!visible) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      {/* Overlay */}
      <div
        className={`absolute inset-0 bg-black transition-opacity duration-200 ${closing ? 'opacity-0' : 'opacity-50'}`}
        style={{ opacity: closing ? 0 : (popup.overlay_opacity ?? 50) / 100 }}
        onClick={handleClose}
      />

      {/* Modal */}
      <div
        className={`relative w-full max-w-md overflow-hidden rounded-2xl shadow-2xl transition-all duration-200 ${closing ? 'scale-95 opacity-0' : 'scale-100 opacity-100'}`}
        style={{ backgroundColor: popup.bg_color || '#ffffff', color: popup.text_color || '#111827' }}
      >
        {/* Close button */}
        <button
          onClick={handleClose}
          className="absolute right-3 top-3 z-10 rounded-full bg-black/10 p-1.5 backdrop-blur-sm transition-colors hover:bg-black/20"
        >
          <X className="h-4 w-4" style={{ color: popup.text_color || '#111827' }} />
        </button>

        {/* Image */}
        {popup.image_url && (
          <div className="relative h-48 w-full overflow-hidden">
            <img
              src={popup.image_url}
              alt={popup.title}
              className="h-full w-full object-cover"
            />
          </div>
        )}

        {/* Content */}
        <div className="p-6 text-center">
          {popup.title && (
            <h2 className="text-2xl font-bold mb-2" style={{ color: popup.text_color || '#111827' }}>
              {popup.title}
            </h2>
          )}
          {popup.message && (
            <p className="text-sm leading-relaxed mb-6 opacity-80" style={{ color: popup.text_color || '#111827' }}>
              {popup.message}
            </p>
          )}
          {popup.cta_text && (
            <Link
              href={popup.cta_link || '/products'}
              onClick={handleClose}
              className="inline-flex items-center gap-2 rounded-full px-8 py-3 text-sm font-semibold shadow-lg transition-all hover:scale-105 hover:shadow-xl"
              style={{
                backgroundColor: popup.text_color || '#111827',
                color: popup.bg_color || '#ffffff',
              }}
            >
              <ShoppingBag className="h-4 w-4" />
              {popup.cta_text}
            </Link>
          )}
          <button
            onClick={handleClose}
            className="mt-4 block w-full text-xs opacity-50 hover:opacity-70 transition-opacity"
            style={{ color: popup.text_color || '#111827' }}
          >
            No thanks, maybe later
          </button>
        </div>
      </div>
    </div>
  );
}
