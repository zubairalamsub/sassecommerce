package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler returns a middleware that handles errors
func ErrorHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error
			requestID := GetRequestID(c)
			tenantID := GetTenantID(c)

			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"tenant_id":  tenantID,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"error":      err.Error(),
			}).Error("Request error")

			// Return error response if not already sent
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal server error",
					"request_id": requestID,
				})
			}
		}
	}
}

// Recovery returns a middleware that recovers from panics
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				requestID := GetRequestID(c)
				tenantID := GetTenantID(c)

				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"tenant_id":  tenantID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"panic":      err,
				}).Error("Panic recovered")

				// Return error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal server error",
					"request_id": requestID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
