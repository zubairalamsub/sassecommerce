package middleware

import (
	"net/http"
	"strings"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// AuthorizationHeader is the HTTP header for authorization
	AuthorizationHeader = "Authorization"
	// UserIDKey is the context key for user ID
	UserIDKey = "user_id"
	// UserRoleKey is the context key for user role
	UserRoleKey = "user_role"
	// UserEmailKey is the context key for user email
	UserEmailKey = "user_email"
	// TenantIDKey is the context key for tenant ID
	TenantIDKey = "tenant_id"
)

// AuthMiddleware returns a middleware that validates JWT tokens
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header required",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization header format",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Verify token
		claims, err := authService.VerifyToken(c.Request.Context(), tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   err.Error(),
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(TenantIDKey, claims.TenantID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole returns a middleware that checks if the user has the required role
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(UserRoleKey)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "User role not found",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Invalid user role",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			if role == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Insufficient permissions",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(UserIDKey); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetTenantID extracts tenant ID from context
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get(TenantIDKey); exists {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) models.UserRole {
	if userRole, exists := c.Get(UserRoleKey); exists {
		if role, ok := userRole.(models.UserRole); ok {
			return role
		}
	}
	return ""
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) string {
	if userEmail, exists := c.Get(UserEmailKey); exists {
		if email, ok := userEmail.(string); ok {
			return email
		}
	}
	return ""
}
