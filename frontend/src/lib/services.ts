// Server-side service registry for the BFF proxy (src/app/proxy/[service]/[...path]).
//
// This is the single source of truth for which backend each `/proxy/{service}`
// prefix maps to. It used to live inline in next.config.ts as a rewrite table;
// it moved here when the transparent rewrite was replaced by a Route Handler
// that injects the auth token from the HttpOnly cookie (B03).
//
// Do NOT import this from client components — it is only meaningful on the
// server (it reads process.env service URLs).

const API_BASE = process.env.API_BASE || 'http://localhost';

export const SERVICE_PORTS: Record<string, number> = {
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
export function serviceTarget(service: string, port: number): string {
  const envKey = `${service.toUpperCase()}_SERVICE_URL`;
  return process.env[envKey] || `${API_BASE}:${port}`;
}

// Resolves a `/proxy/{service}` prefix to its backend origin, or null if the
// service name is not one we proxy (so the handler can 404 instead of
// forwarding to an arbitrary host).
export function resolveServiceOrigin(service: string): string | null {
  const port = SERVICE_PORTS[service];
  if (port === undefined) return null;
  return serviceTarget(service, port);
}
