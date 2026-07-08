import { NextRequest, NextResponse } from 'next/server';
import { resolveServiceOrigin } from '@/lib/services';
import { readAuthCookie } from '@/lib/auth-cookie';

// BFF proxy (B03). Replaces the old transparent next.config.ts rewrite of
// `/proxy/{service}/:path*`. The browser no longer holds the JWT — it lives in
// an HttpOnly cookie — so this handler reads that cookie server-side and injects
// the `Authorization: Bearer <jwt>` header when forwarding to the backend.
//
// Tenant is NOT derived here: the backend services already resolve the tenant
// from the verified JWT for authenticated requests (see the tenant-isolation
// remediation), and fall back to the X-Tenant-ID header only for anonymous
// requests — which we forward unchanged so public storefront reads keep working.

// This handler must run per-request (it reads cookies + forwards live traffic).
export const dynamic = 'force-dynamic';

// Hop-by-hop headers must not be forwarded (RFC 7230 §6.1); plus a few that the
// runtime manages itself. Everything else from the client is passed through.
const STRIP_REQUEST_HEADERS = new Set([
  'host',
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'content-length',
  // The client must never dictate the credential — we set it from the cookie.
  'authorization',
  'cookie',
]);

const STRIP_RESPONSE_HEADERS = new Set([
  'connection',
  'keep-alive',
  'transfer-encoding',
  'content-encoding',
  'content-length',
]);

async function handle(
  request: NextRequest,
  ctx: { params: Promise<{ service: string }> },
): Promise<Response> {
  const { service } = await ctx.params;

  const origin = resolveServiceOrigin(service);
  if (!origin) {
    return NextResponse.json({ error: `Unknown service '${service}'` }, { status: 404 });
  }

  // Reconstruct the upstream path from the raw pathname (not the decoded
  // catch-all param) so percent-encoding in ids/codes round-trips exactly and
  // is never double-encoded.
  const prefix = `/proxy/${service}/`;
  const { pathname, search } = request.nextUrl;
  const suffix = pathname.startsWith(prefix) ? pathname.slice(prefix.length) : '';
  const target = `${origin}/${suffix}${search}`;

  // Rebuild the outgoing headers: copy the client's (minus hop-by-hop), then set
  // Authorization from the HttpOnly cookie. During the B03 transition we fall
  // back to any client-supplied Authorization so requests keep working before
  // the cookie cutover; that fallback is removed once the client stops sending
  // the token.
  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!STRIP_REQUEST_HEADERS.has(key.toLowerCase())) headers.set(key, value);
  });

  const cookieToken = readAuthCookie(request);
  const clientAuth = request.headers.get('authorization');
  const token = cookieToken ?? (clientAuth?.startsWith('Bearer ') ? clientAuth.slice(7) : null);
  if (token) headers.set('authorization', `Bearer ${token}`);

  const method = request.method;
  const hasBody = method !== 'GET' && method !== 'HEAD';

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method,
      headers,
      body: hasBody ? await request.arrayBuffer() : undefined,
      redirect: 'manual',
      cache: 'no-store',
    });
  } catch {
    return NextResponse.json({ error: 'Upstream service unavailable' }, { status: 502 });
  }

  const responseHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (!STRIP_RESPONSE_HEADERS.has(key.toLowerCase())) responseHeaders.set(key, value);
  });

  return new Response(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: responseHeaders,
  });
}

export const GET = handle;
export const POST = handle;
export const PUT = handle;
export const PATCH = handle;
export const DELETE = handle;
export const HEAD = handle;
export const OPTIONS = handle;
