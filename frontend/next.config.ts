import type { NextConfig } from "next";

const API_BASE = process.env.API_BASE || 'http://localhost';

const SERVICE_PORTS: Record<string, number> = {
  tenant: 8081,
  user: 8082,
  order: 8096,
  product: 8083,
  inventory: 8084,
  payment: 8085,
  shipping: 8086,
  notification: 8087,
  review: 8088,
  cart: 8089,
  search: 8090,
  promotion: 8091,
  vendor: 8092,
  analytics: 8093,
  recommendation: 8094,
  config: 8095,
  prometheus: 9090,
};

// Per-service URLs override the default `${API_BASE}:${port}`. Used in Docker
// where each service has its own DNS hostname on the internal network.
function serviceTarget(service: string, port: number): string {
  const envKey = `${service.toUpperCase()}_SERVICE_URL`;
  return process.env[envKey] || `${API_BASE}:${port}`;
}

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
  async rewrites() {
    return Object.entries(SERVICE_PORTS).map(([service, port]) => ({
      source: `/proxy/${service}/:path*`,
      destination: `${serviceTarget(service, port)}/:path*`,
    }));
  },
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
