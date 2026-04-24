package api

import (
	"os"
	"time"

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
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "order-service",
		})
	})

	// Rate limiting
	router.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}))

	// JWT Auth config
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production-12345"
	}
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
