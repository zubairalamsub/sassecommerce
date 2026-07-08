import { NextRequest, NextResponse } from 'next/server';
import { resolveServiceOrigin } from '@/lib/services';
import {
  AUTH_COOKIE,
  CHALLENGE_COOKIE,
  authCookieOptions,
  readChallengeCookie,
} from '@/lib/auth-cookie';

// Completes a 2FA-gated login (A04-3). The challenge token lives in a short-lived
// HttpOnly cookie set by /api/auth/login; the browser only supplies the code. On
// success the JWT is swapped into the auth cookie and the challenge is cleared.
export const dynamic = 'force-dynamic';

export async function POST(request: NextRequest) {
  const body = await request.json().catch(() => null);
  const code = body?.code;
  if (!code) {
    return NextResponse.json({ error: 'Missing code' }, { status: 400 });
  }

  const challengeToken = readChallengeCookie(request);
  if (!challengeToken) {
    return NextResponse.json({ error: 'No pending 2FA challenge' }, { status: 400 });
  }

  const origin = resolveServiceOrigin('user');
  if (!origin) {
    return NextResponse.json({ error: 'Auth service not configured' }, { status: 500 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${origin}/api/v1/auth/login/2fa`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ challenge_token: challengeToken, code }),
      cache: 'no-store',
    });
  } catch {
    return NextResponse.json({ error: 'Auth service unavailable' }, { status: 502 });
  }

  const body2 = await upstream.json().catch(() => ({}));
  if (!upstream.ok) {
    return NextResponse.json({ error: body2?.error || 'Invalid code' }, { status: upstream.status });
  }

  const payload = body2?.data ?? body2;
  if (!payload?.token) {
    return NextResponse.json({ error: 'Invalid code' }, { status: 502 });
  }

  const res = NextResponse.json({ user: payload.user, expires_at: payload.expires_at });
  res.cookies.set(AUTH_COOKIE, payload.token, authCookieOptions());
  // Challenge consumed — clear it.
  res.cookies.set(CHALLENGE_COOKIE, '', authCookieOptions(0));
  return res;
}
