package api

import (
	"bytes"
	"io"
	"strings"
	"time"

	sharedconfig "github.com/ecommerce/shared/go/pkg/config"
	"github.com/ecommerce/shared/go/pkg/metrics"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Router sets up the HTTP routes
type Router struct {
	commandHandler *CommandHandler
	queryHandler   *QueryHandler
	logger         *zap.Logger
}

// NewRouter creates a new router
func NewRouter(
	commandHandler *CommandHandler,
	queryHandler *QueryHandler,
	logger *zap.Logger,
) *Router {
	return &Router{
		commandHandler: commandHandler,
		queryHandler:   queryHandler,
		logger:         logger,
	}
}

// Setup configures all routes
func (r *Router) Setup() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metrics.Middleware("order-service"))
	router.Use(r.requestResponseLogger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "order-service",
		})
	})

	// Prometheus metrics endpoint — registered before Auth/RateLimit so scrapes are not blocked.
	router.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Rate limiting
	router.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}))

	// JWT Auth config
	jwtSecret := sharedconfig.MustGetJWTSecret()
	authMw := sharedmiddleware.Auth(sharedmiddleware.AuthConfig{SecretKey: jwtSecret})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Guest-accessible routes (no auth required)
		orders := v1.Group("/orders")
		{
			orders.POST("", r.commandHandler.CreateOrder)   // Guest checkout
			orders.GET("/:id", r.queryHandler.GetOrder)     // Order tracking
		}

		// Authenticated order operations
		authOrders := v1.Group("/orders")
		authOrders.Use(authMw)
		{
			authOrders.POST("/:id/items", r.commandHandler.AddOrderItem)
			authOrders.DELETE("/:id/items/:itemId", r.commandHandler.RemoveOrderItem)
			authOrders.POST("/:id/confirm", r.commandHandler.ConfirmOrder)
			authOrders.POST("/:id/cancel", r.commandHandler.CancelOrder)
			authOrders.POST("/:id/ship", r.commandHandler.ShipOrder)
			authOrders.POST("/:id/deliver", r.commandHandler.DeliverOrder)
			// POS receipt email — publishes a ReceiptRequested event onto
			// Kafka for the notification-service consumer to render+send.
			authOrders.POST("/:id/send-receipt", r.commandHandler.SendReceipt)
		}

		// Authenticated query routes
		customers := v1.Group("/customers")
		customers.Use(authMw)
		{
			customers.GET("/:customerId/orders", r.queryHandler.GetOrdersByCustomer)
		}

		tenants := v1.Group("/tenants")
		tenants.Use(authMw)
		{
			tenants.GET("/:tenantId/orders", r.queryHandler.GetOrdersByTenant)
		}
	}

	return router
}

// responseBodyWriter captures response body for logging.
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

func (r *Router) requestResponseLogger() gin.HandlerFunc {
	const maxBodySize = 10 * 1024

	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"x-api-key":     true,
	}

	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		startTime := time.Now()

		// Capture request body
		var reqBody string
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize))
			if err == nil {
				reqBody = string(bodyBytes)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Capture headers
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if sensitiveHeaders[strings.ToLower(key)] {
				headers[key] = "[REDACTED]"
			} else {
				headers[key] = strings.Join(values, ", ")
			}
		}

		// Capture response body
		respWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			max:            maxBodySize,
		}
		c.Writer = respWriter

		c.Next()

		duration := time.Since(startTime)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("client_ip", c.ClientIP()),
			zap.Any("req_headers", headers),
			zap.String("req_body", reqBody),
			zap.String("resp_body", respWriter.body.String()),
			zap.Int("resp_size", c.Writer.Size()),
		}

		switch {
		case status >= 500:
			r.logger.Error("HTTP request completed with server error", fields...)
		case status >= 400:
			r.logger.Warn("HTTP request completed with client error", fields...)
		default:
			r.logger.Info("HTTP request completed", fields...)
		}
	}
}
