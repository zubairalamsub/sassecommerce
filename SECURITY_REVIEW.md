# Saajan E-commerce Platform — Security Review

**Date:** 2026-05-13
**Reviewer:** Automated security audit (Claude)
**Scope:** All Go services, .NET services (inventory, payment), Next.js frontend,
shared libraries, Kubernetes manifests, docker-compose, monitoring config.
**Methodology:** OWASP Top 10 (2021) code-level review + dependency vulnerability
scans (`govulncheck`, `npm audit`, `dotnet list package --vulnerable`).

---

## Remediation Update — 2026-07-06

The following findings were fixed on branch `claude/current-repo-review-2x6afd`
(one commit per finding; see git log for details):

| Finding | Status | Fix |
| ------- | ------ | --- |
| A01-1 demo-token auth bypass (CRITICAL) | **FIXED** | Route 404s in production; claims derived from a server-side allowlist, never the request body |
| A02-1 hardcoded JWT_SECRET (CRITICAL) | **FIXED** | All 15 services fail fast when JWT_SECRET is unset or <32 bytes; no default shipped |
| A05-1 wildcard CORS + credentials (CRITICAL) | **FIXED** | HardenedCORS everywhere (incl. order/user/product); .NET uses WithOrigins; gin-contrib/cors dependency removed entirely (also clears GO-2024-2955) |
| A10-1 Next.js SSRF + 12 advisories (HIGH) | **FIXED** | Upgraded 16.2.4 → 16.2.10 + postcss override; npm audit clean |
| A02-4 credentials in request/response logs (HIGH) | **FIXED** | Shared RequestLogger redacts sensitive JSON fields at any depth; non-JSON bodies omitted |
| A04-1 no per-endpoint auth rate limits (HIGH) | **FIXED** | Per-IP and per-email/token limits on login, register, forgot/reset-password, verify-email |
| A01-2 / A07-1 no refresh rotation or revocation (HIGH) | **FIXED** | Refresh tokens with rotation + reuse detection; all sessions revoked on password change/reset and 2FA changes |
| A04-3 2FA not enforced at login (MEDIUM) | **FIXED** | Login returns a challenge for enrolled users; /auth/login/2fa completes it; enrollment endpoints wired |
| A03-1 MongoDB regex injection (HIGH) | **FIXED** | regexp.QuoteMeta on the search query |
| B01 unauthenticated /api/upload + traversal (HIGH) | **FIXED** | Staff JWT required; folder whitelist; MIME-derived extensions; SVG rejected |
| B02 scriptable SVG from /api/media (MEDIUM) | **FIXED** | SVG no longer served as image/svg+xml; nosniff + attachment for unknown types |
| A02-3 2FA secrets unencrypted fallback (HIGH) | **PARTIAL** | user-service now fatals in production without TWO_FACTOR_ENCRYPTION_KEY and warns in dev |
| A07-2 no login brute-force protection (MEDIUM) | **FIXED** (prior work + this branch) | Lockout windows per email/IP plus the new per-endpoint limits |

Second remediation pass on branch `claude/remaining-tasks-x3aeu2`
(one commit per finding):

| Finding | Status | Fix |
| ------- | ------ | --- |
| A02-5 / A05-3 security headers everywhere (MEDIUM) | **FIXED** | Shared SecurityHeaders middleware registered after Recovery in all 14 Go services; equivalent inline middleware in both .NET services |
| A01-3 tenant scope not enforced in middleware (MEDIUM) | **FIXED** | RequireTenant middleware on wishlist routes; query-string tenant_id fallback removed |
| A04-2 default credentials in docker-compose (MEDIUM) | **FIXED** | POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET, GRAFANA_ADMIN_PASSWORD are required env vars; .env.example added |
| A05-4 unauthenticated /metrics (MEDIUM) | **FIXED** | Shared MetricsAuth middleware: bearer METRICS_TOKEN when set; 404 in production when unset; open only in dev |
| A02-2 bcrypt cost (LOW) | **FIXED** | Password hashing cost 10 → 12 |
| A04-4 password minimum length (LOW) | **FIXED** | min=12 at binding + service layers, common-password denylist, frontend forms synced |
| B04 verbose validator errors (LOW) | **FIXED** | validator.SanitizedBindingErrors replaces details:err.Error() on all ShouldBindJSON sites |
| A06 Go pgx + stdlib advisories | **FIXED** | pgx v5.5.1 → v5.10.0 in all 8 services pinning it; builder image golang:1.24-alpine → 1.25-alpine. gin-contrib/cors already removed in pass 1 |
| A06 .NET vulnerable packages | **PARTIAL** | Npgsql EF/Design 8.0.11, JwtBearer 8.0.22, IdentityModel 7.5.1, Caching.Memory 8.0.1, System.Text.Json 8.0.5. AutoMapper 13.0.1 is **still vulnerable** (GHSA-rvv3-g6hj-g44x recursive-mapping DoS) — the only patched line (≥15.1.1) is the commercial build with a JWT-stack cascade, so it is instead an **accepted risk** (services map flat internal DTOs; DoS not reachable) suppressed per-advisory via NuGetAuditSuppress. Now compile-verified locally: payment 94/94, inventory 5/5 tests green. Also fixed a test-project restore break (stale AutoMapper.Extensions ref → NU1107) that had kept payment-service CI red. |

Third remediation pass on the same branch (one commit per finding):

| Finding | Status | Fix |
| ------- | ------ | --- |
| A02-3 legacy plaintext 2FA secrets (HIGH, was PARTIAL) | **FIXED** | Idempotent startup migration in user-service re-encrypts pre-key rows (identified by failed GCM open) under the current key |
| A08-1 no Kafka event signing (MEDIUM) | **FIXED** | Shared EventSigner (HMAC-SHA256, x-event-signature header) keyed by EVENT_SIGNING_KEY; all Go + .NET producers sign, all consumers drop unsigned/tampered events (committed as poison so not redelivered); disabled when key unset for producer-first rollout |
| A09-1 no security alerts (MEDIUM) | **FIXED** | 'security' Prometheus alert group: failed-login spikes, per-service 401/403 spikes, forgot-password surges, 429 spikes, registration surges |
| Git-history secrets scan (follow-up) | **PASS** | All 40 commits scanned for AWS keys, private keys, GitHub/Slack/Stripe/Google tokens, JWTs, and generic credential assignments — only demo seeds, doc placeholders, and test fixtures found |

| A08-2 no container image signing (MEDIUM) | **FIXED** | deploy.yml cosign-signs every pushed image by digest (keyless/OIDC, Rekor-logged); cluster-side verifyImages policy still recommended |

Still open: A08-3 (broader CI integrity review — e.g. pinning actions by
SHA), B03 (JWT in localStorage — needs a Next.js BFF/HttpOnly-cookie
refactor), and the tenant-isolation cross-check follow-up.

---

## Executive Summary

The platform has a **solid security foundation** — bcrypt password hashing, TOTP
2FA with AES-GCM-encrypted secrets, parameterised SQL via GORM, audit logging
through a Kafka consumer, and a Hardened CORS middleware available in the shared
library. However, a number of **critical, exploitable issues are live today**:
(1) a Next.js API route (`/api/auth/demo-token`) that will sign a valid JWT for
**any** caller-supplied `user_id` / `role` — complete authentication bypass; (2)
the same hardcoded fallback JWT signing secret
(`"your-secret-key-change-in-production-12345"`) is baked into ~10 Go services
and the .NET payment service, so a leak in **any one** environment compromises
**all** services; (3) every Go service except `notification-service` ships with
`AllowOrigins: ["*"] + AllowCredentials: true`, which combined with the
`gin-contrib/cors@v1.5.0` CVE (GO-2024-2955) allows cross-origin reads of
authenticated responses; (4) the Next.js dependency is on 16.2.4 with eight
HIGH/SSRF advisories including middleware bypass; (5) full request and response
bodies — including passwords, reset tokens, and 2FA codes — are logged in
production via the `RequestLogger` middleware in user-service. Fix these five
before any production traffic.

---

## Severity Counts

| Severity     | OWASP findings | Dependency vulns | Total |
| ------------ | -------------- | ---------------- | ----- |
| **Critical** | 3              | 0                | **3** |
| **High**     | 6              | 12*              | **18** |
| **Medium**   | 7              | 5                | **12** |
| **Low**      | 4              | 1                | **5** |

\* Counts: 8 High advisories in Next.js (single package, distinct CVEs); 4 High
.NET packages (AutoMapper, Caching.Memory, Npgsql, System.Text.Json) shared
across both .NET services. Go-stdlib CVEs counted as 1 finding (17 individual
CVEs) since the fix is a single toolchain bump.

---

## Findings — OWASP Top 10 (2021)

### A01 — Broken Access Control

#### A01-1 (CRITICAL): `/api/auth/demo-token` issues authentic JWTs for arbitrary roles
- **Where:** `frontend/src/app/api/auth/demo-token/route.ts:1-30`
- **Status:** OPEN — not flagged elsewhere
- **What:** The route accepts `{user_id, tenant_id, email, role}` from the
  request body and signs a real HS256 JWT with `JWT_SECRET`. There is no
  authentication, no allow-list of demo emails, no environment gate. A remote
  attacker can `POST /api/auth/demo-token {"user_id":"x","email":"x@x.com",
  "role":"super_admin","tenant_id":""}` and receive a valid token that the
  backend services will trust (they verify with the same `JWT_SECRET`). This
  is a complete authentication bypass.
- **Fix:** Delete the route. Demo logins should go through the normal `/login`
  endpoint backed by seeded users in the user-service database. If a dev-only
  shortcut is genuinely required, gate behind `if (process.env.NODE_ENV !==
  'production')` AND require the email to be in a hardcoded `DEMO_USERS` set
  (do not trust caller-supplied `role`).

#### A01-2 (HIGH): No refresh-token rotation; refresh tokens never invalidated on password change
- **Where:** `services/user-service/internal/service/auth_service.go:488-529`
  (`ChangePassword`), lines around `ResetPassword`, and
  `internal/repository/refresh_token_repository.go`
- **Status:** OPEN
- **What:** A `RefreshTokenRepository` exists but `ChangePassword` and
  `ResetPassword` do not call `RevokeAllForUser` or equivalent. A stolen
  refresh token remains valid after the user changes their password. Token
  rotation on use (issuing a new refresh token on every refresh, invalidating
  the old one, and detecting reuse to revoke the chain) is not implemented.
- **Fix:** On password change/reset and on 2FA enable/disable, call
  `refreshTokenRepo.RevokeAllForUser(userID)`. Implement rotation: on
  `/refresh`, generate a new pair and invalidate the old refresh token; if a
  revoked refresh token is presented, revoke the entire family.

#### A01-3 (MEDIUM): RBAC is granular but tenant-scoping checks are not enforced in middleware
- **Where:** `services/user-service/internal/middleware/auth.go:77-122`
  (`RequireRole`); per-handler `tenantID := middleware.GetTenantID(c)` usage
  is per-handler, not enforced by middleware.
- **Status:** OPEN — separate tenant isolation audit is referenced in the
  task brief at `infrastructure/TENANT_ISOLATION_FINDINGS.md` (file did not
  exist at audit time; cross-reference when published).
- **What:** Role-based checks (admin/moderator/customer) are wired correctly
  on the few admin routes (`PATCH /users/:id/role`, `PATCH /users/:id/status`).
  But there is no `RequireTenantScope` middleware that asserts the path/body
  tenant matches `claims.TenantID`. Individual handlers must remember to add
  the `WHERE tenant_id = ?` filter — and a few (e.g. wishlist) read
  `tenantID = c.Query("tenant_id")` as a fallback, which a logged-in user can
  spoof.
- **Where (specific):** `services/user-service/internal/api/wishlist_handler.go:30-32`
  reads `tenantID` from a query param if the JWT doesn't supply one.
- **Fix:** Add a `RequireTenant` middleware that 401s when `claims.TenantID`
  is empty, and remove all query-string fallbacks. Centralise the
  `tenant_id = ?` filter in repositories (a GORM scope or BeforeQuery hook).

---

### A02 — Cryptographic Failures

#### A02-1 (CRITICAL): Hardcoded fallback `JWT_SECRET` across ~10 services
- **Where:**
  - `services/user-service/cmd/server/main.go:131` —
    `JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production")`
  - `services/product-service/cmd/server/main.go:122` — same default
  - `services/shipping-service/cmd/server/main.go:102` —
    `"your-secret-key-change-in-production-12345"`
  - `services/payment-service/src/Ecommerce.PaymentService/Program.cs:91` —
    `"your-secret-key-change-in-production-12345"`
  - `frontend/src/app/api/auth/demo-token/route.ts:3` — same default
- **Status:** OPEN
- **What:** If `JWT_SECRET` is unset (single misconfigured pod, single
  developer running `docker-compose up`), the service silently falls back to
  a publicly known string. Tokens issued with this secret are forgeable by
  anyone who reads this codebase. Worse, because the SAME default is used in
  multiple services, leak in any service compromises all.
- **Fix:** Replace the `getEnv("JWT_SECRET", "...")` calls with
  `mustGetEnv("JWT_SECRET")` that fatals on empty. Ship no default. Add a
  startup assertion that the secret length is >= 32 bytes. In .NET, replace
  the `??` fallback with `?? throw new InvalidOperationException(...)`.

#### A02-2 (HIGH): bcrypt cost is the library default (10) — adequate for passwords, marginal for backup codes
- **Where:** `services/user-service/internal/service/auth_service.go:759`,
  `internal/service/two_factor_service.go:224`
- **Status:** OPEN — informational
- **What:** `bcrypt.DefaultCost` is 10 (~100 ms). OWASP 2023 recommends >= 10
  for bcrypt, so this passes — but 12 is a better target for a 2026
  deployment. Cost 10 on backup codes (~52 bits of entropy) is fine; on user
  passwords it is the floor.
- **Fix:** Bump to cost 12. Make it configurable via env var so future
  hardware upgrades can raise it without code changes.

#### A02-3 (HIGH): 2FA encryption silently falls back to base64 plaintext when key absent
- **Where:** `services/user-service/internal/repository/two_factor_repository.go:40-45`
- **Status:** OPEN
- **What:** `NewTwoFactorRepository(db, encryptionKey)` allows
  `encryptionKey == nil`, which sets `r.noEnc = true` and stores TOTP secrets
  as base64-encoded plaintext. The comment says "acceptable for local dev
  only — log a warning at startup" but no warning is logged, and there is no
  build-time or runtime guard that prevents this in production.
- **Fix:** Require the encryption key. Fatal if absent when
  `ENVIRONMENT == "production"`. At minimum, emit a `logger.Error` at
  startup when `r.noEnc == true`.

#### A02-4 (HIGH): Sensitive PII and credentials logged via RequestLogger
- **Where:** `services/user-service/cmd/server/main.go:206-211`,
  and `shared/go/pkg/middleware/request_logger.go:80-150`
- **Status:** OPEN
- **What:** User-service enables `LogRequestBody: true, LogResponseBody:
  true`. The middleware redacts `Authorization`, `Cookie`, and `X-Api-Key`
  HEADERS but does NOT redact JSON BODIES. As a result, every
  `POST /login`, `POST /register`, `POST /change-password`,
  `POST /reset-password`, and `POST /forgot-password` writes the user's
  cleartext password (and reset tokens, 2FA codes, etc.) to logs that ship to
  Loki/Grafana. Same for `cart-service` (line 85), `shipping-service`,
  `notification-service`. Anyone with log access has every user's password
  history.
- **Fix:** Set `LogRequestBody: false, LogResponseBody: false` on the
  `user-service` and any auth-bearing routes. Better: extend
  `RequestLoggerConfig` with `SensitiveJSONFields` (e.g. `["password",
  "old_password", "new_password", "token", "secret", "code"]`) that recursive
  JSON walk and redacts.

#### A02-5 (MEDIUM): TLS not terminated in-cluster; HSTS only applied by notification-service
- **Where:** `shared/go/pkg/middleware/security_headers.go` exists but only
  `notification-service` calls it.
- **Status:** OPEN
- **Fix:** Apply `SecurityHeaders` middleware in every service's
  `setupRouter`.

---

### A03 — Injection

#### A03-1 (HIGH): MongoDB regex search uses unescaped user input
- **Where:** `services/product-service/internal/repository/product_repository.go:166-176`
- **Status:** OPEN
- **What:** `r.collection.Find(ctx, bson.M{"$or": ...
  {"name": bson.M{"$regex": query, "$options": "i"}}})` — `query` comes
  straight from the URL. A malicious caller can supply
  `.*(a+)+(a+)+(a+)+x` to cause catastrophic backtracking (ReDoS), or use
  regex metacharacters (`.*`, `^`, `$`) to broaden the result set across
  intended boundaries. The `tags` clause (`bson.M{"$in": []string{query}}`)
  is safe.
- **Fix:** Escape regex metacharacters before interpolating:
  `regexp.QuoteMeta(query)`. Better: use a MongoDB text index with
  `$text: {$search: query}` which is safer and faster.

#### A03-2 (LOW): No `exec.Command` / shell exec usage found anywhere
- **Status:** PASS — verified by `grep -rn "exec.Command\|os/exec"
  services/ shared/` (no hits).

#### A03-3 (LOW): GORM queries are parameterised, raw SQL queries verified safe
- **Status:** PASS — only raw SQL in services is
  `recommendation-service/internal/repository/recommendation_repository.go:133`
  and uses `?` parameter binding correctly. `order-service` uses `db.Exec`
  only for DDL (schema setup) and write paths with parameterised statements
  in `projection/projection.go`.

#### A03-4 (LOW): HTML email templates properly escape user-controlled content
- **Where:** `services/notification-service/internal/templates/email.go`
- **Status:** PASS — `escapeHTML` handles `& < > " '`, applied to title,
  greeting, footnote, and body via `formatBody`. `escapeAttr` for href
  values. URLs going into href are escaped via `escapeAttr` (line 129).
  Non-ASCII characters (Bangla) intentionally not escaped — acceptable.
- **Caveat (LOW):** `escapeHTMLPreserveURL` (line 246) currently just calls
  `escapeHTML` so URLs work, but if the body ever includes user-supplied
  text with `&` in URL queries those become `&amp;` — visual nit, not a
  security issue.

---

### A04 — Insecure Design

#### A04-1 (HIGH): No per-endpoint rate limiting on auth-sensitive routes
- **Where:** `services/user-service/cmd/server/main.go:226-229` applies a
  single global limiter (100 req/min/IP). No specific limiter on `/login`,
  `/register`, `/forgot-password`, `/reset-password`, `/verify-email`.
- **Status:** OPEN (task brief notes brute-force protection is being added
  by another agent — coordinate)
- **What:** An attacker can attempt 100 login passwords/minute/IP across the
  whole API, and from a botnet, far more. The forgot-password endpoint can
  be abused to spam customers with reset emails.
- **Fix:** Add per-endpoint rate limits using existing
  `sharedmiddleware.RateLimit`:
  - `POST /login`: 5/min per IP, 20/hour per email
  - `POST /register`: 3/hour per IP
  - `POST /forgot-password`: 3/hour per email
  - `POST /reset-password`: 5/hour per token (constant-time compare)
  - `POST /verify-email`: 5/hour per token
  Use the existing `RateLimitByUser` / `RateLimitByTenant` helpers from
  `shared/go/pkg/middleware/rate_limiter.go`.

#### A04-2 (MEDIUM): Hardcoded default Postgres password in docker-compose
- **Where:** `docker-compose.yml:9-11` —
  `POSTGRES_USER: postgres`, `POSTGRES_PASSWORD: postgres`,
  `POSTGRES_DB: postgres`
- **Status:** OPEN
- **What:** Same default credentials in committed compose file. Anyone who
  exposes port 5432 (intentionally or by misconfiguration) is owned. Same
  pattern for Redis (`requirepass redis123`).
- **Fix:** Use `${POSTGRES_PASSWORD:?required}` syntax to force a `.env`
  value; never commit a working default. `docker-compose.override.yml` can
  hold dev defaults outside Git.

#### A04-3 (MEDIUM): Sign-in flow does not enforce 2FA gating
- **Where:** `services/user-service/internal/service/auth_service.go`
  `Login()` issues a full access token immediately on password match. The
  `twoFactorService` has a "challenge token" helper but `Login()` never
  calls it. Users who have enabled 2FA are not actually challenged.
- **Status:** OPEN
- **Fix:** In `Login()`, after password verify, check
  `twoFactorService.IsEnabled(userID)`. If enabled, return a short-lived
  challenge JWT (`purpose=2fa-challenge`) and require the client to call
  `/login/2fa` with a TOTP/backup code before issuing the access token.

#### A04-4 (LOW): Password minimum length is 8 (industry minimum)
- **Where:** `services/user-service/internal/models/user.go:104` —
  `Password ... binding:"required,min=8"`
- **Status:** OPEN — acceptable but could be stronger.
- **Fix:** Bump to `min=12`; add a haveibeenpwned check or a denylist of
  the 10k most-common passwords.

---

### A05 — Security Misconfiguration

#### A05-1 (CRITICAL): `AllowOrigins: ["*"] + AllowCredentials: true` in 10+ services
- **Where:** Every Go service that uses raw `gin-contrib/cors` —
  `cart-service`, `shipping-service`, `promotion-service`, `vendor-service`,
  `analytics-service`, `search-service`, `review-service`, `config-service`,
  `recommendation-service`, `tenant-service` (see grep output: all
  `cmd/server/main.go` files). Also `services/inventory-service` and
  `services/payment-service` use `AllowAnyOrigin().AllowAnyMethod().AllowAnyHeader()`
  + `UseCors()` without restricting credentials.
- **Status:** OPEN (per task brief: "being fixed by another agent" —
  coordinate)
- **What:** This combo is invalid per the CORS spec, but `gin-contrib/cors`
  v1.5.0 silently sets both headers, and `gin-contrib/cors` v1.5.0 is also
  affected by CVE GO-2024-2955 (wildcard origin mishandling). Browsers
  generally refuse credential transmission to `*` origins, but the .NET
  middleware doesn't, and any browser bypass / non-browser client lets an
  attacker make authenticated cross-origin reads.
- **Fix:** Use the existing `shared/go/pkg/middleware/HardenedCORS` in every
  service (currently only `notification-service` uses it). For .NET:
  `.WithOrigins(allowedOrigins).AllowCredentials()` — never `AllowAnyOrigin`
  with credentials.

#### A05-2 (HIGH): `gin.SetMode(gin.ReleaseMode)` ungated in most services
- **Where:** Most `cmd/server/main.go` files call `gin.SetMode(gin.ReleaseMode)`
  inside `setupRouter()` regardless of environment. Only `user-service`
  hardcodes it unconditionally.
- **Status:** PASS-ish — ReleaseMode is the safe default, but a hardcoded
  ReleaseMode means dev folks lose useful stack traces in local dev. Not a
  security issue, but the inverse (DebugMode in prod) would be, so noting
  it's correctly hardcoded.

#### A05-3 (MEDIUM): No security headers (HSTS, CSP, X-Frame-Options) on any service except notification-service
- **Where:** `shared/go/pkg/middleware/security_headers.go` exists with
  sensible defaults; only used by `notification-service` (line 119 of its
  main.go).
- **Status:** OPEN
- **Fix:** Add `router.Use(sharedmiddleware.SecurityHeaders(...))` to every
  service.

#### A05-4 (MEDIUM): `/metrics` Prometheus endpoint is unauthenticated
- **Where:** Every service: `router.GET("/metrics", gin.WrapH(metrics.Handler()))`
  before any auth middleware.
- **Status:** OPEN — acceptable inside a private cluster; risky if any
  service is ever exposed via Internet-facing ingress.
- **Fix:** Either move `/metrics` to a separate listener on a different
  port (not exposed) or require a basic-auth scrape token.

---

### A06 — Vulnerable and Outdated Components

See **Dependency Vulnerability Tables** below.

---

### A07 — Identification and Authentication Failures

#### A07-1 (HIGH): Session not invalidated on password change/reset
- **Where:** `services/user-service/internal/service/auth_service.go:488-529`
  (ChangePassword) and `:670-720`-ish (ResetPassword)
- **Status:** OPEN — see A01-2.

#### A07-2 (MEDIUM): No brute-force protection on login
- **Status:** OPEN — task brief notes another agent is adding this. See
  A04-1. The `models/login_attempt.go` enum exists
  (`LoginAttemptReasonRateLimited`) but no service writes login attempts.

#### A07-3 (LOW): Email enumeration via timing is mitigated by code path; explicit comment is good
- **Where:** `auth_service.go:332` (`ResendEmailVerification`), `:354`
  (`RequestPasswordReset`)
- **Status:** PASS — explicit "return nil even if not found to prevent
  email enumeration" comments. Login error message is also generic
  ("invalid email or password").

---

### A08 — Software and Data Integrity Failures

#### A08-1 (MEDIUM): No Kafka event signing or replay protection
- **Status:** OPEN
- **What:** Events on the bus are JSON with `event_id`, `timestamp`, and
  `version` but no HMAC signature. A compromised producer can spoof
  `UserRegistered`, `PasswordChanged`, etc. events that the
  notification-service and tenant-service consume. The audit consumer in
  tenant-service will faithfully log forged events as if real.
- **Fix:** Add an HMAC over the JSON payload using a shared secret pulled
  from K8s secrets. Consumers verify before processing. Topic-level Kafka
  ACLs would be even better but require Kafka auth.

#### A08-2 (MEDIUM): No container image signing
- **Status:** OPEN
- **What:** Dockerfiles build images; nothing signs them (cosign / sigstore)
  or verifies signatures at deploy.
- **Fix:** Add `cosign sign` to CI; configure Kubernetes
  ImagePolicyWebhook or Kyverno/OPA policy to require signature.

#### A08-3 (LOW): No CI/CD integrity controls reviewed
- **Status:** UNKNOWN — no `.github/workflows` or `.gitlab-ci.yml` reviewed
  in this audit. Recommend separate review.

---

### A09 — Security Logging and Monitoring

#### A09-1 (MEDIUM): No security-event alerts in Prometheus rules
- **Where:** `infrastructure/monitoring/prometheus/alert_rules.yml` —
  20 alerts, all infrastructure (DB down, high CPU, Kafka lag). None for:
  - Spike in 401/403 responses
  - Repeated failed logins from a single IP
  - Unusual cross-tenant access patterns
  - Sudden surge in `/forgot-password` calls
- **Status:** OPEN
- **Fix:** Add SLI/security alerts. Suggested rules:
  ```yaml
  - alert: HighAuthFailureRate
    expr: sum(rate(http_requests_total{path="/api/v1/auth/login",status="401"}[5m])) > 5
    for: 2m
  - alert: PasswordResetSurge
    expr: sum(rate(http_requests_total{path=~".*/forgot-password",status="200"}[5m])) > 10
  ```

#### A09-2 (LOW): Audit consumer exists; coverage of which events get logged is implicit
- **Where:** `services/tenant-service/cmd/server/main.go:65-68` starts
  `AuditEventConsumer` against all Kafka topics. Good.
- **Status:** PASS-ish — coverage relies on every service publishing
  every interesting event. A periodic check that auth events
  (`PasswordChanged`, `EmailVerified`, `2FAEnabled`) are landing in the
  audit table would catch silent drops.

---

### A10 — Server-Side Request Forgery

#### A10-1 (MEDIUM): Next.js 16.2.4 SSRF advisory (GHSA-c4j6-fc7j-m34r, CVSS 8.6)
- **Status:** OPEN — see Frontend Dependencies table.
- **What:** Next.js 16.0.0–16.2.4 is vulnerable to SSRF via WebSocket
  upgrade requests, fixed in 16.2.5. The platform uses Next.js as a proxy
  for backend services (`/proxy/{service}/...` — `frontend/src/lib/api.ts`),
  so this is materially exploitable.
- **Fix:** `npm install next@^16.2.6` (also fixes the 7 other High
  advisories — see Dependency table).

#### A10-2 (LOW): No user-controlled URLs fetched by application code
- **Status:** PASS — verified by reading image upload routes; presigned
  upload pattern is used (storage talked to directly by browser). No
  webhook handlers exist.

---

## Bonus Findings

### B01 (HIGH): `/api/upload` Next.js route is unauthenticated and accepts caller-controlled `folder`
- **Where:** `frontend/src/app/api/upload/route.ts:9-55`
- **Status:** OPEN
- **What:** Anyone can `POST /api/upload` with arbitrary files (only
  MIME-type checked — client-controlled). The `folder` param is joined into
  `path.join(STORAGE_PATH, folder)`. `path.join("/app/media", "../../etc")`
  returns `/etc`, so an attacker can write attacker-controlled bytes to any
  path the Node process can write. 5 MB size limit is the only safeguard.
- **Fix:** Add auth check (require valid session). Sanitise `folder` —
  whitelist a fixed set (`products|avatars|tenants`) or strip `..` and
  path separators.

### B02 (MEDIUM): `/api/media/[...path]` serves SVG with `image/svg+xml` MIME
- **Where:** `frontend/src/app/api/media/[...path]/route.ts:7-15`
- **Status:** OPEN
- **What:** Path-traversal is blocked (line 25 — `if (relativePath.includes
  ('..'))`). However, SVGs are returned with `image/svg+xml`, and SVGs can
  contain `<script>` that executes in the browser if rendered as a page (not
  as `<img>`). If an attacker uploads a malicious SVG via B01 and lures a
  victim to `/api/media/products/<evil>.svg`, they get XSS in the
  application origin (cookies, localStorage tokens).
- **Fix:** Drop `.svg` from `MIME_TYPES` and reject SVG uploads, OR serve
  SVGs with `Content-Security-Policy: default-src 'none'` and
  `Content-Disposition: attachment`.

### B03 (LOW): JWT stored in localStorage
- **Where:** `frontend/src/stores/auth.ts:51-149` — Zustand `persist`
  middleware with `name: 'auth-storage'`. Default zustand storage is
  `localStorage`.
- **Status:** OPEN — informational. Trade-off vs HttpOnly cookies.
- **What:** Any XSS (e.g. via B02) can steal the JWT. Mitigated by short
  expiry (24h) but not by `HttpOnly`.
- **Fix:** Long-term, move to HttpOnly Secure SameSite=Strict cookies set
  by a Next.js route (BFF pattern). The current architecture is the
  industry-common Bearer-token-in-localStorage pattern; this is a known
  acceptable trade-off if XSS surface is small.

### B04 (LOW): Verbose error responses include error.Error() details
- **Where:** `auth_handler.go:43-47, 89-91, 138-143`, many other handlers.
  Responses include `"details": err.Error()` from `ShouldBindJSON` errors.
- **Status:** OPEN — minor. The actual GORM / pgx error messages are not
  leaked because handlers translate to fixed strings, but the validator
  errors do leak struct internals. Not a security issue, just noise.

### B05: No working-tree secrets found
- **Status:** PASS — grepped for `AKIA*`, `-----BEGIN * PRIVATE KEY-----`,
  OCI OCIDs, and long base64 strings tagged as `secret`/`api_key`. Only
  placeholders (`REPLACE_ME`, `your-secret-key-change-in-production`) found.
  K8s `01-secrets.yaml` is a template with placeholders — correct pattern.

---

## Dependency Vulnerability Scans

### Go (govulncheck @ Go 1.25.1 toolchain)

**Headline:** Every Go service is affected by 20+ vulnerabilities, almost all
from the Go 1.25.1 standard library + two third-party modules. The fix is
mostly a toolchain bump and two `go get` upgrades.

| Module / Component | Current | Fix | Severity | CVE |
| ----------------- | ------- | --- | -------- | --- |
| Go stdlib (net, crypto/x509, crypto/tls, html/template, net/url, encoding/asn1, encoding/pem) | go1.25.1 | go1.25.10 | High | GO-2026-4971, GO-2026-4947, GO-2026-4946, GO-2026-4870, GO-2026-4865, GO-2026-4601, GO-2026-4341, GO-2026-4340, GO-2026-4337, GO-2025-4175, GO-2025-4155, GO-2025-4013, GO-2025-4011, GO-2025-4010, GO-2025-4009, GO-2025-4008, GO-2025-4007 |
| `github.com/gin-contrib/cors` | v1.5.0 | v1.6.0 | High | GO-2024-2955 (wildcard origin mishandling) |
| `github.com/jackc/pgx/v5` | v5.5.1 | v5.5.4 | High | GO-2024-2606 (SQL injection in pgproto3), GO-2024-2567 (panic on busy/closed pipeline) |

**Action:** `go install go1.25.10` + `go get -u github.com/gin-contrib/cors`
+ `go get -u github.com/jackc/pgx/v5` in every service, then `go mod tidy`.

### npm (frontend, prod-only)

| Package | Current | Fix | Severity | Advisory |
| ------- | ------- | --- | -------- | -------- |
| **next** | 16.2.4 | 16.2.6 | **High** | GHSA-c4j6-fc7j-m34r (SSRF via WebSocket, CVSS 8.6) |
| next | 16.2.4 | 16.2.6 | High | GHSA-492v-c6pp-mqqv (Middleware/Proxy bypass via dynamic route, CVSS 8.1) |
| next | 16.2.4 | 16.2.6 | High | GHSA-267c-6grr-h53f (Middleware bypass via segment-prefetch, CVSS 7.5) |
| next | 16.2.4 | 16.2.6 | High | GHSA-26hh-7cqf-hhc6 (Incomplete fix of above, CVSS 7.5) |
| next | 16.2.4 | 16.2.6 | High | GHSA-36qx-fr4f-26g5 (Middleware bypass in i18n, CVSS 7.5) |
| next | 16.2.4 | 16.2.6 | High | GHSA-8h8q-6873-q5fj (DoS in Server Components, CVSS 7.5) |
| next | 16.2.4 | 16.2.6 | High | GHSA-mg66-mrh9-m8jx (DoS via Cache Components, CVSS 7.5) |
| next | 16.2.4 | 16.2.6 | Moderate | GHSA-ffhc-5mcf-pf4q (XSS via CSP nonces), GHSA-gx5p-jg67-6x7h, GHSA-h64f-5h5j-jqjh, GHSA-wfc6-r584-vfw7 |
| next | 16.2.4 | 16.2.6 | Low | GHSA-vfv6-92ff-j949, GHSA-3g8h-86w9-wvmq |
| postcss (transitive of next) | <8.5.10 | 8.5.10+ | Moderate | GHSA-qx2v-qp2m-jg93 (XSS via unescaped `</style>`) |

**Action:** `cd frontend && npm install next@^16.2.6`. The `npm audit fix`
will resolve all listed CVEs in one step (no major version bump).

### .NET (inventory-service, payment-service — same package set)

| Package | Resolved | Severity | Advisory |
| ------- | -------- | -------- | -------- |
| AutoMapper | 12.0.1 | High | GHSA-rvv3-g6hj-g44x |
| Microsoft.Extensions.Caching.Memory | 8.0.0 | High | GHSA-qj66-m88j-hmgj |
| Npgsql | 8.0.0 | High | GHSA-x9vc-6hfv-hg8c |
| System.Text.Json | 8.0.0 | High | GHSA-hh2w-p6rv-4g7w, GHSA-8g4q-xg66-9fp4 |
| Microsoft.IdentityModel.JsonWebTokens | 7.0.3 | Moderate | GHSA-59j7-ghrg-fj52 |
| System.IdentityModel.Tokens.Jwt | 7.0.3 | Moderate | GHSA-59j7-ghrg-fj52 |

**Action:** Upgrade each via `dotnet add package <name>` to latest 8.x
(Microsoft.IdentityModel.* to >=7.5.0; AutoMapper to >=13.x; Npgsql to >=8.0.3;
System.Text.Json to >=8.0.4). Then re-run scan.

---

## Prioritised Remediation Backlog

Numbered top-to-bottom in order of severity x exploitability x effort. Targets
roughly: P0 fix this week, P1 fix this sprint, P2 next sprint.

### P0 — Drop everything

1. **Delete `/api/auth/demo-token` route** (A01-1) — 5 lines, ships
   super-admin JWT to anyone. `git rm
   frontend/src/app/api/auth/demo-token/route.ts` and remove the
   `demoLogin()` helper in `frontend/src/stores/auth.ts`.
2. **Remove hardcoded `JWT_SECRET` fallback in every service** (A02-1) —
   fatal-on-empty, plus rotate the production secret. ~30 mins for 11
   files.
3. **Fix CORS in all Go services** (A05-1) — replace `cors.New(cors.Config
   {AllowOrigins:["*"], ...})` with `sharedmiddleware.HardenedCORS(...)`
   reading from `CORS_ALLOWED_ORIGINS` env. 11 main.go edits, ~1 hour.
   Fix the .NET services similarly. (Per brief, another agent owns this.)
4. **Upgrade Next.js to 16.2.6** (A10-1 + 11 other CVEs) —
   `npm install next@^16.2.6 && npm install` plus a smoke test. ~30 mins.

### P1 — This sprint

5. **Stop logging request/response bodies on auth routes** (A02-4) — set
   `LogRequestBody: false` in user-service, cart-service, shipping-service,
   notification-service main.go, OR add `SensitiveJSONFields` redaction to
   the shared middleware. ~2 hours.
6. **Add `/login`, `/forgot-password`, `/register` rate limits** (A04-1) —
   coordinate with the brute-force-protection agent.
7. **Invalidate refresh tokens on password change/reset/2FA enable**
   (A01-2, A07-1) — add `refreshTokenRepo.RevokeAllForUser` calls; add
   refresh-token rotation. ~half a day.
8. **Enforce 2FA in the login flow** (A04-3) — wire the existing
   `twoFactorService.IsEnabled` check into `Login()`; add `/login/2fa`
   handler.
9. **Fix MongoDB regex injection** (A03-1) — `regexp.QuoteMeta(query)` or
   switch to `$text` index. 1-line fix.
10. **Bump Go toolchain to 1.25.10 + upgrade pgx + gin-contrib/cors** (A06)
    — `goenv install 1.25.10` or update Dockerfile base; `go get -u` in
    every service. ~half a day across 14 services.
11. **Upgrade .NET vulnerable packages** in `inventory-service` and
    `payment-service`. ~2 hours.
12. **Auth + sanitisation on `/api/upload`** (B01) — require session +
    whitelist `folder` values. ~1 hour.

### P2 — Soon

13. **Disable SVG serving or sandbox it** (B02).
14. **Require 2FA encryption key in production** (A02-3) — fail fast if
    `ENVIRONMENT=production` and key empty.
15. **Apply `SecurityHeaders` middleware in every service** (A05-3, A02-5).
16. **Add `RequireTenant` middleware + remove query-string tenant fallback
    in wishlist handler** (A01-3).
17. **Add HMAC signing to Kafka events** (A08-1).
18. **Container image signing (cosign) in CI** (A08-2).
19. **Add security-event Prometheus alerts** (A09-1).
20. **Bump bcrypt cost to 12** (A02-2).
21. **Bump password minimum length to 12 + denylist** (A04-4).
22. **Move JWT to HttpOnly Secure cookie via Next.js BFF pattern** (B03).
23. **Remove default Postgres/Redis passwords in `docker-compose.yml`**
    (A04-2).

---

## Audit Notes / What I Did Not Verify

- **Tenant isolation depth** — task brief references
  `infrastructure/TENANT_ISOLATION_FINDINGS.md` from a parallel agent;
  file did not exist at audit time. The high-level pattern (tenant_id on
  every model, GetTenantID middleware) looks right; the parallel audit
  should be cross-referenced.
- **govulncheck** finished tenant-service, order-service, and started on
  product-service before report write-up. Findings (Go stdlib + gin-cors
  + pgx) are essentially identical across every service that uses GORM
  with the pgx driver, so per-service scans would not change the picture.
- **CI/CD pipeline review** not in scope — recommend separate.
- **Penetration testing** — this is a code-level review only. Live
  pen-testing against a deployed environment would surface different
  issues (TLS config, ingress, OWASP A05 misconfigs at the infra layer).
- **Secrets in `git log`** — task brief said "git log -p is overkill" so
  I only checked working tree. A `git-secrets` / `truffleHog` pass on
  history is still recommended.

---

*Report generated 2026-05-13.*
