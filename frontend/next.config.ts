import type { NextConfig } from "next";

const API_BASE = process.env.API_BASE || 'http://localhost';

const SERVICE_PORTS: Record<string, number> = {
  tenant: 8081,
  user: 8082,
  order: 8080,
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
};

const nextConfig: NextConfig = {
  output: 'standalone',
  async rewrites() {
    return Object.entries(SERVICE_PORTS).map(([service, port]) => ({
      source: `/proxy/${service}/:path*`,
      destination: `${API_BASE}:${port}/:path*`,
    }));
  },
};

export default nextConfig;
