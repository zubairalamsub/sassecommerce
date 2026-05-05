package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLoggerConfig configures the request/response logging middleware.
type RequestLoggerConfig struct {
	// Logger is the logrus logger instance.
	Logger *logrus.Logger
	// MaxBodySize is the maximum request/response body size to log (in bytes).
	// Bodies larger than this are truncated. Default: 10KB.
	MaxBodySize int
	// SkipPaths are URL paths that should not be logged (e.g., health checks).
	SkipPaths []string
	// LogRequestBody enables logging of the request body. Default: true.
	LogRequestBody bool
	// LogResponseBody enables logging of the response body. Default: true.
	LogResponseBody bool
	// SensitiveHeaders are header names whose values will be redacted.
	SensitiveHeaders []string
}

const defaultMaxBodySize = 10 * 1024 // 10 KB

// responseBodyWriter wraps gin.ResponseWriter to capture the response body.
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
	max  int
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body.Len() < w.max {
		remaining := w.max - w.body.Len()
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

// RequestLogger returns a middleware that logs full request and response details
// including headers, body, status, and timing.
func RequestLogger(config RequestLoggerConfig) gin.HandlerFunc {
	if config.MaxBodySize <= 0 {
		config.MaxBodySize = defaultMaxBodySize
	}
	if config.SensitiveHeaders == nil {
		config.SensitiveHeaders = []string{"authorization", "cookie", "x-api-key"}
	}

	skipSet := make(map[string]bool, len(config.SkipPaths))
	for _, p := range config.SkipPaths {
		skipSet[p] = true
	}

	sensitiveSet := make(map[string]bool, len(config.SensitiveHeaders))
	for _, h := range config.SensitiveHeaders {
		sensitiveSet[strings.ToLower(h)] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip configured paths
		if skipSet[path] {
			c.Next()
			return
		}

		startTime := time.Now()
		requestID := GetRequestID(c)

		// Capture request body
		var requestBody string
		if config.LogRequestBody && c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(config.MaxBodySize)))
			if err == nil {
				requestBody = string(bodyBytes)
				// Restore body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Capture request headers (redact sensitive ones)
		reqHeaders := make(map[string]string)
		for key, values := range c.Request.Header {
			if sensitiveSet[strings.ToLower(key)] {
				reqHeaders[key] = "[REDACTED]"
			} else {
				reqHeaders[key] = strings.Join(values, ", ")
			}
		}

		// Wrap response writer to capture body
		var respBodyWriter *responseBodyWriter
		if config.LogResponseBody {
			respBodyWriter = &responseBodyWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
				max:            config.MaxBodySize,
			}
			c.Writer = respBodyWriter
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)
		status := c.Writer.Status()

		// Build log fields
		fields := logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       path,
			"query":      c.Request.URL.RawQuery,
			"status":     status,
			"duration_ms": duration.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"tenant_id":  c.GetString("tenant_id"),
			"user_id":    c.GetString("user_id"),
			"req_headers": reqHeaders,
		}

		if config.LogRequestBody && requestBody != "" {
			if len(requestBody) >= config.MaxBodySize {
				requestBody = requestBody[:config.MaxBodySize] + "...[truncated]"
			}
			fields["req_body"] = requestBody
		}

		if config.LogResponseBody && respBodyWriter != nil {
			respBody := respBodyWriter.body.String()
			if len(respBody) >= config.MaxBodySize {
				respBody = respBody[:config.MaxBodySize] + "...[truncated]"
			}
			fields["resp_body"] = respBody
		}

		fields["resp_size"] = c.Writer.Size()

		// Log at appropriate level based on status code
		entry := config.Logger.WithFields(fields)
		switch {
		case status >= 500:
			entry.Error("HTTP request completed with server error")
		case status >= 400:
			entry.Warn("HTTP request completed with client error")
		default:
			entry.Info("HTTP request completed")
		}
	}
}
