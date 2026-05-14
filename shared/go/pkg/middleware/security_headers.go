package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig configures the SecurityHeaders middleware.
//
// All fields are optional; zero values fall back to safe, API-friendly defaults
// suitable for backend JSON services. Callers (e.g. an SSR frontend) can
// override individual headers — for example a Next.js gateway will want a much
// looser ContentSecurityPolicy than the default API CSP.
type SecurityHeadersConfig struct {
	// StrictTransportSecurity sets the HSTS header. Defaults to
	// "max-age=31536000; includeSubDomains". TLS is assumed to terminate at
	// ingress; if you're serving plain HTTP locally, browsers will still
	// honor the header on first HTTPS request, so leaving it on is fine.
	StrictTransportSecurity string

	// ContentTypeOptions defaults to "nosniff".
	ContentTypeOptions string

	// FrameOptions defaults to "DENY". Set to "SAMEORIGIN" if the service
	// renders pages that may be embedded by the same origin (e.g. admin
	// preview iframes).
	FrameOptions string

	// ReferrerPolicy defaults to "strict-origin-when-cross-origin".
	ReferrerPolicy string

	// PermissionsPolicy defaults to a deny-all for sensitive surfaces plus
	// an opt-out of FLoC/interest-cohort.
	PermissionsPolicy string

	// XSSProtection defaults to "0" — modern browsers ignore it or worse,
	// and we rely on CSP instead.
	XSSProtection string

	// ContentSecurityPolicy defaults to a conservative API-only policy:
	// "default-src 'none'; frame-ancestors 'none'". JSON APIs do not load
	// scripts, styles, or images from anywhere, so the strictest possible
	// CSP is appropriate. Override for HTML-serving services.
	ContentSecurityPolicy string

	// DisableContentSecurityPolicy lets callers explicitly omit the CSP
	// header (e.g. if it's set elsewhere in the chain). When false (the
	// default), ContentSecurityPolicy (or the default API CSP) is written.
	DisableContentSecurityPolicy bool
}

const (
	defaultHSTS              = "max-age=31536000; includeSubDomains"
	defaultContentTypeOpts   = "nosniff"
	defaultFrameOptions      = "DENY"
	defaultReferrerPolicy    = "strict-origin-when-cross-origin"
	defaultPermissionsPolicy = "camera=(), microphone=(), geolocation=(), interest-cohort=()"
	defaultXSSProtection     = "0"
	defaultAPICSP            = "default-src 'none'; frame-ancestors 'none'"
)

// SecurityHeaders returns a middleware that writes a set of conservative
// HTTP security headers on every response. The middleware is intended to be
// registered early in the chain (after Recovery, before Auth) so that the
// headers land even on error responses produced by downstream middleware.
func SecurityHeaders(config SecurityHeadersConfig) gin.HandlerFunc {
	hsts := config.StrictTransportSecurity
	if hsts == "" {
		hsts = defaultHSTS
	}

	contentTypeOpts := config.ContentTypeOptions
	if contentTypeOpts == "" {
		contentTypeOpts = defaultContentTypeOpts
	}

	frameOpts := config.FrameOptions
	if frameOpts == "" {
		frameOpts = defaultFrameOptions
	}

	referrerPolicy := config.ReferrerPolicy
	if referrerPolicy == "" {
		referrerPolicy = defaultReferrerPolicy
	}

	permissionsPolicy := config.PermissionsPolicy
	if permissionsPolicy == "" {
		permissionsPolicy = defaultPermissionsPolicy
	}

	xssProtection := config.XSSProtection
	if xssProtection == "" {
		xssProtection = defaultXSSProtection
	}

	csp := config.ContentSecurityPolicy
	if csp == "" {
		csp = defaultAPICSP
	}

	writeCSP := !config.DisableContentSecurityPolicy

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Strict-Transport-Security", hsts)
		h.Set("X-Content-Type-Options", contentTypeOpts)
		h.Set("X-Frame-Options", frameOpts)
		h.Set("Referrer-Policy", referrerPolicy)
		h.Set("Permissions-Policy", permissionsPolicy)
		h.Set("X-XSS-Protection", xssProtection)
		if writeCSP {
			h.Set("Content-Security-Policy", csp)
		}

		c.Next()
	}
}
