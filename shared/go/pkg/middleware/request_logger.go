package middleware

import (
	"bytes"
	"encoding/json"
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
	// SensitiveJSONFields are JSON keys (case-insensitive, matched at any
	// nesting depth) whose values are redacted from logged request and
	// response bodies. Defaults to DefaultSensitiveJSONFields.
	SensitiveJSONFields []string
}

// DefaultSensitiveJSONFields covers credentials and secrets that must never
// reach log storage: passwords on login/register/change-password, issued and
// refresh tokens, password-reset and email-verification tokens, TOTP/2FA
// codes, and payment card data.
var DefaultSensitiveJSONFields = []string{
	"password", "old_password", "new_password", "current_password",
	"confirm_password", "password_confirmation",
	"token", "access_token", "refresh_token", "id_token", "challenge_token",
	"reset_token", "verification_token",
	"secret", "client_secret", "api_key",
	"otp", "code", "backup_code", "backup_codes", "totp_code", "two_factor_code",
	"card_number", "cvv", "pin",
}

// SensitiveFieldSet builds the lowercase lookup set RedactJSONBody expects.
func SensitiveFieldSet(keys []string) map[string]bool {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[strings.ToLower(k)] = true
	}
	return set
}

// RedactJSONBody returns body with the values of sensitive keys replaced by
// "[REDACTED]" at any nesting depth. Bodies that do not parse as JSON are
// omitted entirely rather than logged verbatim — a truncated or non-JSON
// payload could still contain credentials.
func RedactJSONBody(body string, sensitive map[string]bool) string {
	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "[non-JSON body omitted]"
	}
	redactNode(data, sensitive)
	out, err := json.Marshal(data)
	if err != nil {
		return "[unloggable body omitted]"
	}
	return string(out)
}

func redactNode(v any, sensitive map[string]bool) {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if sensitive[strings.ToLower(k)] {
				node[k] = "[REDACTED]"
			} else {
				redactNode(val, sensitive)
			}
		}
	case []any:
		for _, item := range node {
			redactNode(item, sensitive)
		}
	}
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
	if config.SensitiveJSONFields == nil {
		config.SensitiveJSONFields = DefaultSensitiveJSONFields
	}
	sensitiveFields := SensitiveFieldSet(config.SensitiveJSONFields)

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
			fields["req_body"] = RedactJSONBody(requestBody, sensitiveFields)
		}

		if config.LogResponseBody && respBodyWriter != nil && respBodyWriter.body.Len() > 0 {
			fields["resp_body"] = RedactJSONBody(respBodyWriter.body.String(), sensitiveFields)
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
