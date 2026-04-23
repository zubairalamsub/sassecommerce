package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditMiddleware logs all API requests for audit purposes
func AuditMiddleware(auditService service.AuditService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Read request body
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			// Restore the body for subsequent handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Create custom response writer to capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime).Milliseconds()

		// Determine action based on HTTP method
		action := getActionFromMethod(c.Request.Method)

		// Determine resource from path
		resource := getResourceFromPath(c.Request.URL.Path)

		// Get tenant ID from header or path
		tenantID := c.GetHeader("X-Tenant-Id")
		if tenantID == "" {
			tenantID = c.Param("id") // For tenant-specific operations
		}

		// Get user ID from context (set by auth middleware)
		userID, _ := c.Get("user_id")
		userIDStr := ""
		if userID != nil {
			userIDStr = userID.(string)
		}

		// Capture error message if any
		errorMessage := ""
		if len(c.Errors) > 0 {
			errorMessage = c.Errors.String()
		}

		// Create audit log entry
		auditReq := &models.CreateAuditLogRequest{
			TenantID:     tenantID,
			UserID:       userIDStr,
			Action:       action,
			Resource:     resource,
			ResourceID:   c.Param("id"),
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestBody:  requestBody,
			ResponseCode: c.Writer.Status(),
			ErrorMessage: errorMessage,
			Duration:     duration,
			Metadata: map[string]interface{}{
				"query_params": c.Request.URL.Query(),
				"headers": map[string]string{
					"content-type": c.Request.Header.Get("Content-Type"),
					"user-agent":   c.Request.UserAgent(),
				},
			},
		}

		// Don't block the response for audit logging
		go func() {
			if err := auditService.CreateAuditLog(c.Request.Context(), auditReq); err != nil {
				logger.WithError(err).Error("Failed to create audit log")
			}
		}()

		// Log to application logger as well
		logger.WithFields(logrus.Fields{
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status":        c.Writer.Status(),
			"duration_ms":   duration,
			"ip":            c.ClientIP(),
			"tenant_id":     tenantID,
			"user_id":       userIDStr,
			"action":        action,
			"resource":      resource,
		}).Info("API Request")
	}
}

func getActionFromMethod(method string) models.AuditAction {
	switch method {
	case "POST":
		return models.ActionCreate
	case "GET":
		return models.ActionRead
	case "PUT", "PATCH":
		return models.ActionUpdate
	case "DELETE":
		return models.ActionDelete
	default:
		return models.ActionRead
	}
}

func getResourceFromPath(path string) models.AuditResource {
	if contains(path, "/tenants") {
		if contains(path, "/config") {
			return models.ResourceTenantConfig
		}
		return models.ResourceTenant
	}
	if contains(path, "/users") {
		return models.ResourceUser
	}
	if contains(path, "/products") {
		return models.ResourceProduct
	}
	if contains(path, "/orders") {
		return models.ResourceOrder
	}
	return models.ResourceTenant
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
