// Single source of truth for the storefront/admin "default tenant" used
// when no logged-in user is providing a tenant context.
//
// Resolution order:
//   1. `NEXT_PUBLIC_DEFAULT_TENANT_ID` build-time env var
//   2. `tenant_saajan` legacy fallback (kept so existing seeded data keeps working)
//
// Components that have authenticated users should prefer `auth.tenantId`;
// this helper is for unauthenticated views (storefront browsing, login pages).
export const DEFAULT_TENANT_ID: string =
  process.env.NEXT_PUBLIC_DEFAULT_TENANT_ID || 'tenant_saajan';
