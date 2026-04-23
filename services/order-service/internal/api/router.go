package api

import (
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

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Order commands (write operations)
		orders := v1.Group("/orders")
		{
			orders.POST("", r.commandHandler.CreateOrder)
			orders.POST("/:id/items", r.commandHandler.AddOrderItem)
			orders.DELETE("/:id/items/:itemId", r.commandHandler.RemoveOrderItem)
			orders.POST("/:id/confirm", r.commandHandler.ConfirmOrder)
			orders.POST("/:id/cancel", r.commandHandler.CancelOrder)
			orders.POST("/:id/ship", r.commandHandler.ShipOrder)
			orders.POST("/:id/deliver", r.commandHandler.DeliverOrder)

			// Order queries (read operations)
			orders.GET("/:id", r.queryHandler.GetOrder)
		}

		// Query by customer
		customers := v1.Group("/customers")
		{
			customers.GET("/:customerId/orders", r.queryHandler.GetOrdersByCustomer)
		}

		// Query by tenant
		tenants := v1.Group("/tenants")
		{
			tenants.GET("/:tenantId/orders", r.queryHandler.GetOrdersByTenant)
		}
	}

	return router
}
