package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware
func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set CORS headers
		if len(config.AllowOrigins) > 0 {
			if config.AllowOrigins[0] == "*" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowOrigin := range config.AllowOrigins {
					if origin == allowOrigin {
						c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
		}

		if len(config.AllowMethods) > 0 {
			methods := ""
			for i, method := range config.AllowMethods {
				if i > 0 {
					methods += ", "
				}
				methods += method
			}
			c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		}

		if len(config.AllowHeaders) > 0 {
			headers := ""
			for i, header := range config.AllowHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		}

		if len(config.ExposeHeaders) > 0 {
			headers := ""
			for i, header := range config.ExposeHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			c.Writer.Header().Set("Access-Control-Expose-Headers", headers)
		}

		if config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if config.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
