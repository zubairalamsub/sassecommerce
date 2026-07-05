import { SignJWT } from 'jose';

// Server-side allowlist of demo accounts. The JWT claims (user_id, tenant_id,
// role) are taken from this map — never from the request body — so a caller
// can only obtain a token for one of these fixed identities, and only by
// presenting the matching password.
const DEFAULT_TENANT_ID = process.env.NEXT_PUBLIC_DEFAULT_TENANT_ID || 'tenant_saajan';

const DEMO_ACCOUNTS: Record<
  string,
  { password: string; user_id: string; tenant_id: string; role: string }
> = {
  'super@saajan.com.bd': { password: 'super123', user_id: 'su-001', tenant_id: '', role: 'super_admin' },
  'admin@fashion.com.bd': { password: 'admin123', user_id: 'ta-001', tenant_id: DEFAULT_TENANT_ID, role: 'admin' },
  'staff@fashion.com.bd': { password: 'staff123', user_id: 'tm-001', tenant_id: DEFAULT_TENANT_ID, role: 'moderator' },
  'rahim@example.com': { password: 'customer123', user_id: 'cu-001', tenant_id: DEFAULT_TENANT_ID, role: 'customer' },
};

export async function POST(request: Request) {
  // Demo logins are a development convenience only. In production the route
  // does not exist as far as callers can tell.
  if (process.env.NODE_ENV === 'production') {
    return Response.json({ error: 'Not found' }, { status: 404 });
  }

  const JWT_SECRET = process.env.JWT_SECRET;
  if (!JWT_SECRET) {
    return Response.json({ error: 'Server misconfigured: JWT_SECRET is not set' }, { status: 500 });
  }

  try {
    const { email, password } = await request.json();

    if (!email || !password) {
      return Response.json({ error: 'Missing required fields' }, { status: 400 });
    }

    const account = DEMO_ACCOUNTS[email];
    if (!account || account.password !== password) {
      return Response.json({ error: 'Invalid credentials' }, { status: 401 });
    }

    const secret = new TextEncoder().encode(JWT_SECRET);

    const token = await new SignJWT({
      user_id: account.user_id,
      tenant_id: account.tenant_id,
      email,
      role: account.role,
    })
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt()
      .setExpirationTime('24h')
      .sign(secret);

    return Response.json({ token });
  } catch {
    return Response.json({ error: 'Failed to generate token' }, { status: 500 });
  }
}
