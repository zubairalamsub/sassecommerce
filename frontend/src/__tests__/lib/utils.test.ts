/**
 * These are pure-logic helpers (no DOM). Run them in the lighter `node`
 * environment instead of the project-default jsdom to avoid per-file jsdom
 * construction. NOTE: the zustand store tests deliberately stay on jsdom —
 * they rely on localStorage (persist middleware) and @testing-library/react.
 *
 * @jest-environment node
 */
import { cn, formatCurrency, formatDate, formatDateTime, statusColor } from '@/lib/utils';

describe('cn', () => {
  test('merges class names', () => {
    expect(cn('px-4', 'py-2')).toBe('px-4 py-2');
  });

  test('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible');
  });

  test('resolves Tailwind conflicts (last wins)', () => {
    const result = cn('px-4', 'px-6');
    expect(result).toBe('px-6');
  });

  test('handles empty/undefined inputs', () => {
    expect(cn('', undefined, null, 'text-sm')).toBe('text-sm');
  });
});

describe('formatCurrency', () => {
  test('formats BDT by default', () => {
    const result = formatCurrency(1500);
    // Should contain BDT symbol and formatted number
    expect(result).toContain('1,500');
  });

  test('formats zero', () => {
    const result = formatCurrency(0);
    expect(result).toContain('0');
  });

  test('formats decimals up to 2 places', () => {
    const result = formatCurrency(99.5);
    expect(result).toContain('99.5');
  });

  test('accepts different currency', () => {
    const result = formatCurrency(100, 'USD');
    expect(result).toContain('100');
  });

  test('formats large numbers with commas', () => {
    const result = formatCurrency(1284500);
    // Localized formatting should include separators
    expect(result.replace(/[^\d,]/g, '')).toMatch(/1,284,500|12,84,500/);
  });
});

describe('formatDate', () => {
  test('formats ISO date string', () => {
    const result = formatDate('2026-04-17');
    expect(result).toContain('2026');
    expect(result).toContain('17');
  });

  test('formats date with time component', () => {
    const result = formatDate('2026-04-17T10:30:00Z');
    expect(result).toContain('2026');
  });
});

describe('formatDateTime', () => {
  test('formats date with time', () => {
    const result = formatDateTime('2026-04-17T10:30:00Z');
    expect(result).toContain('2026');
    // Should include time portion
    expect(result.length).toBeGreaterThan(formatDate('2026-04-17').length);
  });
});

describe('statusColor', () => {
  test('returns green for active', () => {
    expect(statusColor('active')).toContain('green');
  });

  test('returns yellow for pending', () => {
    expect(statusColor('pending')).toContain('yellow');
  });

  test('returns red for cancelled', () => {
    expect(statusColor('cancelled')).toContain('red');
  });

  test('returns blue for confirmed', () => {
    expect(statusColor('confirmed')).toContain('blue');
  });

  test('returns purple for shipped', () => {
    expect(statusColor('shipped')).toContain('purple');
  });

  test('returns neutral surface for draft', () => {
    // draft/inactive/archived map to the semantic surface-hover token so
    // they swap palettes correctly between light and dark mode.
    expect(statusColor('draft')).toContain('surface-hover');
  });

  test('returns neutral surface fallback for unknown status', () => {
    expect(statusColor('unknown_status')).toContain('surface-hover');
  });

  test('handles capitalized payment statuses', () => {
    expect(statusColor('Completed')).toContain('green');
    expect(statusColor('Failed')).toContain('red');
    expect(statusColor('Refunded')).toContain('orange');
  });
});
