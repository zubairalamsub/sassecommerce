import { SignJWT } from 'jose';

const JWT_SECRET = process.env.JWT_SECRET || 'your-secret-key-change-in-production-12345';

export async function POST(request: Request) {
  try {
    const { user_id, tenant_id, email, role } = await request.json();

    if (!user_id || !email || !role) {
      return Response.json({ error: 'Missing required fields' }, { status: 400 });
    }

    const secret = new TextEncoder().encode(JWT_SECRET);

    const token = await new SignJWT({
      user_id,
      tenant_id: tenant_id || '',
      email,
      role,
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
