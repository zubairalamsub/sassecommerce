package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// TenantIDKey is the key for tenant ID in context
	TenantIDKey = "tenant_id"
	// TenantIDHeader is the HTTP header for tenant ID
	TenantIDHeader = "X-Tenant-ID"
	// TenantSlugHeader is the HTTP header for tenant slug
	TenantSlugHeader = "X-Tenant-Slug"
)

// TenantConfig holds tenant middleware configuration
type TenantConfig struct {
	// Required indicates if tenant ID is required
	Required bool
	// AllowHeader allows tenant ID to be passed via header
	AllowHeader bool
	// AllowSubdomain allows tenant to be identified by subdomain
	AllowSubdomain bool
	// AllowPath allows tenant to be identified by path parameter
	AllowPath bool
	// PathParam is the name of the path parameter for tenant ID
	PathParam string
}

// DefaultTenantConfig returns default tenant configuration
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Required:       true,
		AllowHeader:    true,
		AllowSubdomain: true,
		AllowPath:      true,
		PathParam:      "tenant_id",
	}
}

// Tenant returns a middleware that extracts tenant information
func Tenant(config TenantConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string
		var tenantSlug string

		// 1. Try to get from header
		if config.AllowHeader {
			tenantID = c.GetHeader(TenantIDHeader)
			tenantSlug = c.GetHeader(TenantSlugHeader)
		}

		// 2. Try to get from subdomain
		if tenantID == "" && config.AllowSubdomain {
			host := c.Request.Host
			parts := strings.Split(host, ".")
			if len(parts) > 2 {
				// Extract subdomain (e.g., "tenant1.api.example.com" -> "tenant1")
				tenantSlug = parts[0]
			}
		}

		// 3. Try to get from path parameter
		if tenantID == "" && config.AllowPath && config.PathParam != "" {
			tenantID = c.Param(config.PathParam)
		}

		// Check if tenant ID is required
		if config.Required && tenantID == "" && tenantSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tenant identification required",
				"details": "Provide tenant ID via X-Tenant-ID header, subdomain, or path parameter",
			})
			c.Abort()
			return
		}

		// Set tenant information in context
		if tenantID != "" {
			c.Set(TenantIDKey, tenantID)
		}
		if tenantSlug != "" {
			c.Set("tenant_slug", tenantSlug)
		}

		c.Next()
	}
}

// GetTenantID extracts tenant ID from gin context
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get(TenantIDKey); exists {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return ""
}

// GetTenantSlug extracts tenant slug from gin context
func GetTenantSlug(c *gin.Context) string {
	if tenantSlug, exists := c.Get("tenant_slug"); exists {
		if slug, ok := tenantSlug.(string); ok {
			return slug
		}
	}
	return ""
}
