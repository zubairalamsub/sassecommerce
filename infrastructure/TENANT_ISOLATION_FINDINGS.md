# Tenant Isolation Findings

Audit date: 2026-07-06. Scope: all 16 backend services + the Next.js frontend.

## Summary

The platform has a **systemic cross-tenant access vulnerability**. JWT auth is
enforced on most services and the verified tenant claim is available in request
context (`shared/go/pkg/middleware/auth.go` sets `tenant_id`; `GetTenantID(c)`
reads it), but handlers overwhelmingly **ignore it** and instead take the tenant
(and often the user) from client-controlled inputs:

- `c.Query("tenant_id")` / `c.Query("user_id")`
- request-body `tenant_id` fields
- path params (`/:id`, `/tenants/:tenantId`) with no tenant check
- the `X-Tenant-ID` header (the frontend sends this from client state)

and repositories look records up by primary key alone (`WHERE id = ?` /
`{_id: ...}`) with no `tenant_id` predicate. The net effect: an authenticated
user of tenant A can read and modify tenant B's data by changing a parameter.

Two structural aggravators:
1. **tenant-service has no authentication middleware at all** — every route,
   including audit logs and cross-tenant usage, is anonymous.
2. **review, search, promotion, vendor services have no role enforcement** — any
   authenticated user can hit staff/admin mutations (moderation, coupon
   creation, vendor approval, search reindex).

There is no API gateway; the browser reaches Go services through a plain
Next.js rewrite (`/proxy/{service}`) that forwards a client-supplied
`X-Tenant-ID`. Tenant identity must therefore be enforced **inside each
service from the JWT**, never trusted from transport.

## Standard fix pattern

- Handlers: `tenantID := sharedmiddleware.GetTenantID(c)` (and `GetUserID`);
  reject with 401 when empty. Delete query/body/header tenant sources. Drop
  `TenantID`/`UserID` from request DTOs (tag `json:"-"`) so they can't be bound.
- Repositories: add `tenant_id = ?` (GORM) / `tenant_id` match (Mongo) to every
  `GetByID`/`Update`/`Delete`/`UpdateStatus` on tenant-scoped tables.
- Staff mutations: add `RequireRole(...)` on the route.
- tenant-service: register `sharedmiddleware.Auth` and derive tenant from JWT.

## Remediation status (2026-07-06)

All 16 backend services are fixed on branch `claude/remaining-tasks-x3aeu2`
(one commit per service). Every handler now derives the tenant (and user)
from the verified JWT, request DTOs no longer bind `tenant_id`/`user_id`,
by-id repository lookups carry a tenant predicate (404 cross-tenant), and
staff mutations are gated with `RequireRole`. The 12 Go services were each
built and tested green. The 2 .NET services (payment, inventory) were edited
to the same pattern but **not compiled here** (no dotnet SDK) — the per-service
CI must run `dotnet build`/`dotnet test` to confirm. The frontend BFF item
remains open and is tracked as B03 (JWT → HttpOnly cookie / server-side
tenant enforcement).

## Status legend
✅ fixed on branch · ⬜ open

---

## cart-service — ✅ FIXED
All 5 handlers read `tenant_id`/`user_id` from query/body despite JWT auth →
full cross-user/cross-tenant cart read & write. Fixed: handlers derive both
from JWT; `AddItemRequest.TenantID/UserID` are `json:"-"`; 401 when unauth.

## tenant-service — ✅ FIXED (was CRITICAL)
No `Auth` middleware anywhere. Unauthenticated exposure of:
- `GET /api/v1/audit-logs`, `/audit-logs/:id` — any tenant's audit logs (tenant_id optional in repo `GetByID`)
- `GET /api/v1/admin/usage` — per-tenant usage for all tenants
- full `/api/v1/tenants` CRUD — create/read/update/config/delete tenants
Fix: add Auth; require tenant from JWT; make repo `tenant_id` mandatory.

## payment-service (.NET) — ✅ FIXED (was CRITICAL)
All 14 `[Authorize]` endpoints derive tenant from `[FromQuery] tenantId`, body,
or by-PK lookups with no tenant predicate. Financial data (amounts, gateway
txn ids, saved payment methods) fully cross-tenant. Refund/cancel by id alone.
Repo `GetByIdAsync`/`GetByIdWithDetailsAsync`/`GetByIdempotencyKeyAsync`/
`GetByGatewayTransactionIdAsync` lack tenant filters. Fix: tenant from JWT
claims; add tenant predicate to every lookup.

## order-service — ✅ FIXED (was HIGH)
No authenticated handler calls GetTenantID. `GET /tenants/:tenantId/orders`
trusts the path; `GET /customers/:customerId/orders` has no tenant filter;
all state mutations (`/orders/:id/{confirm,cancel,ship,deliver,items}`) load by
id alone. `POST /orders/:id/send-receipt` emails any order's contents to a
caller-supplied address (exfil). Guest `POST /orders` and `GET /orders/:id`
are intentionally public (INFO). Fix: assert `order.TenantID == JWT` on load.

## inventory-service (.NET) — ✅ FIXED (was HIGH)
Same pattern: create/list from body/query tenant; get/update/delete/adjust/
transfer/reserve/fulfill by id alone. Cross-tenant stock tamper & DoS. Repo
`GetByIdAsync`/`GetByWarehouseAsync` lack tenant filters. Fix: tenant from JWT.

## user-service — ✅ FIXED (was HIGH)
Wishlist is clean (already fixed). But `GET/PUT /users/:id`,
`PATCH /users/:id/role`, `PATCH /users/:id/status` act on path id with no
tenant check, and repo `GetByID`/`Update`/`Delete`/`UpdatePassword` key on id
alone — a tenant-A admin can read/edit/promote tenant-B users. `ListUsers`
is clean (JWT tenant). Fix: scope user CRUD to JWT tenant.

## product-service — ✅ FIXED (was HIGH)
Write routes have Auth+RequireRole but `CreateProduct`/`CreateCategory` take
tenant from body, and `Update`/`Delete`/`UpdateStatus` (product & category)
filter `_id` only → cross-tenant write IDOR. Public `GetProduct`/`GetCategory`
read any tenant's draft/archived items. Image routes are CLEAN (service-layer
tenant check). Fix: JWT tenant on create; `_id AND tenant_id` on mutations;
scope public reads to header tenant + active status.

## review-service — ✅ FIXED (was HIGH)
No role enforcement at all. `CreateReview` trusts body tenant/user (spoof
author). `UpdateReview`/`DeleteReview` check ownership via `c.Query("user_id")`
(attacker-controlled; empty = admin → delete any review). `ModerateReview`/
`RespondToReview` unprotected. Repo `GetByID`/`Update` `_id` only. Fix: JWT
identity, RequireRole on moderation, tenant-scoped repo lookups.

## search-service — ✅ FIXED (was HIGH)
`POST /search/reindex` trusts body `TenantID`/`ID`, no role → any user can
overwrite/pollute another tenant's ES documents. Search/autocomplete reads are
tenant-scoped (INFO). Fix: RequireRole, force JWT tenant, verify doc ownership.

## promotion-service — ✅ FIXED (was HIGH)
No roles. `CreatePromotion`/`CreateCoupon` trust body tenant; `CreateCoupon`
attaches to any promotion (unscoped `GetPromotionByID`). `ProcessLoyaltyPoints`
credits/redeems points for any user in any tenant (financial). `GetPromotion`/
`GetLoyaltyAccount` cross-tenant IDOR. Fix: JWT tenant/user, RequireRole,
tenant-scoped lookups.

## vendor-service — ✅ FIXED (was HIGH)
No roles. `RegisterVendor` trusts body tenant; `UpdateVendor`/
`UpdateVendorStatus` (approve/suspend) load by id alone. `GetVendor`/
`GetVendorOrders`/`GetVendorAnalytics` cross-tenant reads of vendor PII &
revenue. Fix: JWT tenant, RequireRole on status, tenant-scoped repo.

## analytics-service — ✅ FIXED (was HIGH)
Auth is global (good) but all report handlers take tenant from query; `GetReport`
has no tenant filter → any custom report by id. `POST /reports` trusts body
tenant. Fix: JWT tenant; add tenant filter to `GetReport`.

## config-service — ✅ FIXED (was HIGH)
Reads are public by design, which leaks tenant data: `/config/search`
(`ILIKE` over all tenants' values incl. secrets), `/config/audit[/:id]`
(no tenant filter), `/config/get`/`export` (client tenant). `DELETE /config/:id`
and menu update/delete act by id with no tenant check. Fix: move audit/search/
export/get behind Auth; add tenant filters; ownership-check menu mutations.

## notification-service — ✅ FIXED (was HIGH)
Auth global, but tenant from query/`X-Tenant-Id`. `GET /notifications/user/:id`
& `/preferences/:id` cross-tenant (message content/PII). `GetByID`/`MarkAsRead`
`_id` only. Template `GetTemplate`/`Update`/`Delete` `{_id}` only;
`test-send` renders a template to a caller-supplied email (exfil). Fix: JWT
tenant everywhere; tenant-scoped template & notification lookups.

## shipping-service — ✅ FIXED (was HIGH)
Auth global, tenant from query. `GET /shipments/:id` (ship-to PII) and
status/cancel mutations load by id with no tenant filter. Repo `GetByID`/
`GetByIDWithDetails`/`GetByTrackingNumber` unscoped. Fix: JWT tenant; scope
`GetByID`; ownership-check mutations.

## recommendation-service — ✅ FIXED (was MEDIUM)
Auth global, tenant from query/body. `GET /recommendations/user/:id` leaks
per-user behavior cross-tenant; `POST /train` trusts body tenant (resource
abuse); `GetTrainingJob` unscoped. Lower data sensitivity. Fix: JWT tenant.

## frontend BFF — ⬜ OPEN (LOW)
`/api/upload` ignores tenant (files land in shared folders, not partitioned);
`/api/media` serves unpartitioned public media. No cross-tenant read of private
data, but no tenant partitioning either. demo-token route is CLEAN. The core
issue is the direct `/proxy/{service}` rewrite forwarding a client `X-Tenant-ID`
— see B03 (BFF refactor) for the durable fix.
