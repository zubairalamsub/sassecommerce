// Server-side helpers for the HttpOnly auth cookie that carries the backend
// JWT (B03). The token is never exposed to client-side JavaScript: login/logout
// route handlers set/clear this cookie, and the BFF proxy reads it to inject the
// Authorization header when forwarding to backend services.

import type { NextRequest } from 'next/server';

export const AUTH_COOKIE = 'auth_token';

// 24h — matches the backend JWT lifetime (see demo-token route + user-service).
export const AUTH_COOKIE_MAX_AGE = 60 * 60 * 24;

// Short-lived cookie holding the 2FA challenge token between /api/auth/login and
// /api/auth/login/2fa. It only permits completing the second factor, never
// access — kept HttpOnly + brief (matches the backend twoFactorChallengeTTL).
export const CHALLENGE_COOKIE = 'twofa_challenge';
export const CHALLENGE_COOKIE_MAX_AGE = 5 * 60;

export interface AuthCookieOptions {
  httpOnly: true;
  secure: boolean;
  sameSite: 'lax';
  path: '/';
  maxAge?: number;
}

// Flags applied whenever we set the cookie. Secure is on outside development so
// the cookie still works over plain http on localhost during `next dev`.
export function authCookieOptions(maxAge = AUTH_COOKIE_MAX_AGE): AuthCookieOptions {
  return {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    path: '/',
    maxAge,
  };
}

// Reads the raw JWT from the request cookie, or null if absent.
export function readAuthCookie(request: NextRequest): string | null {
  return request.cookies.get(AUTH_COOKIE)?.value ?? null;
}

// Reads the short-lived 2FA challenge token, or null if absent.
export function readChallengeCookie(request: NextRequest): string | null {
  return request.cookies.get(CHALLENGE_COOKIE)?.value ?? null;
}
