package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireTenant returns a middleware that rejects requests whose JWT claims
// did not carry a tenant ID. It must be registered after AuthMiddleware.
// Handlers behind it can trust GetTenantID(c) to be non-empty and must never
// fall back to caller-controlled sources (query string, body) for the tenant.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetTenantID(c) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Tenant context required",
				"code":    "TENANT_REQUIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
