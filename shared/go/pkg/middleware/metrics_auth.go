package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const envMetricsToken = "METRICS_TOKEN"

// MetricsAuth guards the Prometheus /metrics endpoint with a static bearer
// token read from METRICS_TOKEN at startup.
//
//   - When METRICS_TOKEN is set, scrapes must send
//     "Authorization: Bearer <token>" (configure `authorization` in the
//     Prometheus scrape config).
//   - When unset in production (ENVIRONMENT=production), the endpoint is
//     disabled with 404 rather than served unauthenticated.
//   - When unset outside production, the endpoint stays open for local dev
//     and private-cluster scrapes.
func MetricsAuth() gin.HandlerFunc {
	token := os.Getenv(envMetricsToken)
	prod := strings.EqualFold(os.Getenv(envEnvironment), "production")

	return func(c *gin.Context) {
		if token == "" {
			if prod {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
			c.Next()
			return
		}

		expected := "Bearer " + token
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte(expected)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
