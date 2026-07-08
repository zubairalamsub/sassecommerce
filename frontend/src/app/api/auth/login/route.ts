import { NextRequest, NextResponse } from 'next/server';
import { resolveServiceOrigin } from '@/lib/services';
import { AUTH_COOKIE, authCookieOptions } from '@/lib/auth-cookie';

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

  const data = await upstream.json().catch(() => ({}));
  if (!upstream.ok || !data?.token) {
    return NextResponse.json(
      { error: data?.error || 'Invalid credentials' },
      { status: upstream.ok ? 502 : upstream.status },
    );
  }

  // Return the user (non-secret) to the client; stash the JWT in the cookie.
  const res = NextResponse.json({ user: data.user, expires_at: data.expires_at });
  res.cookies.set(AUTH_COOKIE, data.token, authCookieOptions());
  return res;
}
