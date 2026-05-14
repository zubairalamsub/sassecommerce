// Applies a tenant's branding colors to the CSS custom properties used across
// the app. Call this from layouts (one-time on tenant config load) and from
// the settings page (live preview on color change).

export interface BrandingColors {
  primary_color?: string;
  secondary_color?: string;
}

/**
 * Sets the --color-primary, --color-primary-dark, --color-primary-light,
 * --color-accent, and --color-accent-light CSS variables on `document.documentElement`.
 *
 * Pass empty/undefined to clear (falls back to defaults from globals.css).
 */
export function applyBrandingColors(branding: BrandingColors | null | undefined): void {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;

  if (!branding) {
    // Clear overrides — let globals.css defaults take over.
    root.style.removeProperty('--color-primary');
    root.style.removeProperty('--color-primary-dark');
    root.style.removeProperty('--color-primary-light');
    root.style.removeProperty('--color-accent');
    root.style.removeProperty('--color-accent-light');
    return;
  }

  const primary = branding.primary_color || '';
  const accent = branding.secondary_color || '';

  if (primary) {
    root.style.setProperty('--color-primary', primary);
    root.style.setProperty('--color-primary-dark', shiftLightness(primary, -12));
    root.style.setProperty('--color-primary-light', shiftLightness(primary, 88));
  } else {
    root.style.removeProperty('--color-primary');
    root.style.removeProperty('--color-primary-dark');
    root.style.removeProperty('--color-primary-light');
  }

  if (accent) {
    root.style.setProperty('--color-accent', accent);
    root.style.setProperty('--color-accent-light', shiftLightness(accent, 88));
  } else {
    root.style.removeProperty('--color-accent');
    root.style.removeProperty('--color-accent-light');
  }
}

/**
 * Returns a hex color shifted toward white (positive amount) or black (negative).
 * Amount is roughly the percentage of distance to the target color, in the 0–100 range.
 */
function shiftLightness(hex: string, amount: number): string {
  const cleaned = hex.replace('#', '');
  const expanded = cleaned.length === 3 ? cleaned.split('').map((c) => c + c).join('') : cleaned;
  if (expanded.length !== 6) return hex;

  const r = parseInt(expanded.substring(0, 2), 16);
  const g = parseInt(expanded.substring(2, 4), 16);
  const b = parseInt(expanded.substring(4, 6), 16);

  const target = amount >= 0 ? 255 : 0;
  const factor = Math.abs(amount) / 100;

  const blend = (channel: number) => Math.round(channel + (target - channel) * factor);

  return (
    '#' +
    [blend(r), blend(g), blend(b)]
      .map((c) => Math.max(0, Math.min(255, c)).toString(16).padStart(2, '0'))
      .join('')
  );
}
