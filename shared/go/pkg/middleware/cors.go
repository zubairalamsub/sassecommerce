package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// envCORSOrigins is the environment variable name that, when set, provides a
// comma-separated list of allowed CORS origins. Takes precedence over the
// AllowOrigins field passed to HardenedCORS.
const envCORSOrigins = "CORS_ALLOWED_ORIGINS"

// envEnvironment is the environment variable used to gate stricter checks
// (e.g. refusing wildcard origins in production).
const envEnvironment = "ENVIRONMENT"

// defaultDevOrigins are the localhost origins enabled when running outside
// production and CORS_ALLOWED_ORIGINS is unset.
var defaultDevOrigins = []string{
	"http://localhost:3000",
	"http://localhost:3001",
}

// defaultAllowMethods is the conservative HTTP-verb set HardenedCORS allows
// when CORSConfig.AllowMethods is empty.
var defaultAllowMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

// defaultAllowHeaders are the headers we expect the browser to send. Includes
// our cross-cutting headers (X-Tenant-Id, X-Request-Id).
var defaultAllowHeaders = []string{
	"Origin",
	"Content-Type",
	"Authorization",
	"X-Tenant-Id",
	"X-Request-Id",
}

// defaultMaxAgeSeconds is 12 hours — long enough to amortize preflights, short
// enough to pick up CORS config changes within a working day.
const defaultMaxAgeSeconds = 12 * 60 * 60

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set CORS headers
		if len(config.AllowOrigins) > 0 {
			if config.AllowOrigins[0] == "*" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowOrigin := range config.AllowOrigins {
					if origin == allowOrigin {
						c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
		}

		if len(config.AllowMethods) > 0 {
			methods := ""
			for i, method := range config.AllowMethods {
				if i > 0 {
					methods += ", "
				}
				methods += method
			}
			c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		}

		if len(config.AllowHeaders) > 0 {
			headers := ""
			for i, header := range config.AllowHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		}

		if len(config.ExposeHeaders) > 0 {
			headers := ""
			for i, header := range config.ExposeHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			c.Writer.Header().Set("Access-Control-Expose-Headers", headers)
		}

		if config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if config.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// HardenedCORS returns a stricter CORS middleware suited for production use.
//
// Behavior vs. the legacy CORS function:
//   - Allowed origins are sourced from CORSConfig.AllowOrigins or, if that is
//     empty, from the CORS_ALLOWED_ORIGINS env var (comma-separated). In
//     development (ENVIRONMENT != "production"), an empty configuration falls
//     back to http://localhost:3000 and http://localhost:3001.
//   - Wildcard "*" origins are rejected in production. The Access-Control-
//     Allow-Origin header is only echoed back when the request Origin exactly
//     matches one of the allow-list entries.
//   - AllowCredentials is only honored when origins are explicit. Browsers
//     reject "Access-Control-Allow-Origin: *" combined with credentials, so
//     enforcing this server-side avoids silently-broken clients.
//   - Sensible defaults for AllowMethods, AllowHeaders, and MaxAge (12h).
func HardenedCORS(config CORSConfig) gin.HandlerFunc {
	origins := resolveAllowedOrigins(config.AllowOrigins)
	prod := strings.EqualFold(os.Getenv(envEnvironment), "production")

	// Strip wildcards in production and warn-by-action: anything that survives
	// the filter is treated as an explicit allow-list entry.
	if prod {
		filtered := origins[:0]
		for _, o := range origins {
			if o != "*" {
				filtered = append(filtered, o)
			}
		}
		origins = filtered
	}

	methods := config.AllowMethods
	if len(methods) == 0 {
		methods = defaultAllowMethods
	}

	headers := config.AllowHeaders
	if len(headers) == 0 {
		headers = defaultAllowHeaders
	}

	exposed := config.ExposeHeaders

	maxAge := config.MaxAge
	if maxAge <= 0 {
		maxAge = defaultMaxAgeSeconds
	}

	// Credentials are only safe with explicit, non-wildcard origins.
	allowCredentials := config.AllowCredentials && !containsWildcard(origins)

	methodsHeader := strings.Join(methods, ", ")
	headersHeader := strings.Join(headers, ", ")
	exposedHeader := strings.Join(exposed, ", ")
	maxAgeHeader := strconv.Itoa(maxAge)

	allowAny := containsWildcard(origins)

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Decide whether to echo the origin back.
		switch {
		case allowAny:
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Add("Vary", "Origin")
		case origin != "" && originAllowed(origin, origins):
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Add("Vary", "Origin")
		}

		// Common CORS response headers, set whether or not the origin matched
		// — browsers only act on Allow-Origin so leaking the rest is harmless
		// and saves a code path on preflight.
		c.Writer.Header().Set("Access-Control-Allow-Methods", methodsHeader)
		c.Writer.Header().Set("Access-Control-Allow-Headers", headersHeader)
		if exposedHeader != "" {
			c.Writer.Header().Set("Access-Control-Expose-Headers", exposedHeader)
		}
		c.Writer.Header().Set("Access-Control-Max-Age", maxAgeHeader)
		if allowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Preflight: terminate early with 204 when the origin matched (or when
		// wildcard is in play). For disallowed origins, fall through to the
		// route handler — gin will 404 on most OPTIONS paths, which is fine.
		if c.Request.Method == http.MethodOptions {
			if allowAny || (origin != "" && originAllowed(origin, origins)) {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// resolveAllowedOrigins merges the configured origins with the env-driven
// override and dev fallback. The env var wins if set.
func resolveAllowedOrigins(configured []string) []string {
	if env := os.Getenv(envCORSOrigins); env != "" {
		return splitAndTrim(env)
	}
	if len(configured) > 0 {
		return configured
	}
	if !strings.EqualFold(os.Getenv(envEnvironment), "production") {
		// Dev fallback: assume the Next.js storefront/admin run on standard
		// localhost ports.
		out := make([]string, len(defaultDevOrigins))
		copy(out, defaultDevOrigins)
		return out
	}
	return nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func containsWildcard(origins []string) bool {
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}

func originAllowed(origin string, allow []string) bool {
	for _, o := range allow {
		if o == origin {
			return true
		}
	}
	return false
}
