import { NextResponse } from 'next/server';
import { AUTH_COOKIE, authCookieOptions } from '@/lib/auth-cookie';

// Clears the HttpOnly auth cookie (B03). Client-side logout can't touch it, so
// it must be cleared server-side.
export const dynamic = 'force-dynamic';

export async function POST() {
  const res = NextResponse.json({ ok: true });
  // maxAge 0 expires it immediately; same flags so the browser matches + clears.
  res.cookies.set(AUTH_COOKIE, '', authCookieOptions(0));
  return res;
}
