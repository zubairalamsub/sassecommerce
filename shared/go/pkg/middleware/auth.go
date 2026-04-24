package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// AuthUserIDKey is the context key for user ID
	AuthUserIDKey = "user_id"
	// AuthUserRoleKey is the context key for user role
	AuthUserRoleKey = "user_role"
	// AuthUserEmailKey is the context key for user email
	AuthUserEmailKey = "user_email"
)

// JWTClaims represents the JWT token claims used across all services
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthConfig holds JWT authentication configuration
type AuthConfig struct {
	// SecretKey is the JWT signing key
	SecretKey string
}

// Auth returns a middleware that validates JWT tokens and sets user context.
// It extracts user_id, tenant_id, email, and role from the token claims.
func Auth(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
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

		// Parse and validate token
		claims, err := validateToken(parts[1], config.SecretKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   err.Error(),
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(AuthUserIDKey, claims.UserID)
		c.Set(TenantIDKey, claims.TenantID)
		c.Set(AuthUserEmailKey, claims.Email)
		c.Set(AuthUserRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole returns a middleware that checks if the user has one of the required roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(AuthUserRoleKey)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "User role not found",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Invalid user role",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		for _, required := range roles {
			if role == required {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Insufficient permissions",
			"code":    "FORBIDDEN",
		})
		c.Abort()
	}
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) string {
	if v, exists := c.Get(AuthUserIDKey); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserEmail extracts user email from gin context
func GetUserEmail(c *gin.Context) string {
	if v, exists := c.Get(AuthUserEmailKey); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserRole extracts user role from gin context
func GetUserRole(c *gin.Context) string {
	if v, exists := c.Get(AuthUserRoleKey); exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func validateToken(tokenString, secretKey string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}
