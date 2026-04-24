package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	// Rate is the number of requests allowed per window
	Rate int
	// Window is the time window for rate limiting
	Window time.Duration
	// KeyFunc extracts the rate limit key from the request.
	// Defaults to client IP if nil.
	KeyFunc func(c *gin.Context) string
	// Message is the error message returned when rate limited
	Message string
}

// DefaultRateLimitConfig returns a default config: 100 requests per minute per IP
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientWindow
	rate    int
	window  time.Duration
}

type clientWindow struct {
	count    int
	expireAt time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientWindow),
		rate:    rate,
		window:  window,
	}

	// Periodic cleanup of expired entries
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) allow(key string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cw, exists := rl.clients[key]

	if !exists || now.After(cw.expireAt) {
		rl.clients[key] = &clientWindow{
			count:    1,
			expireAt: now.Add(rl.window),
		}
		return true, rl.rate - 1
	}

	if cw.count >= rl.rate {
		return false, 0
	}

	cw.count++
	return true, rl.rate - cw.count
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, cw := range rl.clients {
		if now.After(cw.expireAt) {
			delete(rl.clients, key)
		}
	}
}

// RateLimit returns a middleware that limits request rate using a fixed window counter.
// It tracks requests per key (default: client IP) and rejects requests exceeding the limit.
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}
	if config.Message == "" {
		config.Message = "Too many requests, please try again later"
	}

	limiter := newRateLimiter(config.Rate, config.Window)

	return func(c *gin.Context) {
		var key string
		if config.KeyFunc != nil {
			key = config.KeyFunc(c)
		} else {
			key = c.ClientIP()
		}

		allowed, remaining := limiter.allow(key)
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   config.Message,
				"code":    "RATE_LIMITED",
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		c.Next()
	}
}

// RateLimitByTenant returns a rate limiter keyed by tenant ID.
// This is useful for per-tenant API quotas.
func RateLimitByTenant(rate int, window time.Duration) gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		Rate:   rate,
		Window: window,
		KeyFunc: func(c *gin.Context) string {
			tenantID := GetTenantID(c)
			if tenantID != "" {
				return "tenant:" + tenantID
			}
			return "ip:" + c.ClientIP()
		},
	})
}

// RateLimitByUser returns a rate limiter keyed by authenticated user ID.
// Falls back to IP-based limiting for unauthenticated requests.
func RateLimitByUser(rate int, window time.Duration) gin.HandlerFunc {
	return RateLimit(RateLimitConfig{
		Rate:   rate,
		Window: window,
		KeyFunc: func(c *gin.Context) string {
			userID := GetUserID(c)
			if userID != "" {
				return "user:" + userID
			}
			return "ip:" + c.ClientIP()
		},
	})
}

