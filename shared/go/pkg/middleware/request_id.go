package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey is the key for request ID in context
	RequestIDKey = "request_id"
	// RequestIDHeader is the HTTP header for request ID
	RequestIDHeader = "X-Request-ID"
)

// RequestID returns a middleware that adds a request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new request ID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID extracts request ID from gin context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
