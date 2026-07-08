import type { NextConfig } from "next";

// NOTE: `/proxy/{service}/*` is no longer a transparent rewrite. It is served
// by the BFF Route Handler at src/app/proxy/[service]/[...path]/route.ts, which
// injects the JWT from the HttpOnly auth cookie. The service→origin map lives in
// src/lib/services.ts (shared with that handler).

// Content-Security-Policy for the SSR storefront/admin. Loose enough to let
// Next.js boot (inline + eval scripts are required by the framework) but
// strict enough to confine third-party origins to the image hosts we use
// (Oracle Object Storage) plus our own services for XHR/fetch.
const contentSecurityPolicy = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https://*.oraclecloud.com https://*.compat.objectstorage.ap-singapore-1.oraclecloud.com",
  "font-src 'self' data:",
  "connect-src 'self' http://localhost:* https://*.oraclecloud.com",
  "frame-ancestors 'self'",
].join('; ');

const securityHeaders = [
  {
    key: 'Strict-Transport-Security',
    value: 'max-age=31536000; includeSubDomains',
  },
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff',
  },
  {
    // SAMEORIGIN (not DENY) so the admin can iframe its own storefront
    // previews without breaking.
    key: 'X-Frame-Options',
    value: 'SAMEORIGIN',
  },
  {
    key: 'Referrer-Policy',
    value: 'strict-origin-when-cross-origin',
  },
  {
    key: 'Permissions-Policy',
    value: 'camera=(), microphone=(), geolocation=(), interest-cohort=()',
  },
  {
    key: 'Content-Security-Policy',
    value: contentSecurityPolicy,
  },
];

const nextConfig: NextConfig = {
  output: 'standalone',
  async headers() {
    return [
      {
        // Apply to every route. Next will still let route handlers override
        // individual headers if they need to (e.g. an embeddable widget).
        source: '/:path*',
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
