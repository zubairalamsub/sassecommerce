import { NextRequest, NextResponse } from 'next/server';
import { resolveServiceOrigin } from '@/lib/services';
import {
  AUTH_COOKIE,
  CHALLENGE_COOKIE,
  CHALLENGE_COOKIE_MAX_AGE,
  authCookieOptions,
} from '@/lib/auth-cookie';

// Server-side login (B03). The browser posts credentials here; we call the
// user-service, capture the JWT server-side, and hand it back to the browser
// ONLY as an HttpOnly cookie — the token never reaches client JavaScript.
export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  const body = await request.json().catch(() => null);
  if (!body?.email || !body?.password || !body?.tenant_id) {
    return NextResponse.json({ error: 'Missing credentials' }, { status: 400 });
  }

  const origin = resolveServiceOrigin('user');
  if (!origin) {
    return NextResponse.json({ error: 'Auth service not configured' }, { status: 500 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${origin}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': body.tenant_id },
      body: JSON.stringify({ tenant_id: body.tenant_id, email: body.email, password: body.password }),
      cache: 'no-store',
    });
  } catch {
    return NextResponse.json({ error: 'Auth service unavailable' }, { status: 502 });
  }

  const body2 = await upstream.json().catch(() => ({}));
  if (!upstream.ok) {
    return NextResponse.json({ error: body2?.error || 'Invalid credentials' }, { status: upstream.status });
  }

  // user-service wraps success as { success, data: LoginResponse }.
  const payload = body2?.data ?? body2;

  // 2FA-enrolled users get a challenge (no token yet). Stash the short-lived
  // challenge token in an HttpOnly cookie and tell the client to collect a code.
  if (payload?.two_factor?.required && payload?.two_factor?.challenge_token) {
    const res = NextResponse.json({ twoFactorRequired: true });
    res.cookies.set(CHALLENGE_COOKIE, payload.two_factor.challenge_token, authCookieOptions(CHALLENGE_COOKIE_MAX_AGE));
    return res;
  }

  if (!payload?.token) {
    return NextResponse.json({ error: 'Invalid credentials' }, { status: 502 });
  }

  // Return the user (non-secret) to the client; stash the JWT in the cookie.
  const res = NextResponse.json({ user: payload.user, expires_at: payload.expires_at });
  res.cookies.set(AUTH_COOKIE, payload.token, authCookieOptions());
  return res;
}
